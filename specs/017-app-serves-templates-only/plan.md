# Plan: App Serves Templates Only

**Feature**: 017-app-serves-templates-only  
**Date**: 2026-07-09

---

## Goal

**The app container must contain only what the app actually serves: the compiled
binary and the four Go-rendered page templates.** All static game assets (JavaScript,
CSS, 3D models, audio, React bundles) are served exclusively by nginx, which already
holds copies of every file. The duplication introduced when nginx was added as the
front door (spec 014) is eliminated.

---

## Architecture: before and after

### Before

```
Browser → nginx:80 → try_files → /usr/share/nginx/html/script.js  ← COPY frontend/game
                   → proxy → app:8080/play → /frontend/index.html  ← COPY frontend/game
                                           → /leaderboard-assets/  ← COPY frontend/leaderboard

app container holds:
  /app            (binary)
  /frontend/      (index.html + all game assets — REDUNDANT for static files)
  /leaderboard-assets/  (React bundles — REDUNDANT)
  /templates/     (getting-started.html, host.html, leaderboard.html)
```

### After

```
Browser → nginx:80 → try_files → /usr/share/nginx/html/script.js  ← COPY frontend/game
                   → proxy → app:8080/play → /templates/index.html

app container holds:
  /app            (binary)
  /templates/     (index.html, getting-started.html, host.html, leaderboard.html)
```

---

## Tech stack

- Go 1.25 (existing)
- nginx (existing — no changes to nginx/Dockerfile or nginx.conf)
- Docker Compose (existing — remove two env vars from app service)

---

## Changes by file

### `templates/index.html` (new)
Copy of `frontend/game/index.html`. This becomes the canonical source the app reads
when serving `/play` and `/play-local`. The original at `frontend/game/index.html`
remains — nginx's `COPY frontend/game /usr/share/nginx/html` continues to pick it up.

### `app/Dockerfile`
Remove:
```dockerfile
COPY frontend/game /frontend
COPY frontend/leaderboard /leaderboard-assets
```
The image then contains only `/app` (binary), `/repo` (git mount point), and
`/templates` (page templates).

### `app/main.go`

**`Config` struct** — remove two fields:
- `FrontendDir string`
- `LeaderboardAssetsDir string`

**`loadConfig()`** — remove two assignments:
- `FrontendDir: envOr("FRONTEND_DIR", "/frontend")`
- `LeaderboardAssetsDir: envOr("LEADERBOARD_ASSETS_DIR", "/leaderboard-assets")`

**`App` struct** — remove two fields:
- `frontendDir string`
- `leaderboardAssetsDir string`

**`main()`** — remove two fields from `app := &App{...}`:
- `frontendDir: cfg.FrontendDir`
- `leaderboardAssetsDir: cfg.LeaderboardAssetsDir`

**`handlePlayIndex`** — change read path from `a.frontendDir` to `a.templatesDir`;
update the doc comment (the leaderboard credential injection reason is stale since
spec 016).

**`ungatedMux()`** — remove:
- `fileServer := http.FileServer(http.Dir(a.frontendDir))`
- `mux.Handle("/leaderboard-assets/", http.StripPrefix(...))` 
- Pass the file server into `handleRootOrAsset` (simplify signature)

**`handleRootOrAsset`** — remove the `assets http.Handler` parameter; for non-root
paths return `http.NotFound` instead of delegating to the file server.

**`watchTemplates`** — add `index.html` to the watched file list so browser
auto-reload fires when the game page template changes on disk.

**Package doc comment** — remove the stale reference to leaderboard API spec files.

### `app/main_test.go`

- `newTestApp()`: write `index.html` into `templatesDir` instead of a separate
  `frontendDir`; remove the `script.js` fake write; remove `frontendDir` field from
  `App` struct literal.
- `appWithErroringStore()` and any other helper that sets `frontendDir`: same
  removal.
- `TestLoadConfigDefaults` / `TestLoadConfigReadsOverrides`: remove `FRONTEND_DIR`
  and `LEADERBOARD_ASSETS_DIR` from the env var loop.
- Any test that references `frontendDir` directly: update to use `templatesDir`.

### `docker-compose.yml`
Remove from the `app` service `environment` block:
- `- FRONTEND_DIR=/frontend`
- `- LEADERBOARD_ASSETS_DIR=/leaderboard-assets`

---

## What does NOT change

- `nginx/Dockerfile` — unchanged; continues to `COPY frontend/game` and
  `COPY frontend/leaderboard` into its document root.
- `nginx/nginx.conf` — unchanged.
- `frontend/game/index.html` — unchanged; remains the source nginx copies.
- `templates/` bind mount in docker-compose.yml — unchanged; already covers the
  new `templates/index.html` for live reload.
- scores-service, commits-service, redis — untouched.
- The leaderboard-assets route through nginx (`try_files`) — unchanged.

---

## Test strategy

All existing tests must continue to pass (`go test ./...`). The only test changes
are mechanical: replace `frontendDir` temp dir setup with writing `index.html` into
the existing `templatesDir` temp dir, and remove config assertions for the two
removed env vars.

No new tests are required. The behaviour under test (`handlePlayIndex` returns 200
with game HTML, `handleRootOrAsset` returns getting-started for `/`) is unchanged.

---

## Validation

After implementation:

1. `go test ./...` — all tests pass.
2. `docker compose up --build -d` — stack starts cleanly.
3. `http://localhost:8081/play-local` — game page loads, scripts and models run.
4. `http://localhost:8081/leaderboard` — leaderboard page loads with live scores.
5. `docker exec whale-runner-app-1 ls /` — no `/frontend` or `/leaderboard-assets`
   directories present.
