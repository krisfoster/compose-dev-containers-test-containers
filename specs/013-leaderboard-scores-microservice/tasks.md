# Tasks: Leaderboard Scores Microservice

**Input**: Design documents from `specs/013-leaderboard-scores-microservice/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓, quickstart.md ✓

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no shared dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Exact file paths included in all descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize the new `scores-service` Go module and Dockerfile before any implementation begins

- [x] T001 Initialize `scores-service/go.mod` as standalone Go module: `module crossywhale/scores-service`, `go 1.25`, with `require` entries for `github.com/redis/go-redis/v9`, `github.com/testcontainers/testcontainers-go`, and `github.com/testcontainers/testcontainers-go/modules/redis`; run `go mod tidy` inside `scores-service/` to populate `go.sum`
- [x] T002 [P] Create `scores-service/Dockerfile` — multi-stage build mirroring `commits-service/Dockerfile`: stage 1 uses `dhi.io/golang:1.25-alpine-dev AS build`, copies `go.mod`/`go.sum`, runs `go mod download`, copies source, builds binary to `/out/scores-service`; stage 2 uses `dhi.io/static:20260611-alpine3.24`, copies binary as `/scores-service`, sets `ENTRYPOINT ["/scores-service"]`

**Checkpoint**: `scores-service/` directory has `go.mod`, `go.sum`, and `Dockerfile`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: App pub/sub publish path and the scores-service Redis store — both must be complete before any user story can be independently tested

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T003 Add `ScoreNotifier` interface to `app/internal/leaderboard/store.go` with a single method `Notify(ctx context.Context) error`; implement `Notify()` on `RedisScoreStore` by calling `s.client.Publish(ctx, "leaderboard:score-updated", "").Err()`; do not modify the `ScoreStore` interface (keep `ScoreNotifier` separate for testability)
- [x] T004 Update `app/internal/leaderboard/handler.go` — add `notifier ScoreNotifier` field to `Handler` struct; update `NewHandler(store ScoreStore, secret string, notifier ScoreNotifier)` signature; in `serveSubmit`, after the successful `h.store.Write(...)` call, add `if err := h.notifier.Notify(r.Context()); err != nil { log.Printf(...) }` (log but do not propagate — a missed notification delays the SSE update, it does not fail the score submission)
- [x] T005 Add a `FakeScoreNotifier` type to `app/internal/leaderboard/leaderboardtest/fake_store.go` implementing `ScoreNotifier` with a `NotifyCalls int` counter and optional `NotifyErr error`; update `newTestHandler()` in `app/internal/leaderboard/handler_test.go` to pass a `FakeScoreNotifier` to the updated `NewHandler` signature; update `main_test.go` if it constructs a `Handler` directly
- [x] T006 Write Testcontainers test in `app/internal/leaderboard/store_test.go` covering `RedisScoreStore.Notify()`: start a real Redis container, subscribe to `leaderboard:score-updated` channel in a goroutine, call `Notify()`, assert the subscriber receives the message within 2 s
- [x] T007 Implement `scores-service/internal/scores/store.go` — define `Standing` struct (`Rank int`, `Name string`, `Score int`); define `Store` struct with `client *redis.Client`, `limit int`, `channel string`; implement `NewStore(client, limit, channel)`; implement `ReadBest(ctx context.Context) ([]Standing, error)` (XRANGE `leaderboard:scores` → group by `name` keeping max `score` in a `map[string]int` → convert to slice → sort descending by score → assign 1-based ranks → truncate to `limit`); implement `Subscribe(ctx context.Context, ch chan<- struct{})` as a goroutine that calls `client.Subscribe(ctx, channel)`, reads messages, and sends `struct{}{}` on `ch` for each message received
- [x] T008 Write Testcontainers tests in `scores-service/internal/scores/store_test.go` using a real Redis container: `ReadBest` returns empty slice when stream is empty; `ReadBest` with one entry returns rank 1; `ReadBest` with two entries from same player returns only the higher score; `ReadBest` with three distinct players returns them ranked by score descending; `Subscribe` fires on `PUBLISH leaderboard:score-updated ""`

**Checkpoint**: `app` publishes to Redis on every score write; `scores-service` store reads and subscribes correctly against real Redis

---

## Phase 3: User Story 1 — Leaderboard Displays Live Score Standings (Priority: P1) 🎯 MVP

**Goal**: The leaderboard page shows live standings that update automatically when a score is submitted — no page reload required

**Independent Test**: `docker compose up --build`; submit a score via `curl`; observe the standings column update within 5 s in the browser without a page reload (quickstart.md Scenario 4)

- [x] T009 [US1] Implement `scores-service/internal/scores/handler.go` — `Handler` struct with `store *Store`; `NewHandler(store *Store) *Handler`; `ServeHTTP` dispatches on path + method; `serveList` handles `GET /scores` (calls `store.ReadBest`, marshals `{"standings":[...]}`, returns 200; returns `{"standings":[]}` — not `null` — when empty); `serveStream` handles `GET /scores/stream` (sets SSE headers, sends initial `event: standings\ndata: {...}\n\n` on connect, then enters loop blocking on a `Subscribe` channel, sending a new `standings` event on each notification, exits on client disconnect via `r.Context().Done()`); `setCORSHeaders` sets `Access-Control-Allow-Origin: *`; OPTIONS preflight returns 204; all responses include CORS headers
- [x] T010 [P] [US1] Write unit tests in `scores-service/internal/scores/handler_test.go` with a stub store — `GET /scores` returns 200 JSON with correct standings shape; `GET /scores` returns `{"standings":[]}` when store returns empty slice; `Content-Type: application/json` on REST; `Content-Type: text/event-stream` on SSE; SSE on-connect emits exactly one `event: standings` block with valid JSON data before the test client disconnects
- [x] T011 [US1] Implement `scores-service/main.go` — `Config` struct with fields `ListenAddr` (env `SCORES_LISTEN_ADDR`, default `:8083`), `RedisAddr` (env `REDIS_ADDR`, default `redis:6379`), `ScoresLimit` (env `SCORES_LIMIT`, default `10`, parse with `strconv.Atoi`), `PubSubChannel` (env `REDIS_PUBSUB_CHANNEL`, default `leaderboard:score-updated`); `main()` creates `redis.Client`, creates `Store`, creates `Handler`, registers `/scores` and `/scores/stream` on a `http.NewServeMux()`; starts `http.Server` with `WriteTimeout: 0` (SSE connections are long-lived); logs startup message
- [x] T012 [P] [US1] Create `frontend/leaderboard/scores-component.js` — export `ScoresComponent({ scoresServiceURL })` as a React functional component using `React.createElement` (no JSX, matching `commits-component.js` style); `useState` for standings array; on mount, open `EventSource(scoresServiceURL + '/scores/stream')`; add listener for `standings` events that parses JSON and calls `setStandings(data.standings)`; add `onerror` handler that activates a fallback `setInterval` polling `fetch(scoresServiceURL + '/scores')` every 5000 ms; render a `<ul>` of standings entries with rank, name, score; return `null` on initial load before first event
- [x] T013 [US1] Update leaderboard page template in `app/main.go` — remove the `<ul id="standings">` element, `<p id="status">` element, and the entire polling IIFE (`(function () { var POLL_INTERVAL_MS...})();`) from the template; add `<div id="scores-root"></div>` in the standings column; add `<script src="/leaderboard-assets/scores-component.js"></script>` alongside the existing React bundle script tags; add a new `<script type="module">` block that imports `ScoresComponent` from `/leaderboard-assets/scores-component.js` and mounts it with `ReactDOM.createRoot(document.getElementById('scores-root')).render(React.createElement(ScoresComponent, { scoresServiceURL: '{{.ScoresServiceURL}}' }))`
- [x] T014 [US1] Update `app/main.go` config and App struct — add `scoresServiceURL string` field to the `App` struct; add `ScoresServiceURL: envOr("SCORES_SERVICE_URL", "http://localhost:8083")` to the config block; update the `handleLeaderboardPage` data struct from `struct{ CommitsServiceURL string }` to `struct{ CommitsServiceURL, ScoresServiceURL string }` and pass both fields; update the leaderboard page template header comment to reflect the ScoresServiceURL injection
- [x] T015 [US1] Remove `serveList()` method and the `case http.MethodGet` branch from `ServeHTTP` in `app/internal/leaderboard/handler.go`; remove the `defaultStandingsLimit` and `maxStandingsLimit` constants; remove `standingsResponse` and `standing` structs; update `ServeHTTP` allowed-methods error to `Allow: POST` only
- [x] T016 [US1] Remove `Top(ctx context.Context, limit int) ([]Entry, error)` from the `ScoreStore` interface in `app/internal/leaderboard/store.go`; remove the `Top()` method from `RedisScoreStore`; remove the exported `RankTop()` function (no longer referenced once `Top()` is gone); remove `Top()` and `TopErr` from `app/internal/leaderboard/leaderboardtest/fake_store.go`; remove `fake_store_test.go` test cases that exercise `Top()` behaviour; remove all `TestHandlerList*` test functions from `app/internal/leaderboard/handler_test.go`; remove `Top()` test cases from `app/internal/leaderboard/store_test.go`

**Checkpoint**: `docker compose up --build`; submit a score; leaderboard standings update live in the browser within 5 s — Principle IV satisfied

---

## Phase 4: User Story 2 — Empty State Feedback (Priority: P2)

**Goal**: When no scores exist, the standings column shows a clear "no scores yet" message instead of a blank area

**Independent Test**: Point the compose stack at a fresh Redis (no scores); open leaderboard; confirm "No scores yet — be the first to play!" is visible in the standings column (quickstart.md Scenario 2)

- [x] T017 [US2] Update `frontend/leaderboard/scores-component.js` to render `<p>"No scores yet — be the first to play!"</p>` (or equivalent styled element) when `standings.length === 0` after at least one event has been received; keep the loading/before-first-event state distinct (render nothing or a neutral spinner before first SSE event arrives, not the empty-state message)
- [x] T018 [P] [US2] Add empty-state test cases to `scores-service/internal/scores/handler_test.go` — verify `GET /scores` with an empty-returning store responds with `{"standings":[]}` (body must be exactly the JSON array wrapped in the object, not `null` or `{"standings":null}`); verify `Content-Type: application/json; charset=utf-8`

**Checkpoint**: Opening the leaderboard with no Redis score data shows the empty-state message; submitting the first score replaces it with a standings entry

---

## Phase 5: User Story 3 — Compose-Orchestrated Service Startup (Priority: P3)

**Goal**: `docker compose up` from a fresh clone starts the scores-service alongside all other services with no additional host-side steps

**Independent Test**: Fresh clone, `docker compose up --build`, `curl -s http://localhost:8083/scores` returns 200 JSON within 60 s (quickstart.md Scenario 1)

