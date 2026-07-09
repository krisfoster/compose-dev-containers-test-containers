# Handoff: nginx auth_request for Score Integrity

**Feature**: 016-nginx-auth-score-integrity
**Status**: Research complete — ready for spec / task breakdown
**Raised**: 2026-07-09

---

## Problem

Score submissions are currently protected by `LEADERBOARD_API_SECRET`, but this secret is **visible in
the browser**. `handlePlayIndex` in `app/main.go` injects it into the served `index.html` as a Go
template variable:

```go
// app/main.go:271
data := struct{ LeaderboardToken string }{LeaderboardToken: a.leaderboardSecret}
tmpl.Execute(w, data)
```

The game JavaScript reads it from `window.__LEADERBOARD_TOKEN__` and sends it as the
`X-Leaderboard-Token` request header (see `CredentialHeader` in
`app/internal/leaderboard/handler.go:14`). Any player who opens DevTools or reads the page source
can extract the token and POST arbitrary scores directly to the API endpoint, with no need to play
the game.

The QR gate (`cw_grant` cookie) stops uninvited users from loading the game page, but once they have
the page they have the token. The two protections are independent and the weaker one (the token)
is exposed to everyone the stronger one (the gate) already trusts.

---

## Proposed Solution

Replace the embedded token with nginx `auth_request`. nginx validates the `cw_grant` cookie before
forwarding any score submission to the Go app. No secret is sent to the browser at all.

### Request flow (after change)

```
POST /api/leaderboard/scores
  │
  └─▶ nginx
        │
        ├─▶ auth_request → GET /auth/check (subrequest, internal)
        │       │  app validates cw_grant cookie — same HMAC + Redis window TTL
        │       │  check already used for /play
        │       └─▶ 200 OK  →  nginx forwards POST to app
        │           401     →  nginx returns 401 to browser; app never sees request
        │
        └─▶ app:8080 /api/leaderboard/scores
              handler validates payload, writes to Redis stream, notifies scores-service
```

The browser game sends no `X-Leaderboard-Token` header. Auth relies entirely on the `cw_grant`
cookie already present in the browser after scanning the QR code.

### nginx config change (nginx/nginx.conf)

```nginx
location /api/leaderboard/scores {
    auth_request /auth/check;
    proxy_pass http://app:8080;
}

location = /auth/check {
    internal;
    proxy_pass http://app:8080/auth/check;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header Cookie $http_cookie;   # forward cw_grant cookie
}
```

### New Go endpoint (app/main.go or app/internal/gate)

Add `GET /auth/check` to the ungated mux. It reads the `cw_grant` cookie, runs the existing
`gate.Signer` + `gate.WindowStore` validation, and returns:
- `200 OK` with empty body — cookie valid
- `401 Unauthorized` with empty body — cookie absent, invalid, or expired

No new logic — it is the same check already performed by `gate.Middleware` on the gated listener.
A thin handler that reuses the existing gate machinery is sufficient.

---

## Dead Code and Cleanup Required

Implementing this solution makes several existing constructs redundant. They must be removed as part
of the same change to avoid leaving misleading code in place.

### 1. `LEADERBOARD_API_SECRET` — remove entirely

| Location | What to remove |
|---|---|
| `app/main.go` `Config.LeaderboardAPISecret` | Field and `envOr("LEADERBOARD_API_SECRET", ...)` |
| `app/main.go` `App.leaderboardSecret` | Field and all assignments |
| `app/main.go` `handlePlayIndex` | `data.LeaderboardToken` injection |
| `app/main.go` `leaderboard.NewHandler(...)` | `secret` argument — pass `""` or remove the parameter |
| `app/internal/leaderboard/handler.go` | `CredentialHeader` const, `validCredential()` function, credential check in `serveSubmit` |
| `app/internal/leaderboard/handler.go` `Handler.secret` | Field and `NewHandler` parameter |
| `docker-compose.yml` | `LEADERBOARD_API_SECRET` environment entry |
| `frontend/game/index.html` | `window.__LEADERBOARD_TOKEN__` script tag |
| `frontend/game/*.js` | Any reference to `__LEADERBOARD_TOKEN__` or the `X-Leaderboard-Token` header |

