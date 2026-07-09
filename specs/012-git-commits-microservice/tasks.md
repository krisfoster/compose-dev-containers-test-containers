---
description: "Task list for git commits microservice feature"
---

# Tasks: Git Commits Microservice

**Input**: Design documents from `specs/012-git-commits-microservice/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/commits-openapi.yaml, contracts/commits-sse-contract.md, quickstart.md

**Tests**: Unit tests included for the commits handler (the handler reads from the filesystem, so standard Go tests apply; no Testcontainers needed at the handler level).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to ([US1], [US2], [US3])

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create new module structure, vendor React assets, and draft the constitution amendment — all parallelizable with each other.

- [X] T001 [P] Create `commits-service/` directory with `go.mod` (`module crossywhale/commits-service`, `go 1.25.0`) and empty `go.sum` at repo root
- [X] T002 [P] Create `commits-service/internal/commits/` directory hierarchy (empty packages, no logic yet) at repo root
- [X] T003 [P] Create `frontend/leaderboard/` directory and download React 18 + ReactDOM 18 production UMD bundles into `frontend/leaderboard/react.production.min.js` and `frontend/leaderboard/react-dom.production.min.js` (source: npm package tarballs, per research.md §7)
- [X] T004 [P] Draft constitution amendment in `.specify/memory/constitution.md`: MINOR version bump adding React 18 to the Technology Stack section (per plan.md Constitution Check — gate required before ship)
- [X] T005 [P] Add React 18 and ReactDOM 18 entries to `ATTRIBUTION.md` (author: Meta, source: react.dev / npm, licence: MIT, no modifications — per plan.md Constitution Check)

**Checkpoint**: Module skeleton exists; React assets vendored; constitution amendment drafted.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: HTTP skeleton for the commits service and the docker-compose entry. Must be complete before any user story work begins.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T006 Add `github.com/go-git/go-git/v5` dependency to `commits-service/go.mod` and run `go mod tidy` inside `commits-service/` to produce a valid `go.sum`
- [X] T007 [P] Create `commits-service/main.go`: environment config struct (fields: `ListenAddr` from `COMMITS_LISTEN_ADDR` defaulting to `:8082`, `GitRepoPath` from `GIT_REPO_PATH` defaulting to `/repo`), `main()` that starts `net/http` on `ListenAddr` with a placeholder mux (no routes yet), and structured startup log line
- [X] T008 [P] Create `commits-service/internal/commits/handler.go`: define `commitEntry` struct (fields: `Hash`, `Author`, `Date`, `Message` per data-model.md), define `Handler` struct holding `gitRepoPath string`, and `NewHandler(gitRepoPath string) *Handler` constructor — no HTTP methods yet
- [X] T009 Add `commits-service` service to `docker-compose.yml`: build context `commits-service/`, image name `whale-runner-commits:local`, environment `GIT_REPO_PATH=/repo`, port `8082:8082`, volume `./.git:/repo/.git:ro`, `depends_on: []` (no Redis dependency), placed after the `app` service definition
- [X] T010 Create `commits-service/Dockerfile`: multi-stage build using `dhi.io/golang:1.25-alpine-dev` as build stage (WORKDIR `/src`, COPY `go.mod go.sum`, `go mod download`, COPY `.`, `go build -o /out/commits-service .`, `mkdir /out/repo`) and `dhi.io/static:20260611-alpine3.24` as final stage (COPY binary and `/out/repo`)

**Checkpoint**: `docker compose up` builds and starts `commits-service` (no routes respond yet, but the service starts clean). All of T006–T010 must be complete before Phase 3.

---

## Phase 3: User Story 1 — Leaderboard Displays Live Commit Feed (Priority: P1) 🎯 MVP

**Goal**: The commits service exposes `GET /commits` (REST) and `GET /commits/stream` (SSE), with CORS headers. The leaderboard page loads a React component that subscribes to the SSE stream and renders the commit list, updating live. The old `/api/commits` handler is removed from `app`.

**Independent Test**: `curl http://localhost:8082/commits` returns JSON with `commits` array; `curl -N http://localhost:8082/commits/stream` streams `event: commits` events; the leaderboard page at `http://localhost:8080/leaderboard` shows the commit list and updates without a page reload.

### Implementation for User Story 1