- [x] T019 [US3] Add `scores-service` service block to `docker-compose.yml` — `build: {context: scores-service, dockerfile: Dockerfile}`; `image: whale-runner-scores:local`; `environment: [REDIS_ADDR=redis:6379, SCORES_LISTEN_ADDR=:8083, SCORES_LIMIT=10, REDIS_PUBSUB_CHANNEL=leaderboard:score-updated]`; `ports: ["8083:8083"]`; `depends_on: [redis]`; add a comment block matching the style of the `commits-service` block explaining the service's role
- [x] T020 [US3] Add `SCORES_SERVICE_URL=${SCORES_SERVICE_URL:-http://localhost:8083}` to the `app` service `environment` block in `docker-compose.yml`; add a comment explaining that this is the browser-visible URL injected into the leaderboard page template (matching the existing `COMMITS_SERVICE_URL` comment)

**Checkpoint**: `docker compose up` starts the scores-service; leaderboard page works end-to-end from a fresh compose stack

---

## Phase 6: Polish & Validation

**Purpose**: Verify correctness after all changes, satisfy Principle IV (visible-in-browser definition of done)

- [x] T021 [P] Run `go test ./...` inside `app/` — all tests must pass; the build must succeed with no references to removed symbols (`Top`, `RankTop`, `serveList`, `defaultStandingsLimit`, `maxStandingsLimit`)
- [x] T022 [P] Run `go test ./...` inside `scores-service/` — all Testcontainers tests pass (real Redis container); confirm no compilation errors
- [x] T023 Execute quickstart.md validation scenarios 1–9 against a running `docker compose up --build` stack — confirm all scenarios pass, including the visible-in-browser score update (Principle IV: feature is not done until observed working in the browser)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — blocks all user story phases
- **US1 (Phase 3)**: Depends on Phase 2 completion — implements the core feature
- **US2 (Phase 4)**: Depends on Phase 3 completion (component exists); adds empty-state behaviour
- **US3 (Phase 5)**: Depends on Phase 3 completion (scores-service binary exists); adds compose wiring
- **Polish (Phase 6)**: Depends on all desired stories complete

