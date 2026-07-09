# Data Model: nginx auth_request for Score Integrity

**Feature**: 016-nginx-auth-score-integrity  
**Date**: 2026-07-09

This feature does not introduce new persistent data entities. All data written to Redis remains
unchanged. This document describes the entities relevant to the auth flow and identifies what
changes in their shape or handling.

---

## Existing Entities (unchanged structure)

### QR Session Grant (`cw_grant` cookie)

The signed cookie that authorises a player's session after scanning the QR code.

| Attribute | Value |
|-----------|-------|
| Cookie name | `cw_grant` |
| Contents | HMAC-signed JSON: `grant_id`, `issued_window_id`, `issued_at` |
| Signing key | `GRANT_COOKIE_SECRET` (server-only, never in the browser) |
| Lifetime | `GRANT_LIFETIME` (default 4 hours) |
| Path | `/` |
| Flags | `HttpOnly`, `Secure`, `SameSite=Lax` |

**Change for this feature**: This entity is unchanged. It was previously used only to gate `/play`
access. After this feature it also gates `/api/leaderboard/scores` — the same cookie, the same
validation, now applied at the nginx layer via the `/auth/check` sub-request.

### QR Window (Redis key, managed by `gate.WindowStore`)

A short-lived Redis key representing the currently active QR scan window. Used during grant
issuance (when a player scans a fresh QR code).

**Change for this feature**: Unchanged. The `/auth/check` endpoint uses `gate.Signer.Verify()`
only, not `gate.WindowStore`. Window lookup is still used by `gate.Middleware` on `/play` during
grant issuance.

### Leaderboard Score Entry (Redis Stream `leaderboard:scores`)

A player name + score pair written to Redis when a score submission is accepted.

**Change for this feature**: Unchanged. The `Handler.store.Write()` call path is not modified.

---

## Retired Entity

### Leaderboard API Secret (`LEADERBOARD_API_SECRET` / `X-Leaderboard-Token`)

Previously: a symmetric secret injected into the game page and sent as a request header on score
submissions.

| Attribute | Before | After |
|-----------|--------|-------|
| Server config | `LEADERBOARD_API_SECRET` env var | **Removed** |
| App field | `Config.LeaderboardAPISecret`, `App.leaderboardSecret` | **Removed** |
| Browser exposure | `window.__LEADERBOARD_TOKEN__` global in page source | **Removed** |
| Wire protocol | `X-Leaderboard-Token: <secret>` request header | **Removed** |
| Validation | `validCredential()` in `handler.go` | **Removed** |

After this feature, no leaderboard-specific credential exists. The `cw_grant` session cookie is
the sole authorisation mechanism for both game access and score submission.

---

## New Internal Endpoint

### `/auth/check` — Grant Validation Sub-request

An HTTP endpoint added to the Go app, exposed only as an nginx internal sub-request.

| Attribute | Value |
|-----------|-------|
| Method | `GET` |
| Path | `/auth/check` |
| Registered on | Single mux (port 8080) |
| Accessibility | `internal;` in nginx — not reachable by external clients |
| Input | `Cookie: cw_grant=<value>` forwarded by nginx |
| Success response | `200 OK`, empty body |
| Failure response | `401 Unauthorized`, empty body |
| Validation logic | `gate.Signer.Verify(cookieValue)` — HMAC + lifetime check |
| Side effects | None — read-only; does not write to Redis, issue cookies, or redirect |

**Validation rules** (inherited from `gate.Signer.Verify`):
- Cookie must be present
- Cookie value must parse as `<base64-payload>.<base64-mac>`
- MAC must match HMAC-SHA256 of payload with the configured secret
- `issued_at` in the payload must be within `GRANT_LIFETIME` of the current time

---

## Mux Route Table (after change)

Single mux on port 8080. Replaces the previous `ungatedMux` (8080) + `gatedMux` (8081) pair.

| Route | Handler | Gate |
|-------|---------|------|
| `GET /` | `handleRootOrAsset` | none |
| `GET /play` | `gate.Middleware(handlePlayIndex)` | `cw_grant` cookie OR `?w=<windowID>` |
| `GET /qr.png` | `handleQRPNG` | none |
| `GET /repo-qr.png` | `handleRepoQRPNG` | none |
| `GET /api/ping` | `handlePing` | none |
| `GET /host` | `handleHost` | none |
| `POST /host/rotate` | `handleHostRotate` | none |
| `POST /api/leaderboard/scores` | `leaderboardHandler` | nginx `auth_request /auth/check` |
| `GET /leaderboard` | `handleLeaderboardPage` | none |
| `/leaderboard-assets/` | file server | none |
| `GET /auth/check` | `handleAuthCheck` | none (internal sub-request only) |

Note: nginx enforces `auth_request /auth/check` for `/api/leaderboard/scores` before the request
reaches the Go app. The Go handler itself performs no credential check.