- [X] T011 [US1] Implement `(*Handler).ServeHTTP` in `commits-service/internal/commits/handler.go`: route `GET /commits` to `serveList`, `GET /commits/stream` to `serveStream`, `OPTIONS` to `serveOptions`; add `setCORSHeaders(w)` helper that sets `Access-Control-Allow-Origin: *`
- [X] T012 [US1] Implement `(*Handler).serveList` in `commits-service/internal/commits/handler.go`: open git repo at `h.gitRepoPath`, walk HEAD log up to 20 entries, build `[]commitEntry` (hash=7-char short SHA, author=name truncated to 64 chars, date formatted as `2006-01-02 15:04` UTC, message=subject line), JSON-encode as `{"commits": [...]}`, set `Content-Type: application/json` and `Cache-Control: no-store`, return 503 text on git errors (per data-model.md derivation rules)
- [X] T013 [US1] Implement `(*Handler).serveStream` in `commits-service/internal/commits/handler.go`: set SSE headers (`Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`, CORS), flush initial `event: commits\ndata: <json>\n\n` immediately, then tick every 30 s using `time.NewTicker`, write next event on each tick, return cleanly on client disconnect (`r.Context().Done()`), detect flush support via `http.Flusher` interface
- [X] T014 [US1] Wire `commits/Handler` into `commits-service/main.go`: instantiate `commits.NewHandler(cfg.GitRepoPath)` and register on the mux at `/commits` and `/commits/stream`; remove the placeholder mux from T007
- [X] T015 [P] [US1] Write unit tests in `commits-service/internal/commits/handler_test.go`: create a temporary bare git repo fixture with 2 commits using `go-git` in-process, test `GET /commits` via `httptest.NewRecorder` returns 200 + valid JSON with 2 entries in correct field shapes; test `GET /commits` against an empty (no-commit) repo returns 200 + `{"commits":[]}` (not 503); test CORS header present on all responses
- [X] T016 [P] [US1] Create `frontend/leaderboard/commits-component.js`: ES module exporting a React component `CommitsComponent` that (a) opens `EventSource` to the commits service URL, (b) sets state on each `commits` event via `JSON.parse(e.data).commits`, (c) falls back to 30 s `setInterval` + `fetch` if `EventSource` is unavailable, (d) renders a `<ul>` of commit entries (hash, author, date, message), (e) renders nothing while loading (first render before first event), (f) keeps last known list on transient SSE error — uses `React.createElement` only (no JSX); accepts `{ commitsServiceURL }` prop
- [X] T017 [US1] Update `leaderboardPageHTML` Go string constant in `app/main.go`: (a) add `<script>` tags to load vendored React and ReactDOM from `/leaderboard-assets/react.production.min.js` and `/leaderboard-assets/react-dom.production.min.js`, (b) replace the existing `<script>` commit-feed section (lines `// Commit feed: poll /api/commits...` through `setInterval(refreshCommits, 30000)`) with a `<div id="commits-root"></div>` target element, (c) add `<script type="module">` that imports `CommitsComponent` from `/leaderboard-assets/commits-component.js` and calls `ReactDOM.createRoot(document.getElementById('commits-root')).render(React.createElement(CommitsComponent, {commitsServiceURL: 'http://localhost:8082'}))`, (d) add CSS for `#commits-root` to match the existing `.commit-col` layout
- [X] T018 [US1] Add `/leaderboard-assets/` route to `app/main.go`: add `http.FileServer(http.Dir(cfg.LeaderboardAssetsDir))` handler at `/leaderboard-assets/` on `ungatedMux` and `gatedMux`; add `LeaderboardAssetsDir` field to `Config` with `envOr("LEADERBOARD_ASSETS_DIR", "/leaderboard-assets")` default
- [X] T019 [US1] Remove `handleCommits`, `gitRepoPath` field from `App` struct, and `GitRepoPath` from `Config`/`loadConfig` in `app/main.go`; remove the `/api/commits` route registrations from both `ungatedMux` and `gatedMux`; remove the `gogit` and `go-git` imports
- [X] T020 [US1] Run `go mod tidy` in `app/` to remove `go-git/go-git/v5` and all its transitive deps from `app/go.mod` and `app/go.sum` after T019
- [X] T021 [US1] Update `app/Dockerfile`: add `COPY frontend/leaderboard /leaderboard-assets` line to the final stage (after the existing `COPY frontend/game /frontend` line)
- [X] T022 [US1] Update `docker-compose.yml` `app` service: add `- LEADERBOARD_ASSETS_DIR=/leaderboard-assets` to its `environment` list; remove the `.git` volume from the `app` service (it no longer needs git access — only `commits-service` does)

