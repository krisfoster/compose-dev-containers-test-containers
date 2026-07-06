---

description: "Task list template for feature implementation"
---

# Tasks: Host Web App with Public Ngrok Access

**Input**: Design documents from `/specs/001-host-webapp-ngrok/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, quickstart.md

**Tests**: No automated tests are generated — this feature introduces no application code (per plan.md Technical Context). Each story's validation is a `quickstart.md` scenario run against the real compose stack in a browser, per constitution Principle IV.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Infra-only, single-project layout (see plan.md Project Structure): `docker-compose.yml` and `.env.example` at repo root, `webserver/nginx.conf` for the static-file-server config, `frontend/game/` consumed read-only and unmodified.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the standalone config files neither story depends on the other for.

- [X] T001 [P] Create `.env.example` at repo root with `WEB_PORT=8080` and `NGROK_AUTHTOKEN=` (blank), each with a one-line comment; note that `NGROK_AUTHTOKEN` is only required for the `public` profile (per plan.md Technical Context, research.md #4).
- [X] T002 [P] Create `webserver/nginx.conf` at repo root: a minimal static-file server block (`listen 80;`, document root `/usr/share/nginx/html`, `location / { try_files $uri $uri/ =404; }`) — no app logic, per research.md #1.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create the single `docker-compose.yml` both stories add services into.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Create `docker-compose.yml` at repo root with a top-level `services:` key and no services defined yet, ready for US1/US2 to populate.

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Presenter hosts the app locally with one command (Priority: P1) 🎯 MVP

**Goal**: `docker compose up` serves the existing `frontend/game/` content over local HTTP, with no other setup step.

**Independent Test**: On a clean machine with only Docker installed, run the startup command and load the web app in a browser at the local address.

### Implementation for User Story 1

- [X] T004 [US1] Add the `webserver` service to `docker-compose.yml`: `image: nginx:alpine`, mount `./frontend/game:/usr/share/nginx/html:ro`, mount `./webserver/nginx.conf:/etc/nginx/conf.d/default.conf:ro`, `ports: ["${WEB_PORT:-8080}:80"]` (depends on T002, T003).
- [X] T005 [US1] Run `specs/001-host-webapp-ngrok/quickstart.md` Scenario 1 in a browser: confirm the app is reachable at `http://localhost:8080` within 2 minutes of a cold `docker compose up -d` (SC-001), and confirm `docker compose down && docker compose up -d` restores it with no config changes (FR-007).
- [X] T006 [US1] Run `specs/001-host-webapp-ngrok/quickstart.md` Scenario 4 in a terminal: occupy `WEB_PORT` with another container first, confirm `docker compose up` fails fast with Compose's "port is already allocated" message rather than starting silently (FR-008), then confirm the fix (stop the conflicting container, or set a different `WEB_PORT` in `.env`) resolves it.

**Checkpoint**: At this point, User Story 1 is fully functional and demoable on its own — this is the MVP.

---

## Phase 4: User Story 2 - Attendees reach the app over the public internet (Priority: P1)

**Goal**: An opt-in `ngrok` service exposes the `webserver` service on a public HTTPS URL that the presenter can find and share.

**Independent Test**: With `docker compose --profile public up -d` running, open the URL shown at `http://localhost:4040` from a device on a different network (e.g. a phone on cellular data) and confirm the same app loads.

### Implementation for User Story 2

