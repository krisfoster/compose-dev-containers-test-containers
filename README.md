# compose-dev-containers-test-containers

Demo app showing how to use Compose, Dev Containers, and Testcontainers together.

## Develop with Claude

This repo is set up so that typing `claude` inside it launches a Claude Code session **inside a Docker sbx sandbox**, with:

- `--dangerously-skip-permissions` mode on (safe because the whole process is sandboxed)
- The [GSD](https://github.com/open-gsd/gsd-core) plugin auto-enabled
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
| `.claude/settings.json` | Enables the GSD plugin and wires the status line |
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

Inside the sandbox, Claude Code sees `.claude/settings.json` (GSD plugin, status line), the two MCP servers are pre-loaded, and `gh` inside the sandbox uses the token you forwarded. User-level config from your host `~/.claude` is deliberately not visible; only what's in this repo is.

## Running the app

**Crossy Whale** (`frontend/game/`, a browser-based Crossy Road clone starring the Docker whale)
is served by a small Go backend (`app/`) via `docker compose`, backed by Redis.

**Prerequisites**:

- Docker Desktop (or Docker Engine + Compose v2). Nothing else to install.
- A free [Docker account](https://hub.docker.com/signup) and a one-time login to the Docker
  Hardened Images registry, since the app and Redis run on hardened base images pulled from
  `dhi.io`:

  ```bash
  docker login dhi.io        # use your free Docker ID; pulling DHI Community images is free
  ```

  (The one exception is the optional `ngrok` tunnel below — it has no hardened image and is pulled
  from Docker Hub as usual, only when you opt into public access.)

### Run it locally

```bash
docker compose up -d
```

Open <http://localhost:8080/> — that's it, no `.env` file or other setup required for local play.
It's a small landing page linking to `/play` (the game), `/host` (the QR code, for public
sessions), and `/leaderboard` (the wall display). Local access is never gated behind a QR scan;
the QR gate only applies to the public endpoint below.

Stop it with:

```bash
docker compose down
```

### Make it publicly accessible (optional)

Sharing a public URL (e.g. for attendees to join over wifi/cellular data) needs a free
[ngrok](https://ngrok.com) account and its authtoken. Note that `ngrok/ngrok:3` is the one image
**not** migrated to Docker Hardened Images — no hardened equivalent exists, and it only runs in this
optional demo-only `public` profile (never on the core local path):

```bash
cp .env.example .env           # once
# edit .env and set NGROK_AUTHTOKEN=<your token>
docker compose --profile public up -d
open http://localhost:8080/host   # shows the current QR code, with a button to rotate it
```

**Inspecting the tunnel.** ngrok's local web inspector — the current public URL plus a live request
log — is served at <http://localhost:4040> while the `public` profile is running. When you develop
inside the sbx sandbox, `bin/claude` automatically publishes both the app port (`WEB_PORT`, default
8080) and ngrok's `4040` from the sandbox to your host on launch, printing the URLs; on a plain host
the compose `4040:4040` mapping already exposes it on `localhost` directly.

The public URL only serves the game to a visitor who has scanned the current QR code (or held a
grant from a previous scan) — anyone else reaching it sees a "scan the QR code to play" message
instead. Display `/host` on a presenter-only screen: the QR code and its "Rotate" button are not
exposed on the public endpoint. Local play at `localhost:8080/play` keeps working even if the
public tunnel is down, not configured, or no QR code has been generated yet.

### Leaderboard scores

Before playing, each player enters a display name. On death, their score is shown on a "Game
Over" screen and submitted to a Redis-backed leaderboard store automatically — no extra step
required. A "Replay" button restarts immediately, reusing the same name. Score writes are
protected by `LEADERBOARD_API_SECRET` (set in `.env`, injected into the served game page
automatically), so only the game client itself can record a score.

Current standings are visible at `http://localhost:8080/leaderboard` — a wall/booth display that
refreshes itself automatically as new scores come in. Viewing it (unlike submitting a score)
requires no credential.

### Troubleshooting

- **Port 8080 already in use**: `docker compose up` fails with "port is already allocated". Stop
  whatever's using it, or set a different `WEB_PORT` in `.env` and retry.
- **`/qr.png` returns 503**: no QR code has been generated yet (visit `/host` first) or, once
  public access is enabled, the public URL isn't available yet — check
  `docker compose --profile public ps` and `NGROK_AUTHTOKEN` in `.env`.
- **Scanned QR code doesn't work anymore**: it may have expired (default 15 minutes,
  `QR_WINDOW_TTL` in `.env`) or been rotated from `/host` — get the current code and re-scan.
- **Redis errors (`/host` → 503, leaderboard → "failed to load standings", `DENIED Redis is running
  in protected mode`)**: the Docker Hardened Images Redis ships with `protected-mode` **on**, which
  rejects connections from other containers. This project disables it in `docker-compose.yml`
  (`command: ["redis-server", "/etc/redis/redis.conf", "--protected-mode", "no"]`) because Redis has
  no published port and is reachable only on the internal compose network. If you see these errors,
  that override is missing or was edited out. Full explanation and the tests' equivalent fix:
  [`specs/005-dhi-image-migration/contracts/image-inventory.md`](specs/005-dhi-image-migration/contracts/image-inventory.md#known-configuration-difference-dhi-redis-enables-protected-mode).

Full validation walkthroughs are in
[`specs/001-host-webapp-ngrok/quickstart.md`](specs/001-host-webapp-ngrok/quickstart.md) (local +
public hosting),
[`specs/002-qr-gated-access/quickstart.md`](specs/002-qr-gated-access/quickstart.md) (the QR gate),
[`specs/003-leaderboard-score-submission/quickstart.md`](specs/003-leaderboard-score-submission/quickstart.md)
(name entry, Game Over, and leaderboard score submission), and
[`specs/004-leaderboard-page/quickstart.md`](specs/004-leaderboard-page/quickstart.md) (the
`/leaderboard` wall display).

## Develop inside a Dev Container

VS Code with the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) is the recommended local development environment. Opening the project in a dev container gives you a fully-equipped Go 1.25 workspace with no host-side installs beyond Docker Desktop and git.

**Prerequisites**: Docker Desktop running on the host, VS Code, Dev Containers extension installed.

**Open in one step**: Open the repo folder in VS Code, then when prompted click **Reopen in Container** (or use the Command Palette: `Dev Containers: Reopen in Container`). The first build takes a few minutes; subsequent opens are instant.

**What you get inside the container**:

- Go 1.25 toolchain — `go build ./...` and `go test ./...` work immediately
- Docker CLI connected to the host daemon — `docker compose up` runs the full app stack (game, leaderboard, Redis) from inside the container; the game is accessible in your host browser at <http://localhost:8080>
- Testcontainers-based integration tests work — the Go tests that spin up a real Redis container (`internal/leaderboard/`, `internal/gate/`) pass inside the dev container, no extra setup required
- Go editor intelligence — autocomplete, go-to-definition, inline errors, and auto-format on save via the official Go extension

**Full validation walkthrough**: [`specs/006-dev-container-support/quickstart.md`](specs/006-dev-container-support/quickstart.md)

## References

- Docker Sandboxes: <https://docs.docker.com/ai/sandboxes/>
- Claude Code inside sbx: <https://docs.docker.com/ai/sandboxes/agents/claude-code/>
- direnv: <https://direnv.net/>
- GSD plugin: <https://github.com/open-gsd/gsd-core>