**Checkpoint**: `docker compose up` starts both services. `curl http://localhost:8082/commits` returns JSON. The leaderboard at `http://localhost:8080/leaderboard` shows the commit list populated by the React component via SSE. `curl http://localhost:8080/api/commits` returns 404. All handler unit tests pass.

---

## Phase 4: User Story 2 — Empty State Feedback (Priority: P2)

**Goal**: When the commits service returns an empty commits array, the React component displays a user-friendly message instead of a blank column.

**Independent Test**: Point the commits service at an empty git repo (or mock); open the leaderboard in a browser; the commits column shows "No commits yet — make your first commit to see it here!" (or equivalent) instead of a blank or broken UI.

### Implementation for User Story 2

- [X] T023 [US2] Update `CommitsComponent` in `frontend/leaderboard/commits-component.js`: add a conditional render branch — when `commits` state is a non-null empty array (`commits.length === 0`), render a `<p>` with the empty-state message "No commits yet — make your first commit to see it here!" styled consistently with the column's existing appearance; keep the loading-state (null) branch showing nothing so it doesn't flash the empty message before the first event arrives
- [X] T024 [US2] Add a unit test case to `commits-service/internal/commits/handler_test.go` that specifically verifies the HTTP handler for an empty-commit repo returns `{"commits":[]}` (not a 503 or null) — this ensures the service-side contract for the empty state is explicit and regression-protected
- [X] T025 [P] [US2] Update the `#commit-status` CSS rule in `leaderboardPageHTML` (`app/main.go`) if needed to ensure the empty-state `<p>` rendered by the React component inherits correct font size and opacity within the `.commit-col` layout (may be a no-op if existing CSS already covers it)

**Checkpoint**: With an empty commits array delivered by the service, the leaderboard column shows the "no commits" message. With commits present, it shows the list. Both states transition correctly when data changes.

---

## Phase 5: User Story 3 — Compose-Orchestrated Service Startup (Priority: P3)

**Goal**: A fresh clone with only Docker Desktop and git installed reaches the demoable state (commits service running, leaderboard commit feed live) via `docker compose up` with no additional host-side steps.

**Independent Test**: Perform `docker compose up` from a fresh clone on a machine with only Docker Desktop; `docker compose ps` shows `commits-service` as `running`; `curl http://localhost:8082/commits` returns valid JSON.

### Implementation for User Story 3

- [X] T026 [US3] Verify `commits-service/Dockerfile` builds cleanly from a cold Docker cache: run `docker compose build commits-service` with `--no-cache` and confirm the image is produced without errors; fix any COPY paths or build context issues found
- [X] T027 [US3] Add `COMMITS_SERVICE_URL` environment variable injection to the leaderboard HTML rendering: update `leaderboardPageHTML` in `app/main.go` to render the `commitsServiceURL` prop value from an env var (`COMMITS_SERVICE_URL`, defaulting to `http://localhost:8082`) so the URL is configurable without code changes for different demo environments (e.g., an event-specific port or hostname); convert `leaderboardPageHTML` from a `const string` to a `text/template` rendered by `handleLeaderboardPage` (matching how `handlePlayIndex` already uses `html/template`)
- [X] T028 [US3] Add `COMMITS_SERVICE_URL` to `docker-compose.yml` `app` service environment with default `http://localhost:8082`; add a comment noting this must match the published host port of `commits-service`

**Checkpoint**: `docker compose up` from a fresh clone starts all services. `http://localhost:8080/leaderboard` shows the commit feed without any manual setup step. The service URL is configurable via `.env` for demo environments.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validation, cleanup, and ship-gate items.

