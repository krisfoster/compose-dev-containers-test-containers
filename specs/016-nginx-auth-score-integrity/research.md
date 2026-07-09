# Research: nginx auth_request for Score Integrity

**Feature**: 016-nginx-auth-score-integrity  
**Phase**: 0 — pre-design  
**Date**: 2026-07-09

---

## 1. nginx `auth_request` module

**Decision**: Use nginx's standard `ngx_http_auth_request_module` — available in the `nginx:alpine` base image used by this project's nginx service — to gate `/api/leaderboard/scores` before the request reaches the Go app.

**How it works**:
- `auth_request <location>` makes a synchronous internal sub-request to `<location>` for each incoming request.
- If the sub-request returns 2xx, the original request is forwarded to `proxy_pass`.
- If it returns 401 or 403, nginx returns that status to the client immediately; the upstream never sees the request.
- The sub-request location must be marked `internal;` to prevent direct external access.

**Cookie forwarding**: By default, `auth_request` sub-requests do NOT include the original request body, but they DO include request headers. The `Cookie` header must be explicitly forwarded via `proxy_set_header Cookie $http_cookie;` in the `auth_request` target location so the Go app can read the `cw_grant` cookie.

**Rationale**: This is the standard nginx pattern for delegating auth to an upstream service. It requires no additional module installation and is idiomatic for the nginx version in use.

**Alternatives considered**:
- Lua scripting (OpenResty): not in the constitution-approved routing layer; would require a stack change.
- Validating the cookie inside nginx using JS module: puts business logic in the routing layer — constitution violation.

---

## 2. `/auth/check` endpoint design

**Decision**: Add a new `GET /auth/check` handler in `app/main.go`. Register it on the single (ungated) mux at port 8080. It is accessible only as a nginx internal sub-request.

**Behavior**:
- Read the `cw_grant` cookie from the request.
- Call `gate.Signer.Verify(cookieValue)` — the same HMAC + lifetime check already used by `gate.Middleware`.
- If the grant is valid: return `200 OK` with an empty body.
- If the cookie is absent, malformed, HMAC-invalid, or expired: return `401 Unauthorized` with an empty body.

**Method**: The handler accepts `GET` (the method nginx uses for `auth_request` sub-requests). Other methods return 405.

**What it does NOT do**: It does not issue or refresh grants. It does not check the window query parameter (`?w=<windowID>`). Its role is purely to validate an already-issued cookie — binary pass/fail.

**Rationale**: Reusing `gate.Signer.Verify()` keeps the validation logic in one place. The endpoint is thin — no new logic, just a new HTTP surface for the existing check.

---

## 3. `/play` gating — design decision

**Issue identified**: The handoff's "After" architecture diagram shows `/play` also behind `auth_request /auth/check`. This is incompatible with the QR grant issuance flow:

1. A fresh QR scan yields `GET /play?w=<windowID>` with no `cw_grant` cookie.
2. `auth_request /auth/check` would check the cookie → not found → return 401.
3. nginx would reject the request before `handlePlayIndex` runs.
4. The player never receives the grant cookie; the grant issuance loop breaks.

The `auth_request` sub-request mechanism cannot set cookies on the main response. Grant issuance requires `gate.Middleware` (or equivalent logic) to run on the main request, not in a sub-request.

