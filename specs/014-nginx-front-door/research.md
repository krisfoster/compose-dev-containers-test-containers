# Research: Nginx Front-Door Routing Layer

## 1. nginx Image Selection (DHI Compliance)

**Decision**: Use `dhi.io/nginx:1-alpine3.24` (DHI hardened nginx, mainline branch, alpine 3.24)

**Rationale**: DHI provides a hardened nginx image at `dhi.io/nginx` with alpine 3.24 distribution,
matching the `alpine3.24` variant already used throughout the project (e.g.,
`dhi.io/static:20260611-alpine3.24`). The `1-alpine3.24` tag tracks nginx 1.x mainline on
alpine 3.24 — STIG-certified, FIPS-compliant, regularly patched. This satisfies the constitution's
implicit requirement to use DHI images where they exist, and avoids pulling from Docker Hub's
`nginx:alpine` directly.

**Alternatives considered**:
- `nginx:alpine` from Docker Hub — available but not hardened; rejected in favour of DHI equivalent
- `dhi.io/nginx:1-debian13` — available but heavier than alpine; alpine3.24 is sufficient for a routing-only container
- OpenResty (DHI: `dhi.io/openresty:1-debian13`) — adds Lua scripting capability; rejected as unnecessary for pure routing

---

## 2. nginx SSE Proxying

**Decision**: Disable proxy buffering and use HTTP/1.1 with empty `Connection` header for all
SSE upstream locations.

**Rationale**: nginx's default proxy buffering caches chunks before forwarding them to the client.
For Server-Sent Events, this would hold events in the buffer until it fills (defaulting to several
KB) rather than flushing immediately — defeating the purpose of a streaming protocol. The required
directives for a streaming-compatible SSE proxy location are:

```nginx
proxy_http_version 1.1;       # use persistent connection to upstream
proxy_set_header Connection '';  # clear hop-by-hop Connection header so keep-alive works
proxy_buffering off;           # send each chunk immediately to the browser
proxy_cache off;               # do not cache SSE responses
proxy_read_timeout 3600s;      # allow connection to stay open for at least 1 hour
```

These directives are applied only to the `/scores` and `/commits` location blocks, not globally,
to avoid disabling buffering for static file serving (where buffering is beneficial).

**Alternatives considered**:
- Global `proxy_buffering off` — simpler but disables buffering for static file serving too; rejected
- Adding `X-Accel-Buffering: no` header in Go microservices — works for some nginx setups but is
  less reliable than explicit nginx config; rejected in favour of explicit config

---

## 3. Cookie and Header Forwarding for Gate Middleware

**Decision**: Forward `Cookie` request header and pass through `Set-Cookie` response header
for proxy locations that interact with the Go gate (`/play`, `/host`).

**Rationale**: The QR gate relies on a grant cookie (`grant` cookie signed by the Go app) to
authorize player access. nginx's default proxy config does not strip cookies, but `Set-Cookie`
response headers can be affected by some proxy configurations. Explicit directives:

```nginx
proxy_set_header Cookie $http_cookie;   # forward incoming cookies
proxy_pass_header Set-Cookie;           # pass Set-Cookie from upstream to client
```

These are added to the `/play` and `/host` proxy blocks. The `/api/` block also includes
`Cookie` forwarding since the leaderboard API uses its own `Authorization` header (not a cookie),
but future routes may need cookies.

**Alternatives considered**:
- Not explicitly forwarding cookies and relying on nginx defaults — risky; nginx does pass cookies
  by default in basic configs but explicit declaration is safer and documents intent

---

## 4. Static File Volume Strategy

**Decision**: Copy static files into the nginx image at build time using a multi-stage approach
in `nginx/Dockerfile`. Build context is the repo root.

**Rationale**: Copying files at build time produces a self-contained image with no runtime volume
mounts, consistent with how the `app` container is built (`COPY frontend/game /frontend`). This
avoids compose-level named volumes, bind mounts, or shared `volumes_from` dependencies between
nginx and app. The nginx image is independently deployable.

Dockerfile layout (build context: repo root):
```dockerfile
FROM dhi.io/nginx:1-alpine3.24
COPY frontend/game /usr/share/nginx/html
COPY container-obstacles.gif /usr/share/nginx/html/container-obstacles.gif
COPY frontend/leaderboard /usr/share/nginx/html/leaderboard-assets
COPY nginx/nginx.conf /etc/nginx/nginx.conf
```

Static files end up at:
- `/usr/share/nginx/html/` — game assets (served at root paths: `/script.js`, `/model.glb`, etc.)
- `/usr/share/nginx/html/leaderboard-assets/` — React bundles + component JS (served at `/leaderboard-assets/*`)
- `/usr/share/nginx/html/container-obstacles.gif` — served at `/container-obstacles.gif`

All paths in the nginx.conf `root /usr/share/nginx/html` then resolve correctly for static files.

