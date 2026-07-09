# Tasks: Nginx Front-Door Routing Layer

**Input**: Design documents from `specs/014-nginx-front-door/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, contracts/routing-table.md ✓, quickstart.md ✓

**Tests**: No Go code changes in this feature; browser/log-inspection validation per Principle IV

**Organization**: Tasks grouped by user story. All changes are infrastructure only (new nginx service + compose/ngrok config updates).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to

---

## Phase 1: Setup

**Purpose**: Create the nginx service directory structure

- [x] T001 Create `nginx/` directory at repo root (`mkdir nginx/`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create the nginx service files that all user stories depend on.

**⚠️ CRITICAL**: Both T002 and T003 must be complete before any docker compose work begins.

- [x] T002 [P] Create `nginx/nginx.conf` with full routing rules per `specs/014-nginx-front-door/contracts/routing-table.md` — include server block on port 80 with: exact-match proxies (`= /play` → `app:8081` with Cookie/Set-Cookie headers; `= /`; `= /leaderboard`; `= /qr.png` with `add_header Cache-Control no-store always`; `= /repo-qr.png` — all → `app:8080`); prefix proxies (`/host` → `app:8080` with Cookie/Set-Cookie; `/api/` → `app:8080`; `/scores` → `scores-service:8083` with SSE directives; `/commits` → `commits-service:8082` with SSE directives); catch-all `location / { try_files $uri =404; }` for static game assets; all proxy locations set `proxy_set_header Host $host`; SSE locations add `proxy_http_version 1.1`, `proxy_set_header Connection ''`, `proxy_buffering off`, `proxy_cache off`, `proxy_read_timeout 3600s`

- [x] T003 [P] Create `nginx/Dockerfile` — `FROM dhi.io/nginx:1-alpine3.24`; build context is repo root; `COPY frontend/game /usr/share/nginx/html`; `COPY container-obstacles.gif /usr/share/nginx/html/container-obstacles.gif`; `COPY frontend/leaderboard /usr/share/nginx/html/leaderboard-assets`; `COPY nginx/nginx.conf /etc/nginx/nginx.conf`

**Checkpoint**: `nginx/Dockerfile` and `nginx/nginx.conf` both exist → proceed to user story phases

---

## Phase 3: User Story 1 — Presenter Starts Demo at a Single Entry Point (Priority: P1) 🎯 MVP

**Goal**: `docker compose up --build` starts the full stack; `http://localhost` is the single browser entry point; landing page, leaderboard, and API routes all work through port 80.

**Independent Test**: `docker compose up --build`, open `http://localhost` in a browser, confirm landing page loads; open `http://localhost/leaderboard`, confirm page renders and SSE panels connect.

### Implementation for User Story 1

- [x] T004 [US1] Add `nginx` service block to `docker-compose.yml` — insert after the `scores-service` block; content: `nginx:`, `  build: {context: ., dockerfile: nginx/Dockerfile}`, `  image: whale-runner-nginx:local`, `  ports: ["${NGINX_PORT:-80}:80"]`, `  depends_on: {app: {condition: service_started}, scores-service: {condition: service_started}, commits-service: {condition: service_started}}`

- [x] T005 [US1] Update `SCORES_SERVICE_URL` and `COMMITS_SERVICE_URL` defaults in `docker-compose.yml` `app` service environment — change `SCORES_SERVICE_URL=${SCORES_SERVICE_URL:-http://localhost:8083}` to `SCORES_SERVICE_URL=${SCORES_SERVICE_URL:-http://localhost}` and `COMMITS_SERVICE_URL=${COMMITS_SERVICE_URL:-http://localhost:8082}` to `COMMITS_SERVICE_URL=${COMMITS_SERVICE_URL:-http://localhost}`

- [x] T006 [US1] Run `docker compose build nginx` and confirm build succeeds with no errors; then run `docker compose up --build` and confirm: (a) nginx container starts without error, (b) `http://localhost` returns the Crossy Whale landing page, (c) `http://localhost/leaderboard` returns the leaderboard page, (d) `http://localhost/api/ping` returns JSON, (e) `http://localhost/leaderboard-assets/react.production.min.js` returns 200

**Checkpoint**: User Story 1 independently verified — landing page, leaderboard, and APIs reachable through `http://localhost`

---

## Phase 4: User Story 2 — Player Joins via QR Code Through the Routing Layer (Priority: P2)

**Goal**: ngrok tunnels to nginx:80; the gate check on `/play` works correctly through the nginx proxy; grant cookies are forwarded intact.

**Independent Test**: `docker compose up --build`, visit `http://localhost/host` to activate QR window, then visit `http://localhost/play` in the same browser; confirm the game loads with `window.__LEADERBOARD_TOKEN__` set to the actual token (not the raw Go template placeholder `{{.LeaderboardToken}}`).

