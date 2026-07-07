# Research: Player Name Entry, Game Over Score Display, and Leaderboard Score Submission

All items below were open questions in the Technical Context; none remain as NEEDS CLARIFICATION.

## 1. Where the leaderboard write API lives

**Decision**: Add the leaderboard score-write API as a new internal package (`app/internal/leaderboard`)
inside the existing `app` Go service, exposed as new routes on both of `app`'s existing listeners
(`POST /api/leaderboard/scores`). No new compose service, no new Go module, no new deployable.

**Rationale**: The spec's Assumptions read "its own Go-based API" as *a distinct, independently
addressable and documented API surface* (its own OpenAPI document, its own package, its own tests),
not necessarily *a separate network service*. A genuinely separate service would need its own
compose entry, its own port, and — critically — a way for the browser to reach it through the same
public ngrok tunnel that only forwards to `app`'s gated port today (`002-qr-gated-access`
`docker-compose.yml`). Solving that would mean either publishing a second port through ngrok (ngrok
free tier serves one tunnel) or having `app` reverse-proxy to a second service — pure overhead that
buys nothing this feature's requirements ask for, and works against constitution Principle I
(Demo-First Delivery: don't spend effort on invisible internals). Keeping it as a package inside
`app` satisfies every functional requirement (FR-011's OpenAPI doc, FR-012's credential check,
independent testability) while adding zero new moving parts to the compose stack.

**Alternatives considered**:
- **Separate `leaderboard/` Go module + compose service**: rejected for the ngrok single-tunnel
  reason above, and because constitution Principle II favors the smallest set of compose services
  that reaches a demoable state — this feature doesn't need a second service to satisfy its FRs.
  Issue 5 (the future leaderboard *viewing* page, explicitly out of scope here) is a more natural
  point to introduce a second Go app, since it has its own reason to exist independently (a
  separate route/page consumers hit directly) — that decision belongs to that feature's own plan.
- **Fold the new routes into `internal/gate`**: rejected — the spec's own Assumptions explicitly
  call for a distinct API surface rather than piggybacking on an unrelated existing endpoint's
  package; `gate` owns QR/visitor access, which is a different concern from leaderboard writes.

## 2. Whether score submission is coupled to the QR access gate

**Decision**: The new `/api/leaderboard/scores` route is registered on both `ungatedMux` and
`gatedMux` without wrapping it in `gate.Middleware`. Authorization for this route is entirely the
new Score Submission Credential check (FR-012), independent of whether the caller also holds a
valid QR visitor grant cookie.

**Rationale**: The spec's Assumptions are explicit that the player's entered name is "independent
of any QR-access-grant identifier from prior access-gating work" — this feature introduces its own,
separate identity and its own, separate write-protection mechanism. Requiring a QR grant in
addition would silently couple two unrelated security concerns and would break local/ungated play
(`ungatedMux` never has grants at all — the presenter demoing on a laptop has never scanned
anything), which would violate this feature's own requirement that the credential check adds no
visible extra step for the player (FR-014) on the one path (local demo) that never touches the gate
at all.

**Alternatives considered**:
- **Require both a valid grant cookie and the credential on the gated listener**: rejected as
  redundant in practice (a player submitting a score already loaded `/play` through the gate, so
  they already hold a grant) and outright broken on the ungated listener, which has no grant
  concept to check.

## 3. Redis data structure for leaderboard entries

**Decision**: A single Redis Stream, `leaderboard:scores`, appended to via `XADD` with fields
`name` and `score` per entry. No other Redis structure is introduced by this feature.

**Rationale**: FR-007 and FR-008 require every completed attempt to produce its own entry, with no
attempt's write ever overwriting another's — including two attempts under the identical name. A
Sorted Set keyed by name (the structure `crossy.md`'s early sketch anticipated) cannot satisfy
this: `ZADD` on an existing member overwrites that member's score rather than adding a new entry,
which is exactly the overwrite FR-008 forbids. A Stream is naturally append-only, requires no
uniqueness key, preserves arrival order, and needs nothing this feature doesn't already ask for
(no ranking or top-N query is in scope here — FR-015 explicitly excludes a viewing/browsing
surface). This is a deliberate, documented deviation from `crossy.md`'s original sorted-set sketch;
`crossy.md` predates this feature's decision to make entries append-only rather than best-score-kept,
and its leaderboard section should be treated as superseded by this data model for the score-storage
mechanism specifically. A future leaderboard-viewing feature (issue 5) can derive whatever ranked
view it needs (e.g. a sorted set of best-score-per-name) from this stream at read time, or maintain
one incrementally, without this feature having to guess that shape now.

**Alternatives considered**:
- **Sorted Set (member = name, score = points)**: rejected — overwrites on repeat names, directly
  violating FR-007/FR-008.
- **Sorted Set (member = generated unique ID, score = points) + a Hash per ID for the name**: would
  work, but is strictly more moving parts (two structures, an ID-generation step) than a Stream for
  a feature that has no read/ranking requirement yet. Revisit if/when the future viewing feature's
  own research shows it needs sorted-by-score reads badly enough to justify maintaining this
  structure incrementally rather than deriving it from the stream on demand.
- **Plain List (`RPUSH` of JSON-encoded entries)**: works too, but Streams provide the same
  append-only guarantee plus built-in entry IDs and range-read support for free, which a future
  read-side feature is more likely to want than a bare list.

## 4. How the browser gets the write credential (FR-014)

**Decision**: `app` renders `frontend/game/index.html` through a small Go template (replacing the
current raw `http.FileServer` pass-through for that one file only) that injects the configured
credential into an inline `<script>window.__LEADERBOARD_TOKEN__ = "...";</script>` block. All other
static assets (`script.js`, `style.css`, the model file) continue to be served unchanged via
`http.FileServer`. The frontend's score-submission call reads `window.__LEADERBOARD_TOKEN__` and
sends it as a request header.

**Rationale**: FR-014 requires the credential to be obtained and presented with no visible extra
step for the player. Since the game is a static single-page app with no build step and no existing
templating, the smallest change that gets a server-held value into client-side JS without the
player doing anything is server-side template injection into the one HTML entry point, mirroring
how `handlePlayIndex` already serves that file dynamically today (`app/main.go`). This keeps
`script.js` and the other static assets exactly as they are served now (per constitution's existing
"no build step" posture for the frontend).

**Alternatives considered**:
- **A dedicated unauthenticated `GET /api/leaderboard/client-token` endpoint**: rejected —
  functionally equivalent exposure (anyone who can reach the game can read the token either way),
  but adds a network round trip and a route that has no purpose other than handing out the very
  credential meant to gate the next call, which reads as security theater without the injection
  approach's simplicity.
- **Build-time secret injection (bundler/env-replace step)**: rejected — this project's frontend has
  no build step today (plain script tags via importmap, per `crossy.md`'s Technology Stack), and
  introducing one solely for this would be a disproportionate new dependency for one string value.

## 5. Score and name validation

**Decision**: Name: reject if empty after trimming leading/trailing whitespace; cap at 32
characters (truncate client-side before submit; server also rejects/truncates defensively). Score:
must be a non-negative integer; the server rejects negative values, non-integers, and missing
fields with `400 Bad Request`.

**Rationale**: FR-002 and FR-003 need concrete bounds to be testable. 32 characters is a generous
display-name length (comparable to common game leaderboard conventions) that comfortably fits the
Game Over screen's layout without needing text wrapping/truncation logic in the UI itself. Score
must be non-negative because the game's own scoring model (`frontend/game/script.js`) only ever
counts forward progress upward from zero; a negative value can only indicate a malformed or hostile
request, not a legitimate low score.

**Alternatives considered**: None significant — these are direct, low-stakes defaults with no
competing interpretation that changes scope, matching the spec's own guidance to document
reasonable defaults as Assumptions rather than raise them as clarifications.

## 6. Testing approach (constitution Principle III)

**Decision**: The `leaderboard` package's Redis-touching code (the `XADD` write) is tested against
a real Redis via Testcontainers-go, reusing the same module dependency already vendored for
`internal/gate`. Request validation (name/score bounds) and credential checking are pure logic,
covered by standard Go unit/handler tests against an in-memory fake of the same small store
interface the Redis-backed implementation satisfies — the same pattern `internal/gate` already
establishes for `WindowStore`. The end-to-end player flow (name prompt → play → death → Game Over
→ submission → Replay) is validated manually per `quickstart.md` and constitution Principle IV.

**Rationale**: Direct application of the constitution's existing, non-negotiable testing principle
and the precedent already set by `002-qr-gated-access`'s `WindowStore`/fake pattern — no new
judgment call beyond identifying which parts of this feature cross the Redis boundary.

**Alternatives considered**: None — this follows an existing, established project pattern.
