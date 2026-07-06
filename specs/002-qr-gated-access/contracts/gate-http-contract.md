# Contract: Gate HTTP Surface

Two listeners on the `app` service. Which listener a request arrives on is what determines whether
the gate applies (see `research.md` §2) — routes are not duplicated by choice, only by which
listener mounts them.

## Ungated listener (local only; only reachable via the presenter's own published port)

| Route | Method | Behavior |
|-------|--------|----------|
| `/play` and its static assets | GET | Always serves the Crossy Whale game, no gate check, no cookie required. Satisfies FR-004. |
| `/qr.png` | GET | Returns the current QR code as a PNG image encoding `https://<public-host>/play?w=<current window_id>`. 503 with a plain-text explanation if no window is currently active (e.g., before first activation). |
| `/host` | GET | Minimal HTML page embedding `/qr.png` and a "Rotate QR" control. Local-only per FR-002/FR-007 — never exposed on the gated listener. |
| `/host/rotate` | POST | Generates a fresh window (or the first one, if none exists yet), replacing any current one. Returns 303 back to `/host`. Idempotent in effect (always ends with exactly one active window). |

## Gated listener (only reachable via the `ngrok` service)

| Route | Method | Behavior |
|-------|--------|----------|
| `/play` | GET | **With a valid grant cookie**: serves the game, no different from the ungated route. **Without a valid grant cookie, but with a `w` query parameter matching the current window**: mints a grant cookie (`Set-Cookie`, HttpOnly, Secure, SameSite=Lax) and responds `302 Found` to `/play` (token stripped from the URL). **Otherwise**: responds `403 Forbidden` with a short HTML page telling the visitor to scan the QR code; does not serve any game asset. |
| `/play/*` (static assets: script, styles, model) | GET | Same authorization as `/play` above, evaluated per request — a visitor who never completed `/play`'s grant step cannot fetch assets directly either. Assets are covered by the same valid-cookie check; they do not accept a `w` token themselves (the entry point for a fresh grant is always `/play`). |
| `/qr.png`, `/host`, `/host/rotate` | — | Not mounted on this listener at all; a request for these paths on the gated listener gets the same `404 Not Found` any undefined route would. |

## Cookie

| Attribute | Value |
|-----------|-------|
| Name | `cw_grant` |
| Value | Base64 of `{grant_id, issued_window_id, issued_at}` plus an HMAC-SHA256 signature, server-verified on every gated request. |
| Flags | `HttpOnly`, `Secure`, `SameSite=Lax` |
| Lifetime | Fixed duration set independently of any QR Access Code's TTL (see `research.md` §4); not renewed or extended by continued use. |

## Error responses

| Situation | Response |
|-----------|----------|
| No grant cookie, no/invalid `w` token | `403 Forbidden`, HTML: "Scan the QR code to play." |
| No QR Access Code active at all (fail closed, FR-009) | Same `403 Forbidden` as above — a visitor cannot distinguish "expired" from "never started" from the response, which is intentional (nothing to leak either way). |
| Grant cookie present but signature invalid (tampered/corrupted) | Treated as no grant cookie — falls through to the `w`-token check above. |
