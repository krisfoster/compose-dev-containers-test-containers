# Research: QR-Gated Public Access to Crossy Whale

All items below were open questions in the Technical Context; none remain as
NEEDS CLARIFICATION.

## 1. Why this feature requires a Go backend now

**Decision**: Introduce the first Go service in this repo (`app/`), backed by Redis, and retire
the `webserver` (nginx) compose service from `001-host-webapp-ngrok`. The Go app serves
`frontend/game/` directly (static files) in addition to owning the QR-gate logic.

**Rationale**: The gate (FR-003, FR-009) requires a per-request authorization decision — is this
visitor holding a valid access grant, and if not, does the request carry a currently-valid QR
window token? That is server-side logic beyond file serving, so the constitution's Interim Static
Hosting carve-out (added in `001-host-webapp-ngrok`) no longer applies to this surface, and its own
sunset clause requires retiring the interim nginx service in the same phase that introduces the Go
backend. This is not a new stack dependency — the constitution already fixes "Backend: Go" and
"State and pubsub: Redis" — this feature is simply the first to need them.

**Alternatives considered**:
- **nginx `auth_request` + a tiny sidecar**: keeps nginx serving files and adds a small
  authorization subrequest to a sidecar. Rejected: it splits gate logic across two services and
  two config languages (nginx config + sidecar code) for no benefit, when Go's standard library
  serves static files in a few lines. It also fights the constitution's explicit sunset
  instruction rather than resolving it.
- **Keep nginx behind the Go app (Go proxies to nginx after the gate passes)**: rejected for the
  same reason — `http.FileServer` makes the proxy hop pure overhead with no functional gain, and
  still leaves two services to maintain instead of one.

## 2. Distinguishing "local" from "public" traffic (FR-004)

**Decision**: The Go app listens on two ports: an ungated port that only the presenter's own
`docker compose up` port-mapping reaches (no gate applied, replaces today's local `webserver`
port), and a second, gated port that only the `ngrok` service points at
(`ngrok http app:<gated-port>`). Routing is decided at listener level, not by inspecting headers.

**Rationale**: Header-based heuristics (Host header, `X-Forwarded-For`, source IP ranges) are
fragile — booth wifi/LAN setups vary, and a presenter demoing over a LAN IP rather than
`localhost` would trip a heuristic gate. Two listeners make the local-vs-public distinction a fact
about network topology (which port ngrok was told to forward to) rather than something inferred
per-request, and it costs nothing extra: `net/http` support for a second `ListenAndServe` on
another port is a few lines, and Compose already publishes exactly one webserver port today.

**Alternatives considered**:
- **Single port, gate by Host header matching the known ngrok domain**: requires querying ngrok's
  own API (port 4040) to learn the current public hostname and keeping that in sync; adds a
  runtime dependency from the gate's correctness on ngrok's inspection API being reachable.
  Rejected as more moving parts for a less reliable signal than "which port did this arrive on."
- **Single port, gate by source IP (RFC1918/loopback bypass)**: breaks for a presenter demoing
  over real venue wifi where their own laptop's browser IP is also non-loopback, and breaks
  FR-004's "local machine" guarantee outright in that setup.

## 3. QR window state (FR-006, FR-007, FR-009)

**Decision**: A single Redis string key, `access:window:current`, holds the current window's
opaque random ID and carries a TTL equal to the window's validity period (default 15 minutes,
overridable via an environment variable). The QR code encodes
`https://<ngrok-host>/play?w=<window_id>`.

- **Automatic expiry (FR-006)**: when the TTL lapses, Redis deletes the key on its own — no
  scheduled job needed. A missing key means "no valid window," which the gate treats as reject.
- **Manual rotation (FR-007)**: the presenter's rotate action overwrites the key with a freshly
  generated window ID and a full TTL, using a single `SET` — the old ID stops being able to match
  the very next lookup, with no separate delete/invalidate step.
- **Fail closed (FR-009)**: on first run, the key does not exist until the presenter's host page
  is opened for the first time (which generates the initial window), so the public gated port
  rejects everyone by default until that happens.

**Rationale**: A single current-pointer key is sufficient because the gate only ever needs to
answer "does this token match the window that's active right now?" — it never needs to look up a
historical/rotated-out window's details. This is simpler than tracking each window as its own
TTL'd key (as sketched in this project's early design notes for a fuller join/session system) and
fully satisfies this feature's requirements without extra Redis structures that nothing here reads.

