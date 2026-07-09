# Research: App Serves Templates Only

**Feature**: 017-app-serves-templates-only  
**Date**: 2026-07-09

---

## Goal

Remove all static game assets from the app container. After this change the app
binary serves only Go-rendered templates and dynamic API routes. nginx becomes the
sole server of static files. The app container image shrinks accordingly.

---

## Why the duplication exists

The app was the original sole server. Before nginx was introduced as the front door
(spec 014), the Go `http.FileServer` served every asset: `script.js`, `style.css`,
3D models, audio, and `index.html`. When nginx was added it took over static file
serving for all browser traffic, but the `COPY frontend/game /frontend` and
`COPY frontend/leaderboard /leaderboard-assets` lines in `app/Dockerfile` were never
removed. The assets have been duplicated across both containers ever since.

---

## What each container currently holds

| Asset | nginx (`/usr/share/nginx/html`) | app (`/frontend`, `/leaderboard-assets`) |
|---|---|---|
| `frontend/game/*` (script.js, style.css, models, audio, favicon, manifest) | ✓ via `COPY frontend/game` | ✓ via `COPY frontend/game /frontend` |
| `frontend/game/index.html` | ✓ (part of the above) | ✓ (part of the above) |
| `frontend/leaderboard/*` (React bundles) | ✓ via `COPY frontend/leaderboard` | ✓ via `COPY frontend/leaderboard /leaderboard-assets` |

---

## Which copy is actually used

All browser traffic enters through nginx. The nginx.conf routing:

- `location = /play` and `location = /play-local` proxy to `app:8080` — the app
  serves `index.html` as a template response.
- `location /leaderboard-assets/` is absent from nginx.conf; the catch-all
  `location / { try_files $uri =404; }` serves these files directly from nginx's
  document root.
- All other game assets (`/script.js`, `/style.css`, models, audio) are also
  handled by the `try_files` catch-all, served from nginx directly.

**The app's file server is never reached through nginx.** The Go `http.FileServer`
at `a.frontendDir` and the `/leaderboard-assets/` handler in `ungatedMux()` are only
reachable via direct access to port 8080, which is a developer debugging port, not a
user-facing path.

---

## What the app actually needs from the frontend directory

Only `index.html`. `handlePlayIndex` calls `template.ParseFiles(filepath.Join(a.frontendDir, "index.html"))` to serve the game page for both `/play` and `/play-local`.

`index.html` contains **no Go template variables**. The `{{.LeaderboardToken}}`
injection was removed in spec 016. `tmpl.Execute(w, nil)` passes nil data. The file
is effectively static HTML parsed unnecessarily through the template engine.

---

## Decision: move index.html to templates/

`templates/` already holds the other app-rendered pages: `getting-started.html`,
`host.html`, and `leaderboard.html`. All are rendered by Go handlers and live-reloaded
via the `./templates:/templates` bind mount in docker-compose.yml. `index.html` fits
naturally in this directory.

Moving it there means:
- `handlePlayIndex` reads from `a.templatesDir` — consistent with all other page handlers.
- Live reload of `index.html` works automatically via the existing bind mount.
- `watchTemplates` gains `index.html` so browser auto-reload triggers on game page changes.
- nginx keeps `frontend/game/index.html` as the source for its own copy (the `COPY`
  in `nginx/Dockerfile` doesn't change). However, since nginx never serves `index.html`
  directly for the `/play` path (always proxied to app), removing it from nginx's
  document root would also be harmless. We leave `nginx/Dockerfile` unchanged for now.

---

## Decision: keep template.ParseFiles rather than switching to http.ServeFile

`template.ParseFiles` is used for every other page handler in this codebase.
Switching `handlePlayIndex` to `http.ServeFile` would be inconsistent and gains
nothing meaningful (no template caching, no measurable performance difference for
this workload). If template variables are added back in a future spec, the
infrastructure is already in place. Keep `template.ParseFiles`.

---

## Impact on the app container image

The files removed from the app image:

| Path | Approximate size |
|---|---|
| `frontend/game/script.js` | ~300 KB |
| `frontend/game/style.css` | ~10 KB |
| `frontend/game/models/` (GLB files) | ~2 MB |
| `frontend/game/` (audio, favicon, manifest, etc.) | ~200 KB |
| `frontend/leaderboard/` (React bundles) | ~200 KB |

Total savings: roughly 2–3 MB from the app image layer. Not enormous, but the image
becomes semantically correct — it contains only what the app actually serves.

---

## Impact on direct :8080 access (developer debug port)

After this change, requests to `http://localhost:8080/script.js` or any other static
asset path return 404. The game page at `http://localhost:8080/play` will load the
HTML but the browser will fail to fetch the associated scripts and styles.

This is acceptable. Port 8080 is documented as a developer debugging port. All normal
access — including the presenter's local play at `http://localhost:8081/play-local`
— goes through nginx, which serves the assets correctly.

---

## Files changing

| File | Change |
|---|---|
| `app/Dockerfile` | Remove two `COPY` lines |
| `app/main.go` | Remove `frontendDir`, `leaderboardAssetsDir` fields and config; update `handlePlayIndex`; simplify `handleRootOrAsset`; update `ungatedMux`; update `watchTemplates`; update package comment |
| `app/main_test.go` | Replace `frontendDir` temp dir setup with `index.html` written into `templatesDir`; remove `FRONTEND_DIR` from config test; remove `script.js` fake |
| `docker-compose.yml` | Remove `FRONTEND_DIR` and `LEADERBOARD_ASSETS_DIR` env vars from app service |
| `templates/index.html` | New file — moved from `frontend/game/index.html` |
| `nginx/Dockerfile` | No change |
| `frontend/game/index.html` | No change — remains as source for nginx's static copy |

---

## Out of scope

- Removing `index.html` from nginx's static document root. nginx never serves it for
  `/play` (always proxied), but it could be served if someone requests
  `http://localhost:8081/index.html` directly. Leaving this unchanged keeps a working
  fallback and avoids touching `nginx/Dockerfile`.
- Kubernetes deployment. The k8s Helm chart uses the `whale-runner:k8s-local` app
  image. After this change that image will not serve static game assets. Whether
  the k8s deployment needs an nginx sidecar is out of scope for this spec.
