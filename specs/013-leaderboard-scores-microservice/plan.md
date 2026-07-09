# Implementation Plan: Leaderboard Scores Microservice

**Branch**: `013-leaderboard-scores-microservice` | **Date**: 2026-07-08 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/013-leaderboard-scores-microservice/spec.md`

## Summary

Extract leaderboard score reads into a new `scores-service` Go microservice that exposes `GET /scores` (JSON standings) and `GET /scores/stream` (SSE). Live updates are triggered by a Redis pub/sub notification that `app` publishes on each successful score write. The `app` service's polling-based GET standings endpoint and all supporting read code are removed; the leaderboard page's standings column is replaced by a new `scores-component.js` React component that subscribes to the SSE stream.

## Technical Context

**Language/Version**: Go 1.25 (matches `commits-service` and `app`)

**Primary Dependencies**:
- `scores-service`: `github.com/redis/go-redis/v9` (Redis client + pub/sub subscriber), `github.com/testcontainers/testcontainers-go` + `testcontainers-go/modules/redis` (boundary tests)
- `app` changes: no new dependencies; existing `github.com/redis/go-redis/v9` used for the new PUBLISH call

**Storage**: Redis (existing shared instance). Reads: `XRANGE leaderboard:scores - +`. Subscribes: `leaderboard:score-updated` pub/sub channel. No new Redis data structures.

**Testing**: Go standard testing + Testcontainers-go for all Redis boundary tests (Principle III, NON-NEGOTIABLE)

**Target Platform**: Linux server, Docker container (`dhi.io/static:*` final image, matching `commits-service`)

**Project Type**: web-service (microservice)

**Performance Goals**: SSE push latency ≤ 1 s after pub/sub notification received; `GET /scores` response ≤ 200 ms p95 at booth scale

**Constraints**: No HTML output; CORS headers required (browser hits scores-service directly); write timeout disabled on SSE handler (long-lived connections); ephemeral Redis (no persistence required)

**Scale/Scope**: Booth-scale — at most a few hundred score stream entries, tens of concurrent SSE connections per session

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Demo-First Delivery | **PASS** | Live standings that update without page reload is a direct improvement to the booth demo. The current polling approach (4 s interval, visible lag) is replaced by event-driven SSE push. |
| II. Compose-Orchestrated Reproducibility | **PASS** | `scores-service` added to `docker-compose.yml`; starts with `docker compose up`; no host-side setup beyond Docker Desktop and git. |
| III. Testcontainers Over Mocks (NON-NEGOTIABLE) | **GATE** | `scores-service` Redis store tests (XRange read, pub/sub subscribe) MUST use Testcontainers-go with a real Redis container. `app` pub/sub publish tests must also use real Redis. No mock-based Redis testing is acceptable. |
| IV. Visible-in-the-Browser Definition of Done | **GATE** | Feature is not done until standings update live in the browser (observed, not just test-passing) when a score is submitted via the game against the compose stack. |
| V. Vendored-Code Hygiene (NON-NEGOTIABLE) | **PASS** | `scores-component.js` is authored in-house — no attribution needed. React bundles (`react.production.min.js`, `react-dom.production.min.js`) are already vendored. No new third-party assets. |
| Technology Stack | **PASS** | New Go module for scores-service (within "Backend: Go"); Redis pub/sub (explicitly in-scope); React component (within Leaderboard React carve-out); `dhi.io/golang` and `dhi.io/static` base images (matching `commits-service`). No stack violations. |

*Post-Phase 1 re-check*: No design decisions introduced after research change this assessment. No Complexity Tracking required.

## Project Structure

### Documentation (this feature)

```text
specs/013-leaderboard-scores-microservice/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   ├── scores-openapi.yaml       # REST API contract
│   └── scores-sse-contract.md    # SSE wire format contract
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

**New microservice**:

```text
scores-service/
├── Dockerfile
├── go.mod                         # module crossywhale/scores-service
├── go.sum
├── main.go                        # config, server setup, mux wiring
└── internal/
    └── scores/
        ├── handler.go             # GET /scores and GET /scores/stream
        ├── handler_test.go        # handler unit tests
        ├── store.go               # Redis: XRange read, best-per-player aggregation, pub/sub subscribe
        └── store_test.go          # Testcontainers Redis boundary tests
```

