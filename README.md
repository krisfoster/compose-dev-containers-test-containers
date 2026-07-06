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

The web app (`frontend/game/`, a browser-based Crossy Road clone) is hosted via `docker compose`.

**Prerequisites**: Docker Desktop (or Docker Engine + Compose v2). Nothing else to install.

### Run it locally

```bash
docker compose up -d
```

Open <http://localhost:8080> — that's it, no `.env` file or other setup required for local play.

Stop it with:

```bash
docker compose down
```

### Make it publicly accessible (optional)

Sharing a public URL (e.g. for attendees to join over wifi/cellular data) needs a free
[ngrok](https://ngrok.com) account and its authtoken:

```bash
cp .env.example .env           # once
# edit .env and set NGROK_AUTHTOKEN=<your token>
docker compose --profile public up -d
open http://localhost:4040     # shows the current public URL to share
```

Local play at `localhost:8080` keeps working even if the public tunnel is down or not configured.

### Troubleshooting

- **Port 8080 already in use**: `docker compose up` fails with "port is already allocated". Stop
  whatever's using it, or set a different `WEB_PORT` in `.env` and retry.
- **No public URL at `localhost:4040`**: usually a missing/invalid `NGROK_AUTHTOKEN` in `.env`, or
  the ngrok service isn't running — check with `docker compose --profile public ps`.

Full validation walkthrough (including simulating a port conflict or a tunnel outage) is in
[`specs/001-host-webapp-ngrok/quickstart.md`](specs/001-host-webapp-ngrok/quickstart.md).

## References

- Docker Sandboxes: <https://docs.docker.com/ai/sandboxes/>
- Claude Code inside sbx: <https://docs.docker.com/ai/sandboxes/agents/claude-code/>
- direnv: <https://direnv.net/>
- GSD plugin: <https://github.com/open-gsd/gsd-core>