### Within Phase 2

```
T003 (ScoreNotifier interface + impl)
  └─► T004 (inject notifier into handler)
  └─► T005 (FakeScoreNotifier + update tests) — parallel with T004
  └─► T006 (Testcontainers test for Notify) — sequential after T003

T007 (scores-service store impl) — parallel with T003–T006
  └─► T008 (Testcontainers tests for store)
```

### Within Phase 3 (US1)

```
T009 (handler impl)
  └─► T010 [P] (handler tests) — parallel with T009

T011 (main.go wiring) — depends on T009

T012 [P] (scores-component.js) — parallel with T009–T011

T013 (template changes) — depends on T012
T014 (config + data struct) — parallel with T013
T015 (remove serveList from handler) — after T013–T014 (replacement ready)
T016 (remove Top/RankTop + clean tests) — after T015
```

### User Story Independence

- **US2 (Phase 4)**: Can proceed after Phase 3 without waiting for Phase 5
- **US3 (Phase 5)**: Can proceed after Phase 3 without waiting for Phase 4

---

## Parallel Opportunities

### Phase 2

```bash
# These can start immediately after T003:
Task T004: "Inject ScoreNotifier into app/internal/leaderboard/handler.go"
Task T005: "Add FakeScoreNotifier to leaderboardtest/fake_store.go"
Task T006: "Testcontainers test for RedisScoreStore.Notify()"
Task T007: "Implement scores-service/internal/scores/store.go"  # independent of T003–T006
```

