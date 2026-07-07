# Research: Dev Container Support

Source material: `compose-and-test-containers.md` (project research document compiled prior to planning).

---

## Decision 1: Docker Access Pattern (DooD vs DinD vs TCC)

**Decision**: Docker-outside-of-Docker (DooD) — mount the host's `/var/run/docker.sock` into the dev container.

**Rationale**:
- Testcontainers' preferred and explicitly recommended pattern, which they call the "Docker wormhole"
- Lightest option: reuses the host image layer cache; no separate privileged daemon; no additional resource overhead
- Test-spawned containers are siblings on the host daemon, visible in `docker ps` on the host — predictable and debuggable
- DooD is already the correct choice for a project that requires Docker Desktop as a host prerequisite (which this project already does)

**Alternatives considered**:

| Option | Verdict | Reason rejected |
|--------|---------|-----------------|
| Docker-in-Docker (DinD) | Rejected (fallback only) | Requires privileged container; heavier resource footprint; no shared image cache; Testcontainers calls it "an instrument of last resort" |
| Testcontainers Cloud (TCC) | Rejected | Paid SaaS; requires a service-account token; unsuitable for a public sample repo |

**DinD as documented fallback**: DinD via `ghcr.io/devcontainers/features/docker-in-docker:2` (with `"moby": true`) remains available if a developer's host environment prohibits socket exposure. This should be noted in developer documentation but is not the default configuration.

**Implementation mechanism**: `ghcr.io/devcontainers/features/docker-outside-of-docker:1` devcontainer feature, plus two required companion settings (Decisions 3 and 4 below).

---

## Decision 2: Dev Container Base Image

**Decision**: `mcr.microsoft.com/devcontainers/base:bookworm` (Debian 12) with Go installed via the `ghcr.io/devcontainers/features/go:1` feature, pinned to `"version": "1.25"`.

**Rationale**:
- Pinning the Go version in the feature config (`"version": "1.25"`) is more reliable than relying on a prebuilt `mcr.microsoft.com/devcontainers/go:1.25` image tag — the exact minor version may not be published as a named tag
- The `base:bookworm` image provides all VS Code devcontainer integration (non-root user, sudo, git, common utilities) without bundling a Go version that might not match the project's `go.mod`
- Debian 12 (Bookworm) is current, LTS, and widely supported by Go tooling

**Alternatives considered**:

| Option | Verdict | Reason rejected |
|--------|---------|-----------------|
| `mcr.microsoft.com/devcontainers/go:1` | Rejected | Pins to Go's latest stable, not 1.25; version drift risk |
| Custom Dockerfile on `dhi.io/golang:1.25-alpine-dev` | Rejected | Production image lacks VS Code devcontainer integration (non-root user, sudo, extension host support); significant ongoing maintenance burden |
| `golang:1.25-bookworm` | Rejected | Plain Go image; missing all devcontainer integration |

**Go version source of truth**: `app/go.mod` declares `go 1.25.0`. The devcontainer feature must be pinned to match.

---

## Decision 3: Workspace Path Matching

**Decision**: Set `workspaceMount` to bind-mount `${localWorkspaceFolder}` at the same target path inside the container, and set `workspaceFolder` to `${localWorkspaceFolder}`.

**Rationale**: Testcontainers forwards bind-mount paths to the Docker daemon using the path string as seen inside the running test process. If the container workspace is at `/workspaces/project` but the host path is `/Users/kris/repos/buk/project`, any `testcontainers.WithBindFiles(...)` call will ask the host daemon to mount `/workspaces/project/fixtures` — which either doesn't exist on the host or points at the wrong directory.

The fix is simple: make the in-container path identical to the host path. VS Code's `${localWorkspaceFolder}` substitution variable resolves to the actual host path at container build time.

**Configuration**:
```json
"workspaceMount": "source=${localWorkspaceFolder},target=${localWorkspaceFolder},type=bind",
"workspaceFolder": "${localWorkspaceFolder}"
```

**Platform notes**: Works correctly on macOS (`/Users/...`), Linux (`/home/...`), and Docker Desktop on Windows via WSL2 (paths are translated by Docker Desktop's filesystem proxy).

---

## Decision 4: TESTCONTAINERS_HOST_OVERRIDE

**Decision**: Set `TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal` as a permanent `containerEnv` entry.

**Rationale**: On Docker Desktop (macOS, Windows, WSL2), a process inside the dev container cannot reach `localhost` to connect to a service port exposed by a container spawned via the host daemon. The spawned container is a sibling — its exposed ports are bound on the host's loopback, not the dev container's loopback. `host.docker.internal` resolves to the host's IP from within any container, making the sibling's ports reachable.

Without this variable, Testcontainers-based tests produce "connection refused" against `localhost:<port>` even though the spawned container started successfully.

**Safety**: The variable is a no-op on Linux hosts where `localhost` already routes correctly (the override just substitutes one reachable address for another). Setting it unconditionally avoids a per-developer configuration step.

**Common failure mode**: Ryuk reaper cleanup errors in test output are typically a symptom of this connectivity issue (Ryuk can't reach the Docker daemon via `localhost`), not a Ryuk bug. Fix the host override first.

---

## Decision 5: VS Code Extensions and Settings

**Decision**: Include `golang.go` (Go official extension) and `ms-azuretools.vscode-docker` (Docker extension) as mandatory devcontainer extensions. Configure auto-format on save and verbose test output.

**Rationale**: These are the two extensions a developer working on this project will need immediately. The Go extension provides gopls (language server, inline errors, autocomplete, go-to-definition, test runner). The Docker extension surfaces the running compose stack and containers without leaving VS Code. Additional extensions (GitLens, etc.) are left to individual developer preference; over-bundling extensions slows container startup.

**VS Code settings** (baked into devcontainer.json `customizations.vscode.settings`):
- `editor.formatOnSave: true` + `[go].editor.defaultFormatter: "golang.go"` — auto-format eliminates manual `gofmt` runs
- `go.testFlags: ["-v", "-count=1"]` — verbose output; `-count=1` prevents test caching so Testcontainers tests always run against real containers

---

## Decision 6: forwardPorts is Empty (DooD Port Access)

**Decision**: `forwardPorts` is set to `[]` (empty) in `devcontainer.json`. Compose ports are accessed directly on the host, not via VS Code tunneling.

**Rationale**: VS Code's `forwardPorts` works by listening for ports that are bound on *the dev container's* localhost and tunneling them to the host browser. With DooD, `docker compose up` runs against the host Docker daemon — the app and Redis containers bind their ports on the **host's** network interface (e.g., `0.0.0.0:8080` on the host), not inside the dev container. VS Code's tunnel therefore forwards an empty socket, serving nothing.

Populated `forwardPorts` entries for compose ports actively prevent access: VS Code occupies the host-side port for its tunnel before the browser can reach the compose-bound port directly.

**Correct access pattern with DooD**: open `http://localhost:8080` (and 8081, 4040) directly in the host browser. The compose containers' port bindings are already on the host daemon's network and require no intermediary.

**Discovery**: This was identified during live validation when `localhost:8080` was unreachable in the host browser despite the compose stack running. Removing the ports from `forwardPorts` resolved it immediately.

---

## Resolved: No Contract Artifacts Needed

This feature delivers developer tooling configuration, not an externally-facing API or service interface. There are no HTTP endpoints, CLI schemas, or library APIs introduced by this feature. The `contracts/` directory is omitted.