### Implementation for User Story 2

- [x] T007 [US2] Update `ngrok.yml` — change `addr: http://app:8081` to `addr: http://nginx:80`

- [x] T008 [US2] Verify gate path with compose running: (a) open `http://localhost/host` in a browser — confirm QR image loads and QR window activates, (b) open `http://localhost/play` in same browser session — confirm game HTML loads, (c) in browser devtools console type `window.__LEADERBOARD_TOKEN__` — confirm value is the actual secret string, not `{{.LeaderboardToken}}`

**Checkpoint**: Gate path verified through nginx — cookie forwarding and token injection work correctly

---

## Phase 5: User Story 3 — Static Assets Served Without Backend Involvement (Priority: P3)

**Goal**: Requests for game JS, CSS, GLB, and audio files are served by nginx directly; the Go app process has no access log entries for those requests.

**Independent Test**: Request a static game asset through nginx; inspect `docker compose logs app` to confirm no entry for that path; inspect `docker compose logs nginx` to confirm nginx served it.

### Implementation for User Story 3

- [x] T009 [US3] Verify static asset isolation with compose running: (a) request `curl -s -o /dev/null -w "%{http_code}" http://localhost/three.module.js` (or another game JS file that exists in `frontend/game/`) — should return 200, (b) run `docker compose logs app` and confirm no access log entry for that path appears, (c) run `docker compose logs nginx` and confirm an access log entry for that path exists — this proves nginx served it directly without a Go hop

**Checkpoint**: All three user stories independently verified

---

## Phase 6: Polish & Validation (Principle IV Definition of Done)

**Purpose**: Complete quickstart.md validation to satisfy Principle IV (Visible-in-the-Browser DoD)

- [x] T010 Run quickstart.md Scenarios 1–5: start stack with `docker compose up --build`, execute each scenario (basic start, game play/gate, leaderboard live updates, static asset isolation, score submission); confirm all expected outcomes are observed in the browser and logs

- [x] T011 Run quickstart.md Scenario 7 (SSE 5-minute hold test): open `http://localhost/leaderboard`, keep browser tab open for 5 minutes, confirm both SSE connections (`/scores/stream`, `/commits/stream`) remain open and active in Network tab with no disconnections or reload errors

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Setup (T001) — T002 and T003 can run in parallel; BLOCKS all user story phases
- **US1 (Phase 3)**: Depends on Foundational complete — T004 → T005 (same file, sequential) → T006 (build/validation)
- **US2 (Phase 4)**: Depends on US1 complete (nginx must be running)
- **US3 (Phase 5)**: Depends on US1 complete (nginx must be running)
- **Polish (Phase 6)**: Depends on US1 + US2 + US3 complete

### Within Each User Story

- US1: T004 → T005 (same file; sequential) → T006 (browser validation)
- US2: T007 (file change) → T008 (browser validation)
- US3: T009 (browser/log validation only — no code change needed)

### Parallel Opportunities

- T002 and T003 (Phase 2) — different files, no dependency
- US2 (Phase 4) and US3 (Phase 5) — different validations, no dependency on each other; both depend on US1 completing

---

## Parallel Example: Foundational Phase

```bash
# Both can be written simultaneously (different files):
Task T002: Write nginx/nginx.conf
Task T003: Write nginx/Dockerfile
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001)
2. Complete Phase 2: Foundational (T002 + T003 in parallel)
3. Complete Phase 3: User Story 1 (T004 → T005 → T006)
4. **STOP and VALIDATE**: `http://localhost` serves the full demo through a single port
5. Proceed to US2 and US3 as incremental improvements

### Incremental Delivery

1. Setup + Foundational → nginx image ready to build
2. US1 → single-port access works locally
3. US2 → ngrok tunnels through nginx; QR gate verified
4. US3 → confirm static serving separation (observational only)
5. Polish → full quickstart.md validation; Principle IV DoD

---

## Notes

- No Go code changes in this feature — all changes are `nginx/Dockerfile`, `nginx/nginx.conf`, `docker-compose.yml`, `ngrok.yml`
- The constitution amendment (v1.4.0 Routing Layer carve-out) was applied during `/speckit-plan` and is already in `.specify/memory/constitution.md`
- T006, T008, T009, T010, T011 are browser/log-inspection tasks — they satisfy Principle IV and cannot be replaced by automated tests
- Port 8080 (app ungated) remains published in compose for developer debugging; it is no longer the primary access path
- `frontend/game/index.html` is still served by nginx as a static file at `/` (through the catch-all); the Go-rendered version with token injection is only at `/play` via the gate proxy
