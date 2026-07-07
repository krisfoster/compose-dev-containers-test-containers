# Quickstart Validation: Dev Container Support

End-to-end validation guide proving the three user stories work. Run these in order from a clean state.

## Prerequisites

- Docker Desktop installed and running on the host
- VS Code with the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) installed
- A fresh `git clone` of the repo (no `.devcontainer/` directory yet — it will be created by the implementation)

---

## User Story 1: Open Project in Dev Container

**Goal**: Fresh clone → working Go + Docker environment in one VS Code action.

### Steps

1. Open the cloned repo folder in VS Code (`File → Open Folder`)
2. When VS Code prompts "Folder contains a Dev Container configuration file. Reopen in Container?", click **Reopen in Container**
   - Alternatively: Command Palette (`Cmd+Shift+P`) → `Dev Containers: Reopen in Container`
3. Watch the container build in the bottom-right status area. First build: 3–5 minutes (downloads base image + features). Subsequent opens: seconds (cached layers).

### Validation checks

From the VS Code integrated terminal (already inside the dev container):

```bash
# Go toolchain present and correct version
go version
# Expected: go version go1.25.x linux/amd64

# Docker CLI present and connected to host daemon (DooD working)
docker version
# Expected: both Client and Server sections printed
# The Server section shows the host Docker daemon version — confirming DooD

# Workspace path matches host path (Testcontainers requirement)
pwd
# Expected: the exact absolute path the project has on your host machine
# e.g. /Users/kris/repos/buk/compose-dev-containers-test-containers

# Project compiles cleanly
cd app && go build ./...
# Expected: exits 0, no output, no errors

# TESTCONTAINERS_HOST_OVERRIDE is set
echo $TESTCONTAINERS_HOST_OVERRIDE
# Expected: host.docker.internal
```

**Pass criteria**: All commands exit cleanly. `go version` shows 1.25.x. `docker version` shows both client and server. `pwd` output matches the host path. `echo` shows `host.docker.internal`.

---

## User Story 2: Run the App Stack from Inside the Dev Container

**Goal**: `docker compose up` from inside the dev container starts all services; game loads in host browser.

### Steps

From the dev container terminal:

```bash
# Start the core stack (redis + app)
cd /path/to/project   # same path as shown by pwd above
docker compose up
```

With DooD, the compose containers bind their ports directly on the **host Docker daemon** — not inside the dev container. VS Code's port-forwarding panel will not show these ports, and that is expected. Open the app directly in your host browser without waiting for a VS Code notification.

### Validation checks

```bash
# In a second terminal tab (both inside the dev container)
docker compose ps
# Expected: redis (running) and app (running) listed
# Note: these containers appear on the HOST daemon — run `docker ps` in a host terminal
# to confirm they're visible there too (DooD: sibling containers)
```

Open **http://localhost:8080** in your **host browser**.
- Expected: Whale Runner landing page with links to `/play`, `/host`, `/leaderboard`

Open **http://localhost:8080/play** in the host browser.
- Expected: Whale Runner game loads and is playable

Stop the stack:

```bash
docker compose down
# Expected: containers stopped and removed cleanly
docker compose ps
# Expected: no containers listed
```

**Pass criteria**: Both services start. Game loads and is playable in the host browser. `docker compose down` cleans up completely.

---

## User Story 3: Run Tests from Inside the Dev Container

**Goal**: Full test suite — including Testcontainers-based integration tests — passes inside the dev container.

### Steps

From the dev container terminal:

```bash
cd app

# Run the full test suite
go test ./... -v -count=1
```

### What to watch for during the run

- Tests in `main_test.go` (unit-level fakes): fast, no container startup
- Tests in `internal/leaderboard/store_test.go` and `internal/gate/window_test.go`: these start real Redis containers via the host Docker daemon

While the Testcontainers tests run, open a terminal **on the host** (outside the dev container) and run:

```bash
docker ps
```

You should see short-lived Redis containers appearing and disappearing — confirming DooD is routing container creation to the host daemon correctly.

### Expected output shape

```
=== RUN   TestRedisScoreStoreWriteThenReadBack
--- PASS: TestRedisScoreStoreWriteThenReadBack (Xs)
=== RUN   TestRedisScoreStoreTopOrdersByScoreDescending
--- PASS: TestRedisScoreStoreTopOrdersByScoreDescending (Xs)
...
ok  	crossywhale/app (Xs)
ok  	crossywhale/app/internal/gate (Xs)
ok  	crossywhale/app/internal/leaderboard (Xs)
```

All packages should show `ok`. No `FAIL` lines. No "connection refused" errors.

### Targeted Testcontainers connectivity check

```bash
go test ./internal/leaderboard/... -v -count=1 -run TestRedisScoreStoreWriteThenReadBack
# Expected: PASS — confirms a real Redis container was spun up via the host daemon,
# connected to successfully, and torn down at test completion
```

**Pass criteria**: All packages show `ok`. Testcontainers-based tests pass. No "connection refused" errors. Redis sibling containers visible in `docker ps` on the host during the test run.

---

## Troubleshooting

### "connection refused" from Testcontainers tests

Check that `TESTCONTAINERS_HOST_OVERRIDE` is set:
```bash
echo $TESTCONTAINERS_HOST_OVERRIDE
# Must print: host.docker.internal
```
If it prints nothing, the container environment variable is not set. Rebuild the dev container: Command Palette → `Dev Containers: Rebuild Container`.

### `docker version` shows only Client, no Server

The DooD socket mount is not working. Check that Docker Desktop is running on the host. Rebuild the dev container.

### Ryuk cleanup errors in test output

Ryuk errors ("Error response from daemon: ...") are typically caused by incorrect socket or host-override configuration, not a Ryuk bug. Verify DooD and `TESTCONTAINERS_HOST_OVERRIDE` first (see above), then re-run the tests.

### `pwd` output differs from the host path

The `workspaceMount` configuration is incorrect — the workspace is not mounted at `${localWorkspaceFolder}`. Testcontainers bind-mounts will be broken. Check `devcontainer.json` and verify `workspaceMount.target` equals `${localWorkspaceFolder}`.

---

## Definition of Done

All of the following must be true before this feature is considered complete (Constitution Principle IV):

- [ ] `go version` inside the dev container shows Go 1.25.x
- [ ] `docker version` inside the container shows both Client and Server (host daemon)
- [ ] `pwd` inside the container matches the project's absolute path on the host
- [ ] `echo $TESTCONTAINERS_HOST_OVERRIDE` prints `host.docker.internal`
- [ ] `go build ./...` succeeds with no errors
- [ ] `go test ./...` passes — all packages `ok`, including Testcontainers tests
- [ ] Testcontainers-spawned Redis containers are visible in `docker ps` on the host during test runs
- [ ] `docker compose up` starts all compose services from inside the dev container
- [ ] Whale Runner game loads and is playable in the host browser at http://localhost:8080
- [ ] `docker compose down` stops and removes all containers cleanly
- [ ] `docker compose up` from **outside** the dev container still works (existing onboarding path unchanged)
