---
description: "Task list for Docker Hardened Images (DHI) Migration"
---

# Tasks: Docker Hardened Images (DHI) Migration

**Input**: Design documents from `/specs/005-dhi-image-migration/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/image-inventory.md, quickstart.md

**Tests**: No new test authoring is requested. The existing Testcontainers boundary tests
(constitution Principle III) are *edited* (image string) and *run* as validation within User Story 2.

**Organization**: Tasks grouped by user story (US1→US2→US3 by priority), each independently testable.

> **Implementation status (2026-07-07)**: Migration implemented and validated end-to-end on the
> hardened stack after `docker login dhi.io` (automated via `bin/dhi-login`). Done `[X]`: T001–T013,
> T015. Build runs on DHI golang-dev → static (app nonroot uid 65532, HTTP 200); Redis on
> `dhi.io/redis:8-alpine`; `go test ./...` green including the Testcontainers boundary tests;
> leaderboard write/read + `/host` QR window verified via the live HTTP API.
>
> **Migration finding (FR-006):** DHI redis ships `protected-mode` ON in its bundled
> `/etc/redis/redis.conf`, which refused the app's cross-container connection (and the tests' mapped-port
> connection). Fixed by disabling protected-mode — redis has no published port, so it stays
> internal-only, matching the prior `redis:7-alpine` behaviour. Applied in `docker-compose.yml`
> (`command:` override) and both test helpers (`testcontainers.WithCmd(...)`, since DHI's entrypoint is
> `tini --`, not a redis-server-prepending shim). See research.md Decision 4.
>
> **Still open**: T014 (CVE scan) — `docker scout` isn't installed in this sandbox; the DHI tags
> reported 0 critical/high/medium/low at plan time, but run scout/trivy on a host that has it to
> capture the before/after delta. T016 (full quickstart) — Scenarios 1–3 and 5 pass; Scenario 4 is
> T014; Scenario 6's *live* ngrok tunnel did not establish (authtoken/network, unrelated to DHI — the
> unchanged, exempt `ngrok/ngrok:3` container did start).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 / US2 / US3 (setup, foundational, and polish tasks carry no story label)

## Path Conventions

Containerized web service (Go backend + static frontend) orchestrated by `docker compose`. Edits
are confined to `app/Dockerfile`, `docker-compose.yml`, two Go test files, and docs — no new
source directories (see plan.md → Project Structure).

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish access to the DHI registry — prerequisite for pulling ANY hardened image.

- [X] T001 Authenticate to the DHI registry: run `docker login dhi.io` using a free Docker account (create one at hub.docker.com/signup if needed). Pulling DHI Community images is free; this is the only new prerequisite (research.md Decision 1).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: De-risk all downstream edits by confirming the pinned DHI tags exist and pull.

**⚠️ CRITICAL**: Blocks User Story 1 and User Story 2 (both consume these images).

- [X] T002 Verify the three pinned DHI tags pull under the logged-in account: `docker pull dhi.io/golang:1.25-alpine-dev`, `docker pull dhi.io/static:20260611-alpine3.24`, `docker pull dhi.io/redis:8-alpine`. If the date-stamped `static` tag no longer exists, note the current one from the catalog and use it consistently in T004, research.md, project.md, and the inventory.

**Checkpoint**: Registry access confirmed and tags resolve — story implementation can begin.

---

## Phase 3: User Story 1 - Application image runs on a hardened base (Priority: P1) 🎯 MVP

**Goal**: Build and run the app from DHI (Go **dev** build stage + `static` runtime), with no
behavioural change to the game.

**Independent Test**: `docker compose build app && docker compose up -d`, then the game is playable
in a browser at <http://localhost:8080/play> and the app container runs as nonroot (uid 65532).

### Implementation for User Story 1

- [X] T003 [US1] In `app/Dockerfile`, change the build stage base from `FROM golang:1.25-alpine AS build` to `FROM dhi.io/golang:1.25-alpine-dev AS build` (dev variant: root + `apk` + shell + Go toolchain). Leave the `go mod download` / `go build -o /out/app` steps unchanged (research.md Decision 2).
- [X] T004 [US1] In `app/Dockerfile`, change the runtime stage base from `FROM alpine:3.20` to `FROM dhi.io/static:20260611-alpine3.24`. Keep `COPY --from=build /out/app /app` and `ENTRYPOINT ["/app"]`; add no shell/user directives — the image is nonroot (uid 65532) and shell-less by design (research.md Decision 3). (Same file as T003 → sequential.)
- [X] T005 [US1] Build and run the app image, then validate in a browser: `docker compose build app && docker compose up -d`; open <http://localhost:8080/> and <http://localhost:8080/play> (game loads and plays); confirm nonroot with `docker inspect --format '{{.Config.User}}' "$(docker compose ps -q app)"` (expect `nonroot`/65532). Matches quickstart.md Scenario 1 (FR-002, FR-003, FR-006, SC-003).

**Checkpoint**: App runs on a fully hardened build+runtime image, game visibly playable — MVP done.

---

## Phase 4: User Story 2 - Redis runs on a hardened image everywhere (Priority: P2)

**Goal**: Move Redis to `dhi.io/redis:8-alpine` in compose AND both Testcontainers tests
(documented 7→8 bump; no hardened 7.x exists), keeping runtime/test parity.

**Independent Test**: Stack up on the hardened Redis image → gameplay/leaderboard/QR work in a
browser; `go test ./...` passes with the boundary tests launching the hardened Redis image.

### Implementation for User Story 2

- [X] T006 [US2] In `docker-compose.yml`, change the `redis` service `image: redis:7-alpine` to `image: dhi.io/redis:8-alpine` (research.md Decision 4). Leave the service's other settings unchanged.
- [X] T007 [P] [US2] In `app/internal/leaderboard/store_test.go`, change `tcredis.Run(ctx, "redis:7-alpine")` to `tcredis.Run(ctx, "dhi.io/redis:8-alpine")`.
- [X] T008 [P] [US2] In `app/internal/gate/window_test.go`, change `tcredis.Run(ctx, "redis:7-alpine")` to `tcredis.Run(ctx, "dhi.io/redis:8-alpine")`.
- [X] T009 [US2] Run the boundary tests: `cd app && go test ./...` (host must be logged in to `dhi.io` so Testcontainers can pull). All tests pass; none skipped/mocked. If the redis module's readiness wait misbehaves against the DHI image (tini + `/etc/redis/redis.conf`), add `wait.ForLog("Ready to accept connections")` to the `newTestRedisStore` helper in each test file (research.md Decision 4). Satisfies FR-005, SC-004.
- [X] T010 [US2] Bring up the stack (`docker compose up -d`) and validate in a browser (quickstart.md Scenario 2): enter a name and play to Game Over → score recorded; <http://localhost:8080/leaderboard> shows it and auto-refreshes; <http://localhost:8080/host> renders a QR code (Redis-backed gate). Confirms FR-004, SC-003.

**Checkpoint**: Redis hardened in runtime and tests with parity; state-backed features unchanged.

---

## Phase 5: User Story 3 - Every image accounted for, exemptions documented (Priority: P3)

**Goal**: A complete, reconciled image inventory; docs (`project.md`, `README.md`) updated for the
hardened sources, the new `docker login dhi.io` prerequisite, and the ngrok demo-only exemption.

**Independent Test**: A repo-wide grep for image references reconciles 1:1 with the inventory (3
migrated + 1 exempt); a new contributor following only the README reaches the playable demo.

### Implementation for User Story 3

- [X] T011 [P] [US3] Update the image table in `project.md` (the `redis`/`ngrok` rows near lines 153–156 and any golang/alpine references) to the DHI sources — `dhi.io/golang:1.25-alpine-dev` (build), `dhi.io/static:20260611-alpine3.24` (runtime), `dhi.io/redis:8-alpine`, and mark `ngrok/ngrok:3` as exempt (demo-only). Add a link to `specs/005-dhi-image-migration/contracts/image-inventory.md`. Satisfies FR-011.
- [X] T012 [P] [US3] Update `README.md`: in "Running the app" → Prerequisites (line ~92), add "a free Docker account and `docker login dhi.io`" as the one new prerequisite for pulling hardened images; in "Make it publicly accessible (optional)", reinforce that ngrok is a demo-only exception not migrated to DHI. Satisfies FR-012, SC-007.
- [X] T013 [P] [US3] Reconcile `specs/005-dhi-image-migration/contracts/image-inventory.md` against a repo-wide image-reference search (`grep -rnE 'image:|^FROM |tcredis\.Run' docker-compose.yml app/Dockerfile app/**/*_test.go`); confirm every reference maps to exactly one inventory row, 3 are `migrated` to `dhi.io/...` and exactly 1 (ngrok) is `exempt` with rationale. Satisfies FR-008, FR-009, SC-001, SC-002, SC-006. (Run after T003–T008 land so the tree matches the inventory.)

**Checkpoint**: All images accounted for; docs and inventory agree with the working tree.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Prove the security value and confirm no regression across the whole feature.

- [ ] T014 [P] Compare known-CVE counts: scan the migrated images and the public originals (e.g. `docker scout cves "$(docker compose images -q app)"` and `docker scout cves dhi.io/redis:8-alpine` vs `docker scout cves redis:7-alpine`); confirm the migrated images report fewer CVEs. Satisfies SC-005, quickstart.md Scenario 4.
- [X] T015 Confirm the optional public profile still runs on the unchanged ngrok image: with `NGROK_AUTHTOKEN` set in `.env`, `docker compose --profile public up -d`, then <http://localhost:8080/host> renders the QR (exemption sanity). quickstart.md Scenario 6.
- [ ] T016 Run the full `quickstart.md` validation end-to-end (Scenarios 1–6) and confirm no behavioural regression in gameplay, gating, or leaderboard against the hardened stack. Constitution Principle IV; SC-003.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (T001)**: No dependencies — start immediately.
- **Foundational (T002)**: Depends on T001 — BLOCKS US1 and US2.
- **US1 (T003–T005)**: Depends on Foundational. MVP.
- **US2 (T006–T010)**: Depends on Foundational. Independent of US1 (its browser check assumes an app container is running — from US1 or an existing build).
- **US3 (T011–T013)**: Docs/inventory. T013 should run after US1+US2 file edits (T003–T008) so the reconciliation matches the working tree; T011/T012 can be done any time after the plan is fixed.
- **Polish (T014–T016)**: After the stories being validated are complete.

### Within Each User Story

- **US1**: T003 → T004 (same file, sequential) → T005 (build/run, needs both edits).
- **US2**: T006 (compose) ∥ {T007, T008} (test files, parallel) → T009 (`go test`, needs T007+T008) ; T010 (browser, needs T006).
- **US3**: T011, T012, T013 all edit different files → parallel (with T013's timing note above).

### Parallel Opportunities

- **US2**: T007 and T008 (different test files) run in parallel.
- **US3**: T011 (`project.md`), T012 (`README.md`), T013 (inventory) run in parallel.
- **Polish**: T014 (scan) is parallelizable with doc work.

---

## Parallel Example: User Story 2

```bash
# The two Testcontainers image-string edits touch different files — do them together:
Task: T007 Edit app/internal/leaderboard/store_test.go → dhi.io/redis:8-alpine
Task: T008 Edit app/internal/gate/window_test.go → dhi.io/redis:8-alpine
# then, once both land:
Task: T009 cd app && go test ./...
```

## Parallel Example: User Story 3

```bash
Task: T011 Update project.md image table + inventory link
Task: T012 Update README.md prerequisites + ngrok exception note
Task: T013 Reconcile contracts/image-inventory.md vs repo grep
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. T001 (login) → T002 (verify pulls) → T003/T004 (Dockerfile) → T005 (build + play in browser).
2. **STOP and VALIDATE**: game playable on the hardened app image, running nonroot.
3. This alone delivers the biggest attack-surface reduction (the app's own image).

### Incremental Delivery

1. Setup + Foundational → registry access proven.
2. US1 → app on hardened base → validate in browser (MVP).
3. US2 → Redis hardened in compose + tests → `go test` + browser validate.
4. US3 → inventory + docs reconciled.
5. Polish → CVE comparison, public-profile sanity, full quickstart.

---

## Notes

- `.env.example` needs no change — DHI auth is via `docker login dhi.io`, not an env var (research.md Decision 6).
- The `static` runtime tag is date-stamped (not semver); keep T004, research.md, project.md, and the inventory referencing the same tag (T002 confirms the current one).
- ngrok stays on `ngrok/ngrok:3` throughout — it is the single documented exemption, not a task to migrate.
- Commit after each logical group; stop at any checkpoint to validate a story independently.
- Every task above follows the checklist format (checkbox, ID, [P]/[Story] where applicable, explicit file path or command).
