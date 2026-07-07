# Research: Leaderboard Display Page

All items below were open questions in the Technical Context; none remain as NEEDS CLARIFICATION.

## 1. How top-N standings are computed from the append-only Stream

**Decision**: The read endpoint performs a bounded `XRange` read over the entire
`leaderboard:scores` Stream (booth scale — at most a few hundred entries per event, per
`plan.md`'s Scale/Scope), decodes each entry's `name`/`score`, sorts in-process by score descending
with the Stream entry ID as a descending tiebreaker (higher ID = more recently written), and
returns the first N.

**Rationale**: 003-leaderboard-score-submission's `research.md` §3 deliberately chose a Stream
specifically because this feature had no ranking requirement *yet* and could "derive whatever
ranked view it needs ... from this stream at read time, or maintain one incrementally, without this
feature having to guess that shape now." At booth scale, reading the whole stream and sorting in Go
on every request is trivially fast (hundreds of entries, well under any latency budget from
FR-006's refresh cadence) and requires zero new Redis structures, no dual-write consistency concern,
and no migration if the write side ever changes. The Stream ID tiebreaker directly satisfies the
spec's edge case ("two entries with the exact same score ... a consistent tiebreaker, e.g. most
recent first").

**Alternatives considered**:
- **Maintain a Sorted Set (`ZADD`) alongside the Stream, updated on every write**: would give O(log
  n) reads instead of O(n), but requires 003's write path to change (a second write per submission,
  a new failure mode to handle if the two writes disagree) for a performance concern this feature's
  Scale/Scope doesn't have. Revisit if a future event's scale actually makes the full-stream read
  measurably slow.
- **Cap the `XRange` read itself (e.g. only the most recent 200 entries) instead of reading
  everything**: unnecessary at current scale, and risks silently dropping a high-scoring early entry
  from a long-running event. Deferred until real scale data says otherwise.

## 2. Whether the leaderboard page is a separate Go service ("standalone app") or a route on `app`

**Decision**: The leaderboard page is a new route (`GET /leaderboard`) on the existing `app` Go
service, registered on both listeners, rendered the same way `handleHost` renders the presenter's
host page (an inline HTML/JS response, no template file, no build step). No new compose service, Go
module, or deployable is introduced.

**Rationale**: Issue 5's "standalone Go-based app/page" wording is ambiguous between "a separate
deployable service" and "a dedicated page, distinct from the game's own page" — and 003's own
research explicitly left this decision to this feature's plan. Two independent signals resolve it
toward a route, not a new service: (1) the project's original design doc (`crossy.md`) describes
the wall/leaderboard view as "a separate route on the same server," and (2) 003 already established
the precedent of avoiding a second Go service for the adjacent leaderboard-write feature, citing the
same constitution principles (I: don't spend effort on invisible internals; II: smallest set of
compose services that reaches a demoable state) that apply equally here. A genuinely separate
service would also need its own compose entry and a way to reach it through the same single ngrok
tunnel `app` already occupies — solvable, but pure overhead this feature's requirements don't ask
for.

**Alternatives considered**:
- **New `leaderboard-web/` Go module + compose service**: rejected for the reasons above; nothing in
  the spec's requirements (FR-001 "separate and standalone from the game itself") requires a
  separate *process* — it's satisfied by a distinct route/page, exactly as `/host` already is
  distinct from `/play` within the same service.

## 3. Read endpoint authorization and route placement

**Decision**: `GET /api/leaderboard/scores` is added as a second method branch on the existing
`leaderboard.Handler` (which already handles `POST` for writes), with no credential check on the
`GET` branch, registered on both `ungatedMux` and `gatedMux` — mirroring exactly how the existing
`POST` route is already registered on both without gate middleware.

**Rationale**: Direct application of the Clarifications recorded in `spec.md` (FR-011, FR-013): the
page and its data have no gating step. Reusing the same `Handler`/route (rather than a new type or
path) keeps one place owning the `/api/leaderboard/scores` resource, consistent with how the same
path already serves as the single leaderboard write surface — this is the same resource, a
different HTTP verb, which is exactly what REST method dispatch is for.

**Alternatives considered**:
- **A separate path, e.g. `GET /api/leaderboard`**: rejected — no requirement calls for a
  path split, and keeping one path for both verbs is one less route to document and reason about.
- **Requiring the same shared credential as `POST`**: rejected per the Clarifications answer (open
  read, matching the ungated page) — see `spec.md` Clarifications session 2026-07-07.

## 4. Client-side refresh mechanism

**Decision**: The leaderboard page's inline JS polls `GET /api/leaderboard/scores` on a fixed
interval (a few seconds) using `fetch`, replacing the rendered list on success and leaving the
existing DOM untouched on failure (FR-007), with no WebSocket or Server-Sent Events.

**Rationale**: The spec (and issue 5's own wording) calls for "polling/calling the leaderboard API,"
not a push mechanism. Polling is the simplest option that satisfies SC-001's 10-second bound with
comfortable margin, requires no new server-side connection state, and matches this feature's
"no build step" frontend posture (a WebSocket client would still be simple, but buys nothing here
that polling doesn't already deliver, and `crossy.md`'s original WebSocket sketch for live updates
was never implemented in any shipped feature so far — introducing it here, for a display page,
would be new complexity with no corresponding requirement).

**Alternatives considered**:
- **WebSocket push** (as `crossy.md`'s original sketch imagined): rejected as unwarranted complexity
  for this feature's actual requirements — no FR or SC calls for sub-second updates, and no other
  shipped feature has introduced the WebSocket hub `crossy.md` originally sketched.

## 5. Testing approach (constitution Principle III)

**Decision**: The new Redis-touching read method (`XRange` + decode) is tested against a real Redis
via Testcontainers-go, extending `store_test.go`'s existing pattern (same container helper). Ranking,
tie-breaking, and limit-bounding logic are pure, covered by handler tests against
`leaderboardtest.FakeScoreStore` extended with the same read method. The page's auto-refresh and
stale-data-on-failure behavior are validated manually per `quickstart.md`, per constitution
Principle IV.

**Rationale**: Direct continuation of the pattern 003 already established for this exact package —
no new judgment call beyond identifying which parts of this feature's code cross the Redis boundary.

**Alternatives considered**: None — this follows an existing, established project pattern.