**Modified: `app` service**:

```text
app/
├── main.go                        # +SCORES_SERVICE_URL config; +scoresServiceURL field;
│                                  #  update handleLeaderboardPage data struct
└── internal/
    └── leaderboard/
        ├── handler.go             # remove serveList + GET case; add ScoreNotifier injection;
        │                          #  call notifier.Notify() after successful store.Write()
        ├── handler_test.go        # remove GET standings tests; add Notify call tests
        ├── store.go               # remove Top() from ScoreStore interface + RedisScoreStore;
        │                          #  remove RankTop(); add Notify() to ScoreStore interface
        │                          #  (or new ScoreNotifier interface); implement in RedisScoreStore
        └── store_test.go          # remove Top() Testcontainers tests; add Notify() tests
```

**Modified: leaderboard page template** (in `app/main.go`):

```text
leaderboardPageTemplate changes:
  - Remove: <ul id="standings">, <p id="status">, and the vanilla-JS polling block (~lines 461–515)
  - Add:    <div id="scores-root"></div>
  - Add:    <script src="/leaderboard-assets/scores-component.js">
  - Add:    ReactDOM.createRoot(#scores-root).render(<ScoresComponent scoresServiceURL="{{.ScoresServiceURL}}" />)
  - Update: handleLeaderboardPage data struct to include ScoresServiceURL
```

**Modified: `docker-compose.yml`**:

```text
docker-compose.yml:
  + scores-service service definition (build: scores-service/, port 8083:8083)
  + SCORES_SERVICE_URL env var on app service (default http://localhost:8083)
  + SCORES_LIMIT env var on scores-service (default 10)
  + REDIS_ADDR env var on scores-service
```

**New frontend asset**:

```text
frontend/leaderboard/
└── scores-component.js            # React component: SSE subscription, standings render,
                                   #  empty-state, fallback polling, reconnect handling
```

**Structure Decision**: Multi-service layout matching the established pattern. `scores-service/` mirrors `commits-service/` exactly in structure (standalone Go module, `internal/scores/` package, DHI-based Dockerfile). App changes are confined to `app/internal/leaderboard/` and the leaderboard page template constant in `main.go`.

## Key Design Decisions

### Redis pub/sub flow

```
app serveSubmit
  └─ store.Write()      → XADD leaderboard:scores
  └─ notifier.Notify()  → PUBLISH leaderboard:score-updated ""
                                    │
scores-service subscribe goroutine  │
  └─ receives message ◄─────────────┘
  └─ reads XRANGE leaderboard:scores - +
  └─ aggregates best-per-player in Go map
  └─ ranks, caps at SCORES_LIMIT
  └─ broadcasts SSE "standings" event to all open connections
```

### Best-per-player aggregation

Full stream read (`XRANGE - +`) on each pub/sub notification → group by `name` → `max(score)` per player → sort descending → cap at `SCORES_LIMIT`. In-process; no additional Redis data structures. Correct at booth scale (≤ few hundred entries).

### ScoreNotifier interface

```go
// ScoreNotifier is the seam for publishing score-change notifications.
// Implemented by RedisScoreStore; stubbed in handler tests.
type ScoreNotifier interface {
    Notify(ctx context.Context) error
}
```

Notification failure is logged but does not fail the HTTP response — a missed pub/sub event delays the SSE update but does not affect score recording.

### SSE connection lifecycle

- On connect: emit one `standings` event immediately (current state from Redis).
- On pub/sub message: emit `standings` event to all open connections.
- On client disconnect: handler returns; goroutine exits cleanly.
- Write timeout: disabled (0) on the HTTP server for the SSE handler path (matching `commits-service`).

### scores-component.js design

- `EventSource` on `scoresServiceURL + '/scores/stream'`; listens for `standings` events.
- On error: exponential backoff reconnect (native `EventSource` behaviour).
- Fallback: if `EventSource` fails permanently (e.g., after N reconnect attempts), switch to `setInterval` polling `GET /scores` every 5 s.
- Empty state: renders "No scores yet — be the first to play!" when `standings.length === 0`.
- Renders plain `React.createElement` calls (no JSX), matching `commits-component.js` style.

## Complexity Tracking

> No constitution violations — this section is intentionally empty.