- [X] T007 [US2] Add the `ngrok` service to `docker-compose.yml`: `image: ngrok/ngrok:3`, `profiles: ["public"]`, `environment: ["NGROK_AUTHTOKEN=${NGROK_AUTHTOKEN}"]`, `command: ["http", "webserver:80"]`, `ports: ["4040:4040"]`. Do **not** add a `depends_on` between `webserver` and `ngrok` in either direction (research.md #5 — keeps local access independent of tunnel health) (depends on T004).
- [X] T008 [US2] Run `specs/001-host-webapp-ngrok/quickstart.md` Scenario 2 end-to-end: `docker compose --profile public up -d`, read the forwarding URL from `http://localhost:4040`, open it from a separate network, confirm the app loads within 5 seconds (SC-002), and confirm it is still reachable after a multi-hour window with no presenter intervention (SC-003).

**Checkpoint**: At this point, User Stories 1 AND 2 both work — local hosting plus optional public sharing.

---

## Phase 5: User Story 3 - Local demo keeps working if public access fails (Priority: P2)

**Goal**: A public-tunnel outage (missing token, provider down, no internet) never takes down local access, and recovers on its own when possible.

**Independent Test**: With the app running and public access enabled, stop the `ngrok` container and confirm the web app is still reachable locally; restart it and confirm public access returns.

### Implementation for User Story 3

- [X] T009 [US3] Add `restart: unless-stopped` to the `ngrok` service in `docker-compose.yml` so transient provider/network failures self-heal without presenter action (research.md #5) (depends on T007).
- [X] T010 [US3] Run `specs/001-host-webapp-ngrok/quickstart.md` Scenario 3 end-to-end: stop the `ngrok` container (or start without `NGROK_AUTHTOKEN` set), confirm `http://localhost:8080` still loads and `http://localhost:4040` is unreachable (the "public sharing unavailable" signal, FR-006), then restart `ngrok` and confirm the public URL returns.

**Checkpoint**: All three user stories are independently functional; the feature is complete.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect the feature as a whole, after all stories are done.

- [X] T011 [P] Add a "Hosting the demo" section to `README.md` summarizing `docker compose up -d` (local) and `docker compose --profile public up -d` (public), linking to `specs/001-host-webapp-ngrok/quickstart.md` for full validation steps.
- [X] T012 Run `docker compose config` against the finished `docker-compose.yml` to confirm both services and the `public` profile parse cleanly, as a final sanity check before calling the feature done.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — T001 and T002 can start immediately, in parallel.
- **Foundational (Phase 2)**: T003 has no file dependency on T001/T002 but is conventionally sequenced after Setup — BLOCKS all user stories.
- **User Stories (Phase 3+)**: All depend on Foundational (Phase 2) completion.
  - US1 has no dependency on US2 or US3.
  - US2 depends on the `webserver` service existing (T004 from US1), since it tunnels to it.
  - US3 depends on the `ngrok` service existing (T007 from US2), since it modifies that service's restart policy.
- **Polish (Final Phase)**: Depends on all three user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2). No dependency on other stories. This alone is a demoable MVP (local-only hosting).
- **User Story 2 (P1)**: Can start after US1's `webserver` service exists (T004) — it tunnels to that service rather than duplicating it.
- **User Story 3 (P2)**: Can start after US2's `ngrok` service exists (T007) — it hardens that service's failure behavior.

### Within Each User Story

- Service definition before validation (config before the quickstart run that exercises it).
- Story complete before moving to the next priority.

### Parallel Opportunities

- T001 and T002 (Setup) can run in parallel — different files, no dependencies.
- T011 (README) can run in parallel with T012 (compose config check) once all stories are done — different files.
- Because US2 and US3 each build directly on the previous story's service definition, their implementation tasks (T007, T009) are sequential rather than parallel with earlier stories' tasks.

---

## Parallel Example: Setup

```bash
# Launch both Setup tasks together:
Task: "Create .env.example at repo root with WEB_PORT and NGROK_AUTHTOKEN"
Task: "Create webserver/nginx.conf with a minimal static-file server block"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Run quickstart.md Scenarios 1 and 4 in a browser/terminal
5. This is already demoable to people physically present — deploy/demo if that's sufficient for the next event

### Incremental Delivery

1. Complete Setup + Foundational → empty compose file ready
2. Add User Story 1 → validate locally → demo-ready for in-room attendees (MVP!)
3. Add User Story 2 → validate publicly → demo-ready for remote/phone attendees
4. Add User Story 3 → validate resilience → demo-safe against tunnel outages
5. Each story adds value without breaking the previous one

---

## Notes

- No [P] markers appear inside the user-story phases: each story's tasks touch the same `docker-compose.yml` service block sequentially (config, then validation against the running stack), so parallelizing them would race on the same file/running containers.
- Commit after each task or logical group, per user preference (this project only commits on explicit request — see repo conventions).
- Stop at any checkpoint to validate a story independently before moving to the next.
- Avoid: vague tasks, same-file conflicts, cross-story dependencies that break independence beyond the explicit "builds on the previous service" relationships called out above.
