# nginx Routing Table Contract

**Service**: `nginx` (compose service, port 80)
**Image**: `dhi.io/nginx:1-alpine3.24`
**Version**: 1.0.0

This document is the authoritative routing contract for the nginx front-door service introduced
by `014-nginx-front-door`. It defines how each incoming path is handled, which upstream receives
the request, and any special proxy directives required.

---

## Routing Rules (evaluated in nginx priority order)

| Priority | Location pattern | Match type | Upstream | Notes |
|----------|-----------------|------------|----------|-------|
| 1 | `= /play` | Exact | `app:8081` (gated) | QR gate middleware; forwards `Cookie`, passes through `Set-Cookie` |
| 2 | `= /` | Exact | `app:8080` (ungated) | Landing page (inline Go HTML) |
| 2 | `= /leaderboard` | Exact | `app:8080` (ungated) | Leaderboard template page |
| 2 | `= /qr.png` | Exact | `app:8080` (ungated) | Live QR code PNG; Cache-Control: no-store |
| 2 | `= /repo-qr.png` | Exact | `app:8080` (ungated) | Static repo QR code PNG |
| 3 | `/host` | Prefix | `app:8080` (ungated) | Matches `/host` and `/host/rotate`; forwards `Cookie`, passes through `Set-Cookie` |
| 3 | `/api/` | Prefix | `app:8080` (ungated) | API routes including `/api/ping`, `/api/leaderboard/scores` |
| 3 | `/scores` | Prefix | `scores-service:8083` | REST + SSE; buffering off; timeout 3600s |
| 3 | `/commits` | Prefix | `commits-service:8082` | REST + SSE; buffering off; timeout 3600s |
| 4 | `/` | Prefix (catch-all) | `static: /usr/share/nginx/html` | Game assets (JS, CSS, GLB, audio), leaderboard bundles at `/leaderboard-assets/` |

**Match types:**
- `Exact` (`=`) — path must match character for character (query strings ignored in location matching)
- `Prefix` — path must start with the given string; longest prefix wins when multiple match

---

## Static File Layout (inside nginx image)

```text
/usr/share/nginx/html/
├── index.html                      ← game index (raw, not the Go-rendered play page)
├── script.js                       ← game bundle
├── three.module.js                 ← three.js module
├── container-obstacles.gif         ← gameplay demo GIF (shown on leaderboard)
├── *.glb, *.wav, *.png, ...        ← game assets
└── leaderboard-assets/
    ├── react.production.min.js
    ├── react-dom.production.min.js
    ├── scores-component.js
    └── commits-component.js
```

Source in repo:
- `frontend/game/` → `/usr/share/nginx/html/` (copied at build time)
- `container-obstacles.gif` → `/usr/share/nginx/html/container-obstacles.gif`
- `frontend/leaderboard/` → `/usr/share/nginx/html/leaderboard-assets/` (copied at build time)

---

## Proxy Headers

For all `proxy_pass` locations, nginx sets:

| Header | Value | Purpose |
|--------|-------|---------|
| `Host` | `$host` | Preserves the original `Host` header for the upstream |
| `X-Real-IP` | `$remote_addr` | Passes the client IP for logging (gated + API routes) |
| `Cookie` | `$http_cookie` | Required for gate cookie forwarding (`/play`, `/host`) |

Response headers passed through from upstream to client:

| Header | Applies to | Purpose |
|--------|-----------|---------|
| `Set-Cookie` | `/play`, `/host` | Allows gate to set grant cookie on the browser |

---

## SSE-Specific Directives

Applied to `/scores` and `/commits` proxy blocks:

```nginx
proxy_http_version 1.1;       # persistent connection to upstream
proxy_set_header Connection '';  # clear hop-by-hop header; enables keep-alive
proxy_buffering off;           # flush each SSE event immediately; do not buffer
proxy_cache off;               # never cache SSE responses
proxy_read_timeout 3600s;      # allow streaming connections open for up to 1 hour
```

---

## Paths NOT Served by nginx

These paths exist on the **gated mux** (`app:8081`) but are not explicitly routed by nginx and
are therefore unreachable through port 80 by design:

| Path | Why excluded |
|------|-------------|
| `app:8081/` (catch-all static files) | Static files served by nginx directly |
| `app:8081/leaderboard` | Served directly by nginx proxy to `app:8080` |
| `app:8081/leaderboard-assets/` | Served as static files by nginx |
| `app:8081/repo-qr.png` | Served via nginx proxy to `app:8080` |
| `app:8081/api/*` | Served via nginx proxy to `app:8080` |

Only `/play` on the gated port is intentionally reachable via nginx.

---

## No-Business-Logic Contract

nginx MUST NOT:
- Inspect or validate the `grant` cookie
- Generate or sign any token
- Apply rate limits based on player identity
- Store any session state
- Return any HTML that is not a static file

All such behaviour remains in `app` (Go).