### Phase 3 (US1)

```bash
# These can start in parallel once Phase 2 is done:
Task T009: "Implement scores-service/internal/scores/handler.go"
Task T010: "Write handler unit tests"  # [P] - can start with T009
Task T012: "Create frontend/leaderboard/scores-component.js"  # [P] - independent
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (`scores-service/` structure)
2. Complete Phase 2: Foundational (app publish + scores-service store with Testcontainers)
3. Complete Phase 3: US1 (endpoints, React component, template swap, old code removal)
4. **STOP and VALIDATE**: Submit a score, watch it appear in the browser — Principle IV
5. Proceed to US2 and US3 once US1 is confirmed working

### Incremental Delivery

1. Phase 1 + Phase 2 → Foundation ready (pub/sub wired, store tested)
2. Phase 3 → Live standings on leaderboard → **MVP** (demo-ready)
3. Phase 4 → Clean empty-state message → polish pass
4. Phase 5 → `docker compose up` wires everything → delivery-ready
5. Phase 6 → Full validation pass → ship

### Total Task Count: 23 tasks

| Phase | Story | Tasks | Parallel [P] |
|-------|-------|-------|--------------|
| Phase 1: Setup | — | T001–T002 | T002 |
| Phase 2: Foundational | — | T003–T008 | T004, T005, T007 |
| Phase 3: Implementation | US1 | T009–T016 | T010, T012 |
| Phase 4: Empty State | US2 | T017–T018 | T018 |
| Phase 5: Compose | US3 | T019–T020 | — |
| Phase 6: Polish | — | T021–T023 | T021, T022 |

---

## Notes

- `[P]` tasks operate on different files with no dependency on a simultaneously-running task
- `[Story]` labels trace each task to the user story it satisfies for PR review and demo verification
- Testcontainers tests (T006, T008) require Docker Desktop running during `go test`
- Principle III (NON-NEGOTIABLE): any test crossing a Redis boundary MUST use Testcontainers — no mock-based Redis testing
- Principle IV (NON-NEGOTIABLE): T023 (browser observation) is required; passing tests alone do not satisfy the definition of done
- The `scores-component.js` uses `React.createElement` (no JSX) to match `commits-component.js` — no build pipeline needed