**Decision**: `/play` is NOT gated via `auth_request`. Instead:
- The single mux registers `/play` with `gate.Middleware` directly (same logic as current `gatedMux`).
- nginx routes `/play` to `app:8080` with no `auth_request`.
- Cookie and `Set-Cookie` headers are forwarded so grant issuance works normally.
- `App.gate` field is retained (still used for `/play`'s middleware).

**Effect on handoff "What to remove" table**: `App.gate` (field) is NOT removed — it is still used. All other items in the handoff's removal table are still removed as planned.

**Rationale**: The feature goal is score integrity, not a rewrite of the play-gate flow. Score integrity is fully achieved by `auth_request` on `/api/leaderboard/scores`. Changing `/play` gating to `auth_request` would require refactoring how grant issuance works (separating the token-redemption step from the game-page-serving step), which is a materially larger change than the feature intends.

---

## 4. Two-listener removal

**Decision**: In scope. Remove the `gatedMux()` method and the second `http.ListenAndServe` goroutine. The single mux (currently `ungatedMux()`) absorbs all routes, with `gate.Middleware` applied to `/play` explicitly.

**Route consolidation**:

| Route | Before (single mux port 8080) | After (single mux port 8080) |
|-------|-------------------------------|------------------------------|
| `/` | handleRootOrAsset (no gate) | unchanged |
| `/play` | handlePlayIndex (no gate) | gate.Middleware(handlePlayIndex) |
| `/qr.png` | handleQRPNG | unchanged |
| `/repo-qr.png` | handleRepoQRPNG | unchanged |
| `/api/ping` | handlePing | unchanged |
| `/host` | handleHost | unchanged |
| `/host/rotate` | handleHostRotate | unchanged |
| `/api/leaderboard/scores` | leaderboardHandler (token check in Go) | leaderboardHandler (no token check; nginx enforces via auth_request) |
| `/leaderboard` | handleLeaderboardPage | unchanged |
| `/leaderboard-assets/` | file server | unchanged |
| `/auth/check` | (does not exist) | handleAuthCheck (new) |

**Config removed**: `GatedPort`, `APP_GATED_PORT`, `Config.GatedPort`, second `http.ListenAndServe`.

**Rationale**: With gate middleware applied per-route on the single mux, the second listener is purely redundant. Removing it simplifies the startup, eliminates a config entry, and removes a compose environment variable.

---

## 5. `LEADERBOARD_API_SECRET` removal

**Decision**: Remove entirely from all surfaces listed in the handoff. No empty-string fallback, no dead config entry, no commented-out code.

**Surfaces to clean**:

| Surface | Change |
|---------|--------|
| `Config.LeaderboardAPISecret` | Remove field + `envOr("LEADERBOARD_API_SECRET", ...)` |
| `App.leaderboardSecret` | Remove field + all assignments |
| `handlePlayIndex` | Remove `LeaderboardToken` injection + struct |
| `leaderboard.NewHandler(...)` | Remove `secret` argument (second param) |
| `leaderboard/handler.go` | Remove `secret` field, `CredentialHeader`, `validCredential()`, credential check in `serveSubmit` |
| `docker-compose.yml` | Remove `LEADERBOARD_API_SECRET` env entry |
| `frontend/game/index.html` | Remove `window.__LEADERBOARD_TOKEN__` script tag |
| `frontend/game/script.js` | Remove `X-Leaderboard-Token` header from score fetch |

The `index.html` template variable `{{.LeaderboardToken}}` is removed, making the file a static HTML file (no template execution needed in `handlePlayIndex`). However, `handlePlayIndex` still renders `index.html` via `html/template` — since the template now has no variables, `tmpl.Execute(w, nil)` works without any data struct.

---

## 6. Frontend score submission

**Decision**: After removing `X-Leaderboard-Token`, the score submission fetch in `script.js` sends no secret header. The `cw_grant` cookie is automatically included by the browser (same-origin, `SameSite: Lax` allows it on top-level navigations and same-site requests). nginx validates the cookie before forwarding.

**Rationale**: The browser includes cookies automatically on same-origin requests. No JS change is needed to "add" the cookie — only to remove the now-unnecessary header.

---

## 7. Constitution check — "gated internal port" clause

**Clause**: "Gate enforcement MUST remain in the Go app and is reached by proxying to the Go gated internal port."

**Analysis**: This clause was written for feature 014 to describe the architecture at that point. After this feature:
- Gate enforcement remains in the Go app ✓ (`gate.Signer.Verify` in `/auth/check` and `gate.Middleware` on `/play`)
- The "gated internal port" (8081) is removed; score-submission auth is instead reached via nginx `auth_request` sub-request to port 8080

**Finding**: A PATCH-level constitution amendment is needed to update the "gated internal port" clause to reflect the post-016 model. The amendment scope: the clause should describe the two mechanisms — `auth_request` sub-request to `/auth/check` (for score submission) and gate middleware on `/play` — rather than a dedicated second port.

**This amendment is a prerequisite before ship**. It does not block task execution, but the Complexity Tracking section of the plan records the violation and justification.

---

## 8. Test impact

**Tests to remove** (rely on gated mux or token injection):

| Test | Reason |
|------|--------|
| `TestHandlePlayIndexInjectsLeaderboardToken` | Token injection removed |
| `TestGatedPlayRejectsWithNoGrantOrToken` | `gatedMux()` removed |
| `TestGatedPlayAllowsValidToken` | `gatedMux()` removed |
| `TestGatedListenerDoesNotExposeHostRoutes` | `gatedMux()` removed |
| `TestHandleLeaderboardPageOnBothListeners` | Only one listener after change |
| `TestOldCommitsEndpointRemoved` (gated mux assertion) | Gated mux assertion no longer valid |
| `TestLoadConfigDefaults` (LEADERBOARD_API_SECRET, APP_GATED_PORT) | Config fields removed |
| `TestLoadConfigReadsOverrides` (LEADERBOARD_API_SECRET) | Config field removed |
| `testLeaderboardSecret`, `testIndexHTMLTemplate`, `testIndexHTMLRendered` constants | Used only by removed tests |

**Tests to update**:

| Test | Change |
|------|--------|
| `TestUngatedPlayRequiresNoGate` | Body assertion: rendered page no longer contains the token; assert content-type and 200 status only |
| `TestHandleRootServesGettingStartedPage` | Still valid; the `__LEADERBOARD_TOKEN__` negative assertion remains valid (landing page still must not be the game file) |
| `newTestApp` | Remove `leaderboardSecret` field, remove secret arg from `leaderboard.NewHandler`, add `/play` registration with gate middleware on test mux |
| `TestOldCommitsEndpointRemoved` | Remove the gated mux assertion; keep ungated mux assertion |

**Tests to add**:

| Test | What it covers |
|------|----------------|
| `TestHandleAuthCheckWithValidCookie` | 200 for a correctly signed, unexpired cookie |
| `TestHandleAuthCheckWithNoCookie` | 401 when no `cw_grant` cookie present |
| `TestHandleAuthCheckWithExpiredCookie` | 401 when signer is configured with zero/negative lifetime |
| `TestHandleAuthCheckWithInvalidCookie` | 401 for tampered/malformed cookie value |
| `TestHandleAuthCheckRejectsNonGet` | 405 for POST to `/auth/check` |
| `TestSingleMuxGatesPlayWithMiddleware` | `/play` on the single mux requires a valid grant (replaces `TestGatedPlayRejectsWithNoGrantOrToken`) |
| `TestSingleMuxIssuesGrantOnValidToken` | `/play?w=<windowID>` issues grant and redirects (replaces `TestGatedPlayAllowsValidToken`) |
| `TestScoreSubmissionNoTokenRequired` | POST to `/api/leaderboard/scores` succeeds without `X-Leaderboard-Token` header |
