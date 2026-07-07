# Data Model: Dev Container Configuration

This feature introduces developer environment configuration rather than application data. The "model" is the schema of `.devcontainer/devcontainer.json` and the environment contract it establishes inside the container.

## devcontainer.json Field Schema

| Field | Type | Value | Purpose |
|-------|------|-------|---------|
| `name` | string | `"Whale Runner Dev"` | Display name shown in VS Code status bar |
| `image` | string | `mcr.microsoft.com/devcontainers/base:bookworm` | Debian 12 base with VS Code devcontainer integration |
| `features["ghcr.io/devcontainers/features/go:1"]` | object | `{"version": "1.25"}` | Go 1.25 toolchain — version must match `app/go.mod` |
| `features["ghcr.io/devcontainers/features/docker-outside-of-docker:1"]` | object | `{}` | Mounts host `/var/run/docker.sock`; installs Docker CLI |
| `workspaceMount` | string | `source=${localWorkspaceFolder},target=${localWorkspaceFolder},type=bind` | Path-matching bind mount — required for Testcontainers |
| `workspaceFolder` | string | `${localWorkspaceFolder}` | Working directory inside container — must match mount target |
| `containerEnv.TESTCONTAINERS_HOST_OVERRIDE` | string | `host.docker.internal` | Enables test processes to reach sibling containers on Docker Desktop |
| `customizations.vscode.extensions` | string[] | `["golang.go", "ms-azuretools.vscode-docker"]` | Mandatory extensions installed on container creation |
| `customizations.vscode.settings` | object | See VS Code Settings below | Editor behaviour baked into the workspace |
| `postCreateCommand` | string | `cd app && go mod download` | Pre-download Go module dependencies after container creation |
| `forwardPorts` | int[] | `[]` (empty) | Intentionally empty — with DooD, compose containers bind ports directly on the host daemon; those ports are already accessible at `localhost:PORT` on the host without VS Code tunneling (see research.md §Decision 6) |
| `portsAttributes` | object | `{}` (empty) | Not used — ports are not forwarded through the dev container |

## VS Code Settings Schema

| Setting | Value | Purpose |
|---------|-------|---------|
| `editor.formatOnSave` | `true` | Format all files on save |
| `[go].editor.defaultFormatter` | `"golang.go"` | Use official Go extension for formatting (goimports) |
| `[go].editor.formatOnSave` | `true` | Explicit per-language override |
| `go.toolsManagement.autoUpdate` | `true` | Keep gopls and related tools current |
| `go.testFlags` | `["-v", "-count=1"]` | Verbose test output; disable test result caching |

## Environment Variables Established Inside Container

| Variable | Value | Set By | Purpose |
|----------|-------|--------|---------|
| `TESTCONTAINERS_HOST_OVERRIDE` | `host.docker.internal` | `devcontainer.json` containerEnv | Testcontainers host resolution for Docker Desktop DooD |
| `GOPATH` | `/go` | Go devcontainer feature | Go module and binary cache |
| `GOROOT` | `/usr/local/go` | Go devcontainer feature | Go toolchain installation root |
| `PATH` | `...:/usr/local/go/bin:/go/bin` | Go devcontainer feature | Go toolchain and installed binaries on PATH |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` | DooD feature | Docker CLI socket path (default; set explicitly by DooD feature) |

## Port Access with DooD

VS Code's `forwardPorts` tunnels ports that are listening *inside* the dev container to the host browser. With DooD, `docker compose up` binds ports on the **host Docker daemon's** network — not inside the dev container. VS Code's tunnel would forward an empty socket, serving nothing.

Compose ports are therefore accessed directly on the host, without any VS Code forwarding:

| Port | Service | Access |
|------|---------|--------|
| 8080 | App (ungated) | `http://localhost:8080` on host (direct, no tunnel) |
| 8081 | App (gated) | `http://localhost:8081` on host (direct, no tunnel) |
| 4040 | ngrok admin UI | `http://localhost:4040` on host (direct, no tunnel) |

## File Layout

```text
Repository root
└── .devcontainer/
    └── devcontainer.json    # The only new file introduced by this feature
```

### What does NOT change

- `app/` — Go source code, tests, go.mod, Dockerfile: no modifications
- `docker-compose.yml` — no new services, no modifications
- `frontend/` — no modifications
- `bin/` — no modifications

## Invariants

1. `workspaceMount.target` MUST equal `workspaceFolder` MUST equal `${localWorkspaceFolder}` — if these diverge, Testcontainers bind-mount forwarding silently breaks.
2. The Go version in `features["ghcr.io/devcontainers/features/go:1"]["version"]` MUST stay in sync with the `go` directive in `app/go.mod`. When `go.mod` is updated to a new Go version, `devcontainer.json` must be updated in the same commit.
3. No service defined in `docker-compose.yml` may be redefined inside `.devcontainer/` — the dev container is not a compose service (Constitution Principle II).