- [X] T029 [P] Run the full `quickstart.md` validation checklist (all 11 items) against the running compose stack; document any gaps and fix before marking complete
- [X] T030 [P] Run `go test ./...` in `commits-service/` and confirm all unit tests pass; run `go test ./...` in `app/` and confirm existing tests still pass after the handler removal
- [X] T031 [P] Verify the commits service SSE stream works end-to-end in the browser: open `http://localhost:8080/leaderboard`, open DevTools Network tab, confirm an `EventStream` connection to `http://localhost:8082/commits/stream` is established and `commits` events arrive; make a git commit and observe the feed update within the next broadcast cycle
- [X] T032 [P] Update `app/main_test.go`: remove any test cases that reference `handleCommits` or `gitRepoPath` (the handler no longer exists); add a test verifying `GET /api/commits` returns 404 on the `ungatedMux` and `gatedMux`
- [X] T033 Finalize and merge the constitution amendment from T004 (version bump applied to `.specify/memory/constitution.md`; Sync Impact Report header filled in; templates reviewed)
- [X] T034 [P] Update `README.md` (or `crossy.md` runtime guidance) to note the `commits-service` as a new compose service and document the `COMMITS_SERVICE_URL` override for demo environments

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — T001–T005 can start immediately and run fully in parallel with each other
- **Foundational (Phase 2)**: T001–T002 must complete before T006–T010 (module must exist before adding deps); T006–T010 can run in parallel with each other; Phase 2 BLOCKS Phase 3
- **User Story 1 (Phase 3)**: Depends on Phase 2 completion — T011→T012→T013→T014 are sequential (handler builds up); T015 and T016 can run in parallel with T011–T014; T017→T018 depend on T016 (component file must exist before HTML references it); T019→T020 depend on T017 (route removed after new route added); T021–T022 depend on T018–T019
- **User Story 2 (Phase 4)**: Depends on T016 (commits component file) and T012 (empty array return from handler); T023–T025 can run in parallel with each other
- **User Story 3 (Phase 5)**: Depends on T009 (compose entry) and T010 (Dockerfile); T027 depends on T007 (leaderboard HTML)
- **Polish (Phase 6)**: Depends on all story phases being complete

### User Story Dependencies

- **US1 (P1)**: Depends on Phase 2 only — the foundational MVP
- **US2 (P2)**: Depends on T016 (component) and T012 (handler returning empty array) — independently testable once those exist
- **US3 (P3)**: Depends on T009 + T010 (compose + Dockerfile) — can be partially validated independently of US1/US2

### Parallel Opportunities

Within Phase 2: T006, T007, T008, T009, T010 — all different files, no mutual deps (after T001–T002 complete)
Within Phase 3: T015 (tests) + T016 (component) can run while T011–T014 (handler impl) is in progress
Within Phase 3: T021 (Dockerfile) + T022 (compose env) can run while T019–T020 (code removal) is in progress
Within Phase 4: T023, T024, T025 can all run in parallel
Within Phase 6: T029, T030, T031, T032, T034 can all run in parallel

---

## Parallel Example: Phase 3 (US1)

```
Start simultaneously:
  Task T015: Write handler unit tests in commits-service/internal/commits/handler_test.go
  Task T016: Write commits-component.js in frontend/leaderboard/commits-component.js
  Task T011: Implement ServeHTTP routing in commits-service/internal/commits/handler.go

After T011:
  Task T012: Implement serveList in commits-service/internal/commits/handler.go
  (T015 should now have a handler to run against — verify tests fail before T013)

After T012:
  Task T013: Implement serveStream in commits-service/internal/commits/handler.go

After T013 + T016:
  Task T014: Wire handler into commits-service/main.go
  Task T017: Update leaderboardPageHTML in app/main.go

After T016 + T017:
  Task T018: Add /leaderboard-assets/ route to app/main.go

After T018:
  Task T019: Remove old handler from app/main.go
  Task T020: go mod tidy in app/
  Task T021: Update app/Dockerfile
  Task T022: Update docker-compose.yml app environment
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001–T005, all parallel)
2. Complete Phase 2: Foundational (T006–T010) — CRITICAL BLOCKER
3. Complete Phase 3: User Story 1 (T011–T022)
4. **STOP and VALIDATE**: `curl http://localhost:8082/commits`, open leaderboard in browser, verify SSE events in DevTools Network tab, confirm `GET /api/commits` on app returns 404
5. Deploy/demo if ready — the live commit feed is the primary visible value

### Incremental Delivery

1. Phases 1–2 → Foundation ready
2. Phase 3 (US1) → Live commit feed in leaderboard → Demo (MVP!)
3. Phase 4 (US2) → Empty state message → More robust demo
4. Phase 5 (US3) → Verified from-scratch compose startup → Ship-ready
5. Phase 6 → Polish → Clean ship

---

## Notes

- [P] tasks operate on different files and have no dependencies on each other within the same phase
- [Story] labels map each task to its user story for traceability to spec.md
- The constitution amendment (T004/T033) is a ship gate — don't skip it
- `commits-service/` is a standalone Go module; run `go mod tidy` and `go test` from inside that directory
- The `commitsServiceURL` prop in the React component uses `http://localhost:8082` — this works for local dev but must be overridable for demo environments (T027–T028 cover this)
- After T019, the `app` binary no longer needs `go-git` — T020's `go mod tidy` will shrink `app/go.sum` noticeably