### 2. Two-listener architecture — consolidate to one port (optional, but logical)

The gated listener (`app:8081`) exists so nginx can route `/play` to a port where Go middleware
enforces the `cw_grant` check. With `auth_request`, nginx enforces the check itself. The Go app no
longer needs a second listener.

| Location | What to remove |
|---|---|
| `app/main.go` `Config.GatedPort` | Field and `envOr("APP_GATED_PORT", ...)` |
| `app/main.go` `main()` | Second `http.ListenAndServe` goroutine for `app.gatedMux()` |
| `app/main.go` `gatedMux()` | Entire method |
| `app/main.go` `App.gate` | Field (gate middleware no longer applied in a mux) |
| `docker-compose.yml` | `APP_GATED_PORT` environment entry |
| `nginx/nginx.conf` | Any proxy to `app:8081`; all routes now point to `app:8080` |

`gate.Gate`, `gate.Middleware`, and the underlying `gate` package are **not** dead — the new
`/auth/check` handler uses `gate.Signer` and `gate.WindowStore` directly. Only the mux-level
middleware wiring goes away.

---

## What This Does Not Fix

Replacing the token with `auth_request` prevents score submission without a valid QR grant. It does
not prevent a legitimate grant-holder from hand-crafting a POST with an inflated score — they have
a real cookie and they know the API shape.

Closing that gap requires server-side session tracking:
- Server issues a short-lived play token when the game loads
- Token is single-use and bound to a session ID
- Score submission validates the token server-side before writing

This is a materially larger change and likely out of scope for a booth demo. Record it here as a
known limitation rather than a defect.

---

## Architecture Before / After

### Before

```
browser (phone)
  └─ cw_grant cookie (QR gate)
  └─ window.__LEADERBOARD_TOKEN__ (visible in page source)

nginx
  /play  →  app:8081  (gated listener — Go middleware checks cw_grant)
  /api/leaderboard/scores  →  app:8080 or app:8081  (no nginx-level auth)

app
  port 8080: ungatedMux — no auth on score endpoint
  port 8081: gatedMux  — gate middleware on /play, not on score endpoint
  handler.go: checks X-Leaderboard-Token header (secret visible in JS)
```

### After

```
browser (phone)
  └─ cw_grant cookie (QR gate + score auth — one credential, one trust boundary)

nginx
  /play                    →  auth_request /auth/check  →  app:8080
  /api/leaderboard/scores  →  auth_request /auth/check  →  app:8080

app
  port 8080: single mux
  GET /auth/check: validates cw_grant via gate.Signer + gate.WindowStore → 200/401
  POST /api/leaderboard/scores: no credential check (nginx enforces before forwarding)
```

---

## Files Affected

| File | Change |
|---|---|
| `nginx/nginx.conf` | Add `auth_request` for `/play` and `/api/leaderboard/scores`; add internal `/auth/check` location; remove `app:8081` upstream |
| `app/main.go` | Add `GET /auth/check` handler; remove `gatedMux`, gated listener, `GatedPort`, `leaderboardSecret`, token injection in `handlePlayIndex` |
| `app/internal/leaderboard/handler.go` | Remove `secret` field, `CredentialHeader`, `validCredential()`, credential check in `serveSubmit` |
| `app/internal/leaderboard/leaderboardtest/fake_store.go` | No change needed (store/notifier interfaces unaffected) |
| `app/main_test.go` | Remove token injection test coverage; add `/auth/check` handler tests |
| `docker-compose.yml` | Remove `LEADERBOARD_API_SECRET` and `APP_GATED_PORT` |
| `frontend/game/index.html` | Remove `__LEADERBOARD_TOKEN__` script injection |
| `frontend/game/*.js` | Remove `X-Leaderboard-Token` header from score submission fetch |

---

## Next Steps

1. Run `/speckit-specify` to write a formal spec from this handoff
2. Run `/speckit-plan` to generate research, data-model, and contracts artifacts
3. Run `/speckit-tasks` for the task breakdown
4. Implement — the cleanup tasks (§ Dead Code and Cleanup Required) must be in scope,
   not deferred, to avoid shipping a half-migrated state where both auth paths exist