**Alternatives considered**:
- **Separate `window:<id>` keys plus a `window:current` pointer** (matching `crossy.md`'s original
  sketch): useful once a fuller join flow needs to answer "was this ever a valid window" for
  session bookkeeping, but nothing in this feature's scope needs that; deferred to whichever future
  feature builds the join/leaderboard flow, which can add it without touching this feature's gate.

## 4. Visitor access grant (FR-005, FR-008, FR-011)

**Decision**: On a request that carries a valid, current `w` token and no existing grant, the app
mints a random grant ID (UUIDv4), packages it with the issuing window ID and issue time into a
cookie payload, HMAC-SHA256-signs it with a server-held secret, and sets it as an HttpOnly, Secure,
SameSite=Lax cookie with a fixed multi-hour lifetime (independent of the QR window's own, shorter
TTL). The app then redirects to the clean `/play` URL (token stripped) so the token never lingers
in browser history/bookmarks. Subsequent requests are authorized purely by verifying the cookie's
signature — no Redis lookup, no server-side grant record.

**Rationale**: The requirement is only that each visitor's access, once granted, keeps working
independent of later rotation/expiry (FR-008), and carries a unique, stable ID a future leaderboard
feature can attribute scores to (FR-011). A signed, stateless cookie satisfies both with no added
storage: the ID is generated once and never needs to be looked up server-side for the gate to keep
working. This keeps Redis reserved for window state now and leaderboard data later, rather than
also holding a per-visitor session record this feature doesn't otherwise need.

**Alternatives considered**:
- **Redis-backed session** (a `session:<id>` hash, per `crossy.md`'s fuller design): the right
  choice once a real join flow (name, emoji) needs to persist visitor-entered data — but this
  feature has no such data to persist, so it would add a write-per-visitor and a lookup-per-request
  to Redis for no behavior this feature's requirements ask for. The chosen grant ID is
  forward-compatible: a future join feature can read the same cookie and use its grant ID as the
  key for a proper Redis session, so this isn't a decision that has to be undone later.

## 5. QR code image generation (FR-001, FR-002)

**Decision**: Generate the QR PNG server-side in Go using a small, actively-used, MIT-licensed
QR-encoding library (e.g. `skip2/go-qrcode`), served from a local-only route (`GET /qr.png`) that
only exists on the ungated listener, plus a minimal local-only `GET /host` page embedding it with a
"Rotate" button.

**Rationale**: Server-side generation means the encoded URL (including the current window token)
is always in sync with the value the gate itself is validating against — no risk of a
client-side-generated code drifting from server state. A small Go module dependency for QR
encoding is a standard library dependency (tracked via `go.mod`/`go.sum`), not vendored/copied code
or a third-party asset, so it does not require an `ATTRIBUTION.md` entry — consistent with how the
`nginx`/`ngrok` base images were treated as plain dependencies in `001-host-webapp-ngrok`.

**Alternatives considered**:
- **Client-side QR generation (JS library) on a wall page**: deferred — the wall page itself
  (leaderboard, "now playing" indicator) is explicitly out of scope for this feature per its
  Assumptions; building only its QR-display sliver now and the rest later would split one page's
  implementation across two features for no benefit. Server-side PNG works standalone today and
  slots into the future wall page as an `<img>` tag either way.
- **Third-party hosted QR generation API**: rejected outright — it would leak the access URL
  (including the live window token) to an external service, undermining the access control this
  feature exists to provide, and adds an internet dependency for something Go can do locally in
  microseconds.

## 6. Testing approach for the gate (constitution Principle III)

**Decision**: Pure logic (cookie signing/verification, window-token comparison) is covered by
standard Go unit tests. Anything that reads or writes `access:window:current` in Redis is tested
against a real Redis via Testcontainers-go, per the constitution's non-negotiable Principle III —
no mocked Redis client for those paths. The end-to-end scan-to-game flow (camera behavior, actual
phone hardware) is validated manually per `quickstart.md` and constitution Principle IV, since a
camera scan itself isn't something a Go test can exercise.

**Rationale**: This directly follows the constitution's existing testing principles; there is no
new judgment call here beyond identifying which parts of this specific feature cross the Redis
boundary.

**Alternatives considered**: None — this is a direct application of an existing, non-negotiable
project principle, not an open design choice.