**Alternatives considered**:
- Runtime bind mount (`volumes: - ./frontend/game:/usr/share/nginx/html:ro`) — simpler for dev
  but couples nginx startup to host filesystem layout; rejected for consistency with the self-contained image pattern
- Named Docker volume shared between app and nginx — adds compose orchestration complexity with
  no benefit; rejected

---

## 5. Port Publishing Strategy

**Decision**: Publish only nginx port 80 to the host; remove no existing published ports but
document that port 8080 is now a development-convenience port, not the primary entry point.

**Current state**:
- `app`: `8080:8080` (published — ungated presenter view)
- `commits-service`: `8082:8082` (published — browser SSE access)
- `scores-service`: `8083:8083` (published — browser SSE access)
- `ngrok`: `4040:4040` (published — ngrok inspection UI)
- `app:8081` — internal only (gated, targeted by ngrok)

**After this change**:
- `nginx`: `80:80` (new — single public ingress)
- `app:8080`: remains published for direct developer access (optional, not the primary path)
- `commits-service:8082`, `scores-service:8083`: remain published for direct service access/debugging
- `ngrok:4040`: unchanged
- `app:8081`: remains internal (nginx proxies to it; not published)

The browser-facing entry point becomes `http://localhost` (port 80 via nginx). Existing direct
service ports remain available for debugging but are no longer the primary path.

**Alternatives considered**:
- Remove all direct service ports (pure single-ingress) — cleaner for demos but removes the
  ability to curl microservices directly during development; rejected for developer ergonomics

---

## 6. SCORES_SERVICE_URL and COMMITS_SERVICE_URL Update

**Decision**: Change default values from `http://localhost:8083` / `http://localhost:8082`
to `http://localhost` (port 80 via nginx) in docker-compose.yml.

**Rationale**: The leaderboard template injects these as browser-visible base URLs for the React
components. The ScoresComponent constructs SSE URLs as `scoresServiceURL + '/scores/stream'`.
With nginx routing `/scores/*` to `scores-service:8083`, the correct browser URL becomes
`http://localhost/scores/stream`. Setting the base URL to `http://localhost` (no port, port 80 default)
produces the correct full URL without modifying any React component code.

Component URL construction (unchanged):
```js
// ScoresComponent
EventSource(scoresServiceURL + '/scores/stream')  → "http://localhost/scores/stream"
fetch(scoresServiceURL + '/scores')               → "http://localhost/scores"

// CommitsComponent  
EventSource(commitsServiceURL + '/commits/stream') → "http://localhost/commits/stream"
fetch(commitsServiceURL + '/commits')              → "http://localhost/commits"
```

Updated docker-compose.yml environment for `app` service:
```yaml
- SCORES_SERVICE_URL=${SCORES_SERVICE_URL:-http://localhost}
- COMMITS_SERVICE_URL=${COMMITS_SERVICE_URL:-http://localhost}
```

For override via `.env` in demo environments where the public URL is used (e.g., ngrok domain),
users set `SCORES_SERVICE_URL=https://dockerdemo.ngrok.app` — this continues to work because
nginx proxies `/scores/*` on the ngrok-facing side too.

**Alternatives considered**:
- Relative URLs (empty `scoresServiceURL`) — would require component code changes to construct
  relative URLs; rejected to avoid Go component changes in this scope
- Keep port-specific URLs and publish ports — the new single-ingress goal is defeated; rejected

---

## 7. ngrok Configuration Update

**Decision**: Change `ngrok.yml` `addr` from `http://app:8081` to `http://nginx:80`.

**Rationale**: With nginx as the single public ingress, all external traffic (including player
phone access) must enter through nginx, which then routes `/play` to the gated port. If ngrok
still pointed to `app:8081`, the entire nginx routing layer would be bypassed for external traffic.

ngrok.yml after change:
```yaml
tunnels:
  whale-runner:
    addr: http://nginx:80
    proto: http
    domain: dockerdemo.ngrok.app
```

The Go app's `discoverPublicHost` function (which queries `ngrok:4040/api/tunnels`) is unaffected;
it reads the public URL regardless of the upstream target.

**Alternatives considered**:
- Multiple ngrok tunnels (one to nginx, one to app) — unnecessarily complex; rejected
- Keep ngrok pointed at app:8081 and have app:8081 proxy SSE — the whole point of this feature
  is to avoid that; rejected

---

## 8. compose `depends_on` for nginx

**Decision**: nginx service depends on `app`, `scores-service`, and `commits-service` with
`condition: service_started` (not `service_healthy`).

**Rationale**: nginx starts quickly and returns 502/503 if an upstream is not yet ready, which is
acceptable at compose start. The compose `depends_on` with `service_started` ensures the upstream
containers exist before nginx tries to resolve their hostnames, but does not wait for them to
become healthy. This matches the pattern used by other services in the compose file.

**Note**: The `app` service's existing dependency on `ngrok` (service_healthy, required: false)
is unchanged — the app still queries ngrok to discover its public URL for QR code generation.
