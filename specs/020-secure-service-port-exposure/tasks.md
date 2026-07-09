# Tasks: Secure Microservice Port Exposure

**Input**: Design documents from `specs/020-secure-service-port-exposure/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, contracts/port-topology.md ✓, quickstart.md ✓

**Tests**: Not requested — this is a Compose-configuration-only change with no automated test framework applicable. Validation is performed manually via `quickstart.md`.

**Organization**: Tasks are grouped by user story. Both stories target the same file (`docker-compose.yml`) so they are sequential by necessity, but each story is independently verifiable once its block is changed.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (no file conflicts)
- **[Story]**: Which user story this task belongs to

---

## Phase 1: Setup (Review & Baseline)

**Purpose**: Confirm current state before making changes

- [x] T001 Read `docker-compose.yml` lines 111–150 to confirm current `ports:` entries for commits-service (8082), scores-service (8083), and qr-service (8084) match the arch-issues.md description

---

## Phase 2: Foundational

No foundational prerequisites — this change requires no shared infrastructure. Skip directly to user story phases.

---

## Phase 3: User Story 1 — Score Integrity Protected from Direct Port Access (Priority: P1) 🎯 MVP

**Goal**: Remove the host-reachable port on `scores-service` so POST /scores cannot bypass the nginx auth_request gate.

**Independent Test**: After completing this phase, `curl --max-time 3 http://localhost:8083/scores` returns "Connection refused" while `curl -X POST http://localhost/api/leaderboard/scores` (without a valid cookie) returns HTTP 401.

### Implementation for User Story 1

- [x] T002 [US1] In `docker-compose.yml`: replace `ports:\n  - "8083:8083"` with `expose:\n  - "8083"` in the `scores-service` block (lines ~132–133)
- [x] T003 [US1] In `docker-compose.yml`: update the `scores-service` service comment (lines ~120–121) to remove the phrase "port 8083 is also published directly for developer access" and replace with a note that the port is internal-only and host access requires a compose override file

**Checkpoint**: `docker compose up -d` then `curl --max-time 3 http://localhost:8083/scores` → Connection refused. `curl http://localhost/scores` → HTTP 200 (via nginx). User Story 1 is independently verified.

---

## Phase 4: User Story 2 — Read-Only Service Ports Protected from Host Access (Priority: P2)

**Goal**: Remove host-reachable ports on `commits-service` and `qr-service` for consistency with the nginx-as-single-ingress principle.

**Independent Test**: After completing this phase, `curl --max-time 3 http://localhost:8082/commits` and `curl --max-time 3 http://localhost:8084/qr.png` both return "Connection refused", while the leaderboard commit feed and QR code continue to work via the nginx front door.

### Implementation for User Story 2

- [x] T004 [US2] In `docker-compose.yml`: replace `ports:\n  - "8082:8082"` with `expose:\n  - "8082"` in the `commits-service` block (lines ~111–112)
- [x] T005 [US2] In `docker-compose.yml`: update the `commits-service` service comment (lines ~101–102) to remove the phrase "port 8082 is also published directly for developer access" and replace with a note that the port is internal-only
- [x] T006 [US2] In `docker-compose.yml`: replace `ports:\n  - "8084:8084"` with `expose:\n  - "8084"` in the `qr-service` block (lines ~149–150)
- [x] T007 [US2] In `docker-compose.yml`: update the `qr-service` service comment (lines ~141) to remove the phrase "Port 8084 is published for direct developer access" and replace with a note that the port is internal-only

**Checkpoint**: `docker compose up -d` then confirm all three microservice ports are refused from host while leaderboard, commit feed, and QR code all work via nginx. User Stories 1 and 2 are both independently verified.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Debug access documentation and final validation

- [x] T008 In `docker-compose.yml`: add a brief comment block near the top of the file (or in a relevant location) explaining that microservices use `expose:` not `ports:`, and how to restore direct port access via a `docker-compose.override.yml` file if needed for debugging (see `contracts/port-topology.md` for the override template)
- [x] T009 Run all validation scenarios from `specs/020-secure-service-port-exposure/quickstart.md` against the running compose stack and confirm all Pass Criteria are met
- [x] T010 [P] Update `specs/020-secure-service-port-exposure/contracts/port-topology.md` if any port numbers or service names differ from what was actually implemented

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **User Story 1 (Phase 3)**: Depends on Phase 1 (baseline confirmed)
- **User Story 2 (Phase 4)**: Depends on Phase 1; can start in parallel with Phase 3 since tasks touch different service blocks — however, since all edits target the same file (`docker-compose.yml`), sequential execution avoids merge conflicts
- **Polish (Phase 5)**: Depends on both Phase 3 and Phase 4 being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start immediately after Phase 1
- **User Story 2 (P2)**: Independent of US1 (different service blocks), but sequential in same file

### Within Each User Story

- Change the `ports:` directive before updating the comment (T002 before T003, T004 before T005, T006 before T007)
- Validate by restarting affected service after each change: `docker compose up -d --no-deps <service-name>`

---

## Parallel Opportunities

All tasks target `docker-compose.yml`, so true parallelism is limited. However:

```bash
# After Phase 1 completes, US1 and US2 edits could be batched into one editing session:
# Edit scores-service block (T002, T003)
# Edit commits-service block (T004, T005)
# Edit qr-service block (T006, T007)
# Then restart all three services and validate together
```

T010 (contract update) can run in parallel with T009 (validation run) since they touch different files.

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Read and confirm current state (T001)
2. Complete Phase 3: Change scores-service `ports:` → `expose:` and update comment (T002, T003)
3. **STOP and VALIDATE**: `curl http://localhost:8083/scores` → Connection refused; `curl http://localhost/api/leaderboard/scores` → 401
4. Ship US1 if the write-path exposure is the urgent concern

### Incremental Delivery

1. T001 → Baseline confirmed
2. T002 + T003 → scores-service secured (US1 done, independently testable)
3. T004 + T005 + T006 + T007 → commits + qr secured (US2 done)
4. T008 + T009 + T010 → polish and full validation

### Single-Session Strategy (Recommended for this change)

Given the small scope (one file, six line-level edits), the most efficient path is a single focused editing session:

1. T001 (read file, confirm state)
2. T002–T007 (all six edits in one pass through docker-compose.yml)
3. `docker compose up -d --build` (restart affected services)
4. T008 (add overview comment)
5. T009 (run quickstart validation)
6. T010 (update contract doc if needed)

---

## Notes

- All edits are in a single file: `docker-compose.yml`
- No Go code, nginx config, or any other file requires changes
- `expose:` syntax in Compose accepts either a string list (`- "8083"`) or a scalar — use the string list form to match the existing `redis` service pattern
- The app service retains its `ports:` entry (`${WEB_PORT:-8080}:8080`) — this is intentional per the port topology contract
- After the change, internal nginx `proxy_pass` directives that reference `scores-service:8083`, `commits-service:8082`, and `qr-service:8084` continue to work unchanged (Docker Compose DNS is unaffected by `expose:` vs `ports:`)
