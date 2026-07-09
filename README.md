# Crossy Whale

![Crossy Whale gameplay](container-obstacles.gif)

A browser-based Crossy Road clone starring the Docker whale, built to show how Docker's core technologies work together in a real application.

## What you will learn

Working through this repo you'll encounter six Docker technologies in a real codebase:

1. **Docker Compose**: define and run a multi-service application with a single command
2. **Dockerfile & multi-stage builds**: compile a Go binary and produce a lean, production-ready image
3. **Docker Hardened Images (DHI)**: use security-hardened base images from `dhi.io` to reduce your attack surface
4. **Testcontainers**: run real Docker containers as part of your Go test suite, eliminating mock/production drift
5. **Dev Containers**: define your entire development environment as code, so any developer gets the same toolchain instantly
6. **Kubernetes**: deploy the running application to a local Kubernetes cluster using a Helm chart

---

## Run the app

**Prerequisites:**

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (or Docker Engine + Compose v2). Nothing else to install.
- A free [Docker account](https://hub.docker.com/signup) and a one-time login to the DHI registry. The app and Redis use images from `dhi.io`, a separate container registry from Docker Hub that requires authentication even for free Community images:

  ```bash
  docker login dhi.io        # use your free Docker ID; DHI Community images are free to pull
  ```

Start everything:

```bash
docker compose up -d
```

Open <http://localhost/>. You'll see a landing page linking to `/play` (the game), `/host` (the QR code presenter view), and `/leaderboard` (the wall display). One command started five services (Redis, the Go app, two microservices, nginx, and optionally ngrok) and wired them together.

Stop with:

```bash
docker compose down
```

That was all Docker Compose. The next section shows how.

---

## Understanding Docker Compose

Open [`docker-compose.yml`](docker-compose.yml).

**What Docker Compose does**: a single `docker-compose.yml` file declares every service your application needs: what image to use, what ports to expose, what environment variables to set, and how services depend on each other. `docker compose up` reads that file and starts everything.

**The five services**: this project defines `redis`, `app`, `commits-service`, `scores-service`, `nginx`, and `ngrok` under the top-level `services:` key:

```yaml
services:
  redis:            # in-memory data store (leaderboard + QR gate state)
  app:              # Go backend - built from source
  commits-service:  # microservice: serves recent git commits as JSON + SSE
  scores-service:   # microservice: serves leaderboard standings as JSON + SSE
  nginx:            # reverse proxy + static file server (single public ingress)
  ngrok:            # public tunnel (optional, --profile public)
```

**How services discover each other**: the Go app needs to know where Redis is. It reads that from the `REDIS_ADDR` environment variable. Look at how that variable is set in the `app` service:

```yaml
environment:
  - REDIS_ADDR=redis:6379
```

The hostname `redis` is not something you configured anywhere. It's the name of the `redis` service in this file. Docker Compose automatically provides DNS so every service can reach any other by its service name. The app just reads `REDIS_ADDR`, and Docker resolves `redis` to the right container at runtime. No hard-coded IP addresses needed.

**`expose:` vs `ports:`**: these two keys control who can reach a port. Redis only needs to be reachable within the stack — no reason to expose it to your laptop. nginx is the single public entry point, so it's the only service that needs a port published to the host. The Go app still publishes port 8080 for direct developer access, but regular users reach everything through nginx on port 80.

```yaml
redis:
  expose:
    - "6379"    # reachable by other services on the Compose network, NOT the host machine

nginx:
  ports:
    - "80:80"   # the single public entry point - open http://localhost in a browser

app:
  ports:
    - "8080:8080"   # developer access only; nginx proxies to this internally
```

**Environment variables as configuration**: the app has no config files. Every runtime setting is injected via environment variables. The `${VAR:-default}` syntax means "use this value if the variable isn't set in `.env` or the shell", so the app works out of the box without any setup:

```yaml
environment:
  - GRANT_COOKIE_SECRET=${GRANT_COOKIE_SECRET:-dev-only-change-me}  # signs QR access grant cookies
  - QR_WINDOW_TTL=${QR_WINDOW_TTL:-15m}                             # how long a QR code stays valid
```

**Profiles for optional services**: for local development you don't need a public URL. Just open `localhost` directly. ngrok is only needed when you want to share the game over the internet (e.g. so booth attendees can scan a QR code and play on their phones). The `profiles: [public]` key keeps ngrok out of the default `docker compose up` and only starts it when you explicitly opt in:

```yaml
ngrok:
  profiles:
    - public    # only starts with: docker compose --profile public up
```

---

## Understanding the Dockerfile

Open [`app/Dockerfile`](app/Dockerfile).

**Multi-stage builds**: the Dockerfile has two stages. The first (`AS build`) uses a full Go toolchain image to compile the binary. The second (final) stage starts from a minimal base image and copies in only the compiled binary and the frontend assets. The Go compiler, source code, and all build tools are left behind. They never appear in the final image. The result is a smaller image that pulls faster on deployment, and a more secure one because fewer packages means fewer CVEs to worry about.

```dockerfile
FROM dhi.io/golang:1.25-alpine-dev AS build   # stage 1: full Go toolchain
WORKDIR /src
COPY app/go.mod app/go.sum ./
RUN go mod download
COPY app/ .
RUN CGO_ENABLED=0 go build -o /out/app .      # compile - output goes to /out/app

FROM dhi.io/static:20260611-alpine3.24         # stage 2: near-empty runtime image
COPY --from=build /out/app /app               # copy only the compiled binary
COPY frontend/game /frontend                  # copy only the frontend assets
ENTRYPOINT ["/app"]
```

**Docker Hardened Images**: both `FROM` lines pull from `dhi.io/`, Docker's registry of hardened base images. Compared to their Docker Hub equivalents, DHI images ship fewer packages, use known-good package versions, and are regularly updated to patch vulnerabilities. Fewer packages means fewer potential CVEs in the running container.

`dhi.io/static` is a near-empty image designed for statically compiled binaries. Go compiles to a single self-contained binary with no external runtime dependencies: no interpreter, no libc needed. The production image ends up containing only the app binary and the frontend asset files.

---

## Understanding Testcontainers

Open [`app/internal/gate/window_test.go`](app/internal/gate/window_test.go) and find the `newTestRedisStore` function.

**What Testcontainers does**: Testcontainers-go is a Go library that starts real Docker containers as part of a test. Each call to `tcredis.Run()` pulls the Redis image, starts a container, maps a random port, and returns a handle. When the test ends, the container is stopped and removed automatically.

**Why a real container instead of a mock**: a mock Redis client can be programmed to return the right answers, but it can't reproduce actual Redis behaviour: TTL expiry timing, stream semantics, command argument expectations. Tests that passed against a mock have broken in production when Redis behaviour differed in subtle ways. A real container gives you the same confidence as running against production, and it only costs a few extra seconds per test run.

For a second example, open [`app/internal/leaderboard/store_test.go`](app/internal/leaderboard/store_test.go). The same pattern tests the leaderboard's Redis Stream operations.

For the interface design that makes this possible (tests use a fast in-memory fake; only the Redis implementation tests use Testcontainers), look at [`app/internal/gate/window.go`](app/internal/gate/window.go) and the `WindowStore` interface comment.

---

## Developing inside a Dev Container

**What a dev container is**: the classic developer problem is "it works on my machine". Two developers install Go, but one has 1.23 and the other has 1.25, and tests pass differently on each. A dev container solves this by defining the development environment as code: Go version, editor extensions, and configuration all live in `.devcontainer/devcontainer.json`, so every developer (and every CI run) gets exactly the same environment, regardless of what's installed on the host.

Open [`.devcontainer/devcontainer.json`](.devcontainer/devcontainer.json) to see how.

**Pinning the Go version**: the `features` key pulls in a pre-built Go toolchain at a specific version. Every developer who opens this project in a dev container gets exactly Go 1.25, no matter what they have installed locally.

```json
"features": {
  "ghcr.io/devcontainers/features/go:1": {
    "version": "1.25"
  }
}
```

Go also declares the minimum required version in `go.mod`, so the module itself refuses to build on an older toolchain:

```
go 1.25.0
```

These two work together: `go.mod` says what the code needs; `devcontainer.json` makes sure that version is what you get.

**Docker access from inside the container**: the `docker-outside-of-docker` feature mounts the host's Docker socket into the container. Docker commands you run inside the container talk to the same Docker engine running on your laptop, so `docker compose up` starts containers on the host, not inside the dev container.

```json
"features": {
  "ghcr.io/devcontainers/features/docker-outside-of-docker:1": {}
}
```

**Editor configuration**: VS Code extensions and settings are declared in `devcontainer.json`, so every developer gets the same editor setup automatically — no manual extension installs.

```json
"customizations": {
  "vscode": {
    "extensions": [
      "golang.go",
      "ms-azuretools.vscode-docker"
    ],
    "settings": {
      "editor.formatOnSave": true
    }
  }
}
```

**Prerequisites**: Docker Desktop running on the host, VS Code, Dev Containers extension installed.

**Open in one step**: open the repo folder in VS Code, then when prompted click **Reopen in Container** (or use the Command Palette: `Dev Containers: Reopen in Container`). The first build takes a few minutes; subsequent opens are instant.

**Full validation walkthrough**: [`specs/006-dev-container-support/quickstart.md`](specs/006-dev-container-support/quickstart.md)

---

## Deploying to Kubernetes

Kubernetes is a container orchestration system. It runs containers across one or more machines and handles restarts, scaling, and networking. This project includes a Helm chart in `k8s/`. Helm is a package manager for Kubernetes that lets you install and uninstall an application with a single command.

Docker Desktop's built-in Kubernetes (Settings → Kubernetes → Enable Kubernetes) gives you a local single-node cluster with no extra infrastructure.

See [`k8s/README.md`](k8s/README.md) for the full deployment commands (install, access via port-forward, uninstall).

---

## What you learned

You've now seen six Docker technologies working together in a real application:

- **Docker Compose**: defines a multi-service application as a single file. One command starts every service, wires the network, and sets environment variables. Service names act as DNS hostnames so containers find each other without hard-coded IPs.
- **Multi-stage Dockerfiles**: separate build-time tools from the runtime image, producing a smaller, more secure final image. Only the compiled binary and assets are shipped.
- **Docker Hardened Images**: security-hardened base images from `dhi.io`. They ship fewer packages than standard images, so there are fewer potential CVEs in your running container, without changing how you write your Dockerfile.
- **Testcontainers**: lets Go tests start real Docker containers on demand. Tests run against the same Redis version and configuration as production, catching bugs that mocks cannot.
- **Dev Containers**: package the full development environment (toolchain, editor extensions, runtime) as a Docker container defined in code. Every developer gets exactly the same environment, eliminating "works on my machine" mismatches.
- **Kubernetes**: orchestrates containers in production. A Helm chart describes the deployment; Docker Desktop provides a local cluster to try it on.

---

## Make it publicly accessible (optional)

ngrok is a tunneling service. It creates a public HTTPS URL on the internet and forwards all traffic from that URL to a port on your local machine. That's what makes it possible for a player's phone to reach your laptop's game server without any firewall or port-forwarding setup. To use it, you need a free [ngrok](https://ngrok.com) account. The authtoken identifies your account to ngrok's service so it can associate the tunnel with you. Note that `ngrok/ngrok:3` is the one image **not** migrated to Docker Hardened Images: no hardened equivalent exists, and it only runs in this optional `public` profile:

```bash
cp .env.example .env           # once
# edit .env and set NGROK_AUTHTOKEN=<your token>
docker compose --profile public up -d
open http://localhost/host   # shows the current QR code, with a button to rotate it
```

**Inspecting the tunnel**: while the `public` profile is running, ngrok's local web inspector is at <http://localhost:4040>. It shows the current public URL and a live log of incoming requests.

The public URL only serves the game to visitors who have scanned the current QR code. Anyone else sees a "scan the QR code to play" message. Display `/host` on a presenter-only screen: the QR code and its "Rotate" button are not exposed on the public endpoint — nginx routes `/play` to the gated port but has no public `/host` route. Local play at `localhost/play` keeps working even if the tunnel is down.

---

## Leaderboard scores

Before playing, each player enters a display name. On death, their score is shown on a "Game Over" screen and submitted to a Redis-backed leaderboard automatically. A "Replay" button restarts immediately, reusing the same name. Score writes are protected by `LEADERBOARD_API_SECRET` (set in `.env`, injected into the served game page automatically), so only the game client itself can record a score.

Current standings are at `http://localhost/leaderboard`, a wall/booth display that refreshes automatically as new scores come in via a live Server-Sent Events connection to the scores microservice.

---

## Troubleshooting

- **Port 80 already in use**: `docker compose up` fails with "port is already allocated". Stop whatever's using it, or set a different `NGINX_PORT` in `.env` and retry (e.g. `NGINX_PORT=8090`).
- **Port 8080 already in use**: the Go app also publishes 8080 for direct developer access. Stop whatever's using it, or set a different `WEB_PORT` in `.env`.
- **`/qr.png` returns 503**: no QR code has been generated yet (visit `/host` first) or, once public access is enabled, the public URL isn't available yet. Check `docker compose --profile public ps` and `NGROK_AUTHTOKEN` in `.env`.
- **Scanned QR code doesn't work anymore**: it may have expired (default 15 minutes, `QR_WINDOW_TTL` in `.env`) or been rotated from `/host`. Get the current code and re-scan.
- **Redis errors (`/host` → 503, leaderboard → "failed to load standings", `DENIED Redis is running in protected mode`)**: the Docker Hardened Images Redis ships with `protected-mode` **on**, which rejects connections from other containers. This project disables it in `docker-compose.yml`. If you see these errors, that override is missing or was edited out. Full explanation: [`specs/005-dhi-image-migration/contracts/image-inventory.md`](specs/005-dhi-image-migration/contracts/image-inventory.md#known-configuration-difference-dhi-redis-enables-protected-mode).

Full validation walkthroughs are in
[`specs/001-host-webapp-ngrok/quickstart.md`](specs/001-host-webapp-ngrok/quickstart.md) (local + public hosting),
[`specs/002-qr-gated-access/quickstart.md`](specs/002-qr-gated-access/quickstart.md) (the QR gate),
[`specs/003-leaderboard-score-submission/quickstart.md`](specs/003-leaderboard-score-submission/quickstart.md) (name entry, Game Over, score submission), and
[`specs/004-leaderboard-page/quickstart.md`](specs/004-leaderboard-page/quickstart.md) (the `/leaderboard` wall display).

---

## Live page refresh on redeploy

The leaderboard page reloads itself automatically whenever the app is updated and restarted. It works by polling a lightweight endpoint, `/api/ping`, every two seconds. The endpoint returns a JSON object containing a startup ID, which is a nanosecond timestamp captured once when the Go process starts:

```json
{ "id": "1783513264497369178" }
```

The browser stores the first value it receives. On every subsequent poll it compares the current value to the stored one. If they differ, the process restarted (a redeploy happened) and the page calls `location.reload()` to pick up the new version. If the server is temporarily unreachable mid-restart, the fetch error is silently ignored and the next poll tries again, so there are no false reloads during the brief window while the container is coming back up.

---

## Optional: Develop with Claude Code (advanced)

This repo is set up so that typing `claude` inside it launches a Claude Code session **inside a Docker sbx sandbox**, with:

- `--dangerously-skip-permissions` mode on (safe because the whole process is sandboxed)
- Two MCP servers pre-loaded via the hosted Docker MCP gateway: `github`, `context7`
- Your host `gh` token forwarded into the sandbox via `sbx secret set -g github`
- A status line showing model, directory, git branch, and context usage

### Quick start (if you've onboarded before)

```bash
cd <this-repo>
claude
```

That's it. `.envrc` puts `./bin` on PATH, so the `claude` command resolves to the wrapper in `bin/claude`, which launches the sandbox for you.

### First time here

Run the onboarding walkthrough. It checks every prerequisite, prints ✓ or ✗ next to each, and gives you the exact command to fix anything that's red:

```bash
./bin/onboard
```

Fix each ✗ (usually one `brew install` or one `export`), re-run `./bin/onboard`, and repeat until everything is green. Then run `claude`.

### What onboard checks

| Group | Checks |
| --- | --- |
| Tools | `sbx`, `gh`, `direnv`, `node`/`npx` |
| Sign-ins | `gh` authenticated |
| Environment | `SBX_MCP_URL`, `GITHUB_TOKEN` exported |
| direnv | Hooked into your shell, `.envrc` allowed |
| MCP | `github` and `context7` registered with sbx |

### What's in this repo

| File | Purpose |
| --- | --- |
| `bin/claude` | Wrapper that launches Claude inside sbx with the right flags and MCP set |
| `bin/onboard` | Idempotent walkthrough. Run this first, and any time something breaks |
| `bin/setup-mcp` | Registers `github` and `context7` MCP servers with sbx (one-off, per machine) |
| `.envrc` | Adds `./bin` to PATH inside this repo (via direnv), sources `.envrc.local` if present |
| `.claude/settings.json` | Wires the status line |
| `.claude/statusline.sh` | Renders the status line output |

### Common tasks

**Rotate your GitHub token.** `sbx secret set -g github` values are picked up at sandbox *creation*, not on every `claude` launch. If your token changes mid-session, remove the sandbox and re-launch:

```bash
sbx rm claude-sandboxes    # default sandbox name; check with `sbx ls`
claude
```

**Change the MCP server set.** Edit `bin/setup-mcp` to add or remove servers, re-run it, then update `--static-mcp github,context7` in `bin/claude` to match. Static mode is fixed at sandbox creation, so also `sbx rm` any existing sandbox.

**Skip the sandbox for one-off work.** The wrapper is only active while direnv has this directory loaded. Outside the repo, `claude` runs your normal host binary.

### How the pieces fit

```
$ claude
    │
    │ (direnv has ./bin on PATH inside this repo)
    ▼
bin/claude
    │
    ├── preflight: sbx installed? SBX_MCP_URL set? MCP servers registered?
    │       └── any ✗ → prints "run ./bin/onboard" and exits
    │
    ├── echo "$(gh auth token)" | sbx secret set -g github
    │
    └── exec sbx run claude --static-mcp github,context7
```

Inside the sandbox, Claude Code sees `.claude/settings.json` (status line), the two MCP servers are pre-loaded, and `gh` inside the sandbox uses the token you forwarded.

---

## References

- Docker Sandboxes: <https://docs.docker.com/ai/sandboxes/>
- Claude Code inside sbx: <https://docs.docker.com/ai/sandboxes/agents/claude-code/>
- direnv: <https://direnv.net/>
