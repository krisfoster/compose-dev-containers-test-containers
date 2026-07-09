---
description: "Task list for Fix Host QR Rotate Route"
---

# Tasks: Fix Host QR Rotate Route

**Input**: Design documents from `specs/019-fix-host-rotate-route/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/host-rotate.md ✓, quickstart.md ✓

**Tests**: Included — spec FR-006 explicitly requires automated tests for success, store error, and method-not-allowed.

**Organization**: Foundational phase delivers the handler (serves both US1 and US2). US1 phase adds the required unit tests. US2 phase validates the auto-rotate timer in the compose stack.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1 = manual rotation, US2 = auto-rotation)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the baseline is green before any changes.

- [x] T001 Run `go test ./...` in `app/` and confirm all existing tests pass (baseline)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Register the route and implement the handler — required by both US1 (manual button) and US2 (auto-rotate timer). Neither user story can be tested until this phase is complete.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T002 Add `mux.HandleFunc("/host/rotate", a.handleHostRotate)` to `ungatedMux()` in `app/main.go` (after the existing `/leaderboard` registration, line ~221)
- [x] T003 Implement `func (a *App) handleHostRotate(w http.ResponseWriter, r *http.Request)` in `app/main.go`: reject non-POST with 405 + `Allow: POST` header; call `a.store.Activate(r.Context(), a.qrWindowTTL)`; return 204 on success, 500 on store error

**Checkpoint**: `go build ./...` in `app/` succeeds — both user stories can now be addressed.

---

## Phase 3: User Story 1 — Manual QR Rotation (Priority: P1) 🎯 MVP

**Goal**: Clicking "Refresh QR" on the leaderboard page produces a new, distinct QR code within 2 seconds.

**Independent Test**: `go test ./... -run TestHandleHostRotate -v` in `app/` — all three test functions pass.

### Tests for User Story 1 (required by FR-006)

> **These test functions can be written and run in parallel with each other once T002/T003 are done.**

- [x] T004 [P] [US1] Write `TestHandleHostRotateActivatesNewWindow` in `app/main_test.go`: POST → 204, and `store.Current()` returns a non-empty ID distinct from the pre-rotation state (use `newTestApp` helper + `gatetest.FakeWindowStore`)
- [x] T005 [P] [US1] Write `TestHandleHostRotateStoreError` in `app/main_test.go`: store failure → 500 (use `appWithErroringStore` helper)
- [x] T006 [P] [US1] Write `TestHandleHostRotateMethodNotAllowed` in `app/main_test.go`: GET → 405 with `Allow: POST` response header (use `newTestApp` helper)

### Validate User Story 1

- [x] T007 [US1] Run `go test ./... -run TestHandleHostRotate -v` in `app/` and confirm all three new tests pass

**Checkpoint**: US1 is fully tested. `POST /host/rotate` works correctly at the unit level.

---

## Phase 4: User Story 2 — Automatic QR Rotation (Priority: P2)

**Goal**: The QR code updates automatically every 60 seconds without manual intervention.

**Independent Test**: Start compose stack, open `/leaderboard` in a browser, wait 60 seconds, observe QR code updates without clicking anything (two consecutive cycles).

**Note**: US2 has no additional implementation tasks — the handler registered in Phase 2 serves both the manual button (US1) and the 60-second timer (US2). The leaderboard JavaScript's timer already calls `POST /host/rotate`; it was silently 404-ing before this fix.

- [ ] T008 [US2] Start the compose stack (`docker compose up --build -d`), open `http://localhost/leaderboard`, wait 60 seconds, and confirm the QR code updates automatically (observe two consecutive auto-rotate cycles per quickstart.md Scenario 3)

**Checkpoint**: US2 confirmed working in the live compose stack.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Browser validation required by Constitution Principle IV (visible-in-the-browser definition of done).

- [ ] T009 [P] Open `http://localhost/leaderboard` in a browser, click "Refresh QR," and confirm the QR code image updates (quickstart.md Scenario 2) — take a screenshot or screen recording as sign-off evidence
- [ ] T010 [P] Confirm `curl -X GET http://localhost/host/rotate` returns 405 (quickstart.md Scenario 4)
- [x] T011 Run the full `app/` test suite (`go test ./...`) one final time to confirm no regressions

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — run immediately
- **Foundational (Phase 2)**: Depends on Phase 1 green baseline — BLOCKS all user story work
- **US1 (Phase 3)**: Depends on Phase 2 complete (handler must exist before tests compile)
- **US2 (Phase 4)**: Depends on Phase 2 complete (handler must exist); independent of Phase 3
- **Polish (Phase 5)**: Depends on Phase 3 + Phase 4 complete

### User Story Dependencies

- **US1 (P1)**: Can start immediately after Foundational — no dependency on US2
- **US2 (P2)**: Can start immediately after Foundational — no dependency on US1

### Within User Story 1

- T004, T005, T006 are all parallel (different test functions, same file — no write conflicts if added sequentially, but logically independent)
- T007 (run tests) depends on T004, T005, T006 all being written

### Parallel Opportunities

- T004, T005, T006 (three test functions) can be drafted simultaneously
- T009 and T010 (Scenario 2 and 4 validations) can be run simultaneously
- Phase 3 and Phase 4 can start in parallel once Phase 2 is done

---

## Parallel Example: Phase 3 test functions

```
Once T002 + T003 (handler) are done, these three tests can be written together:

Task T004: "Write TestHandleHostRotateActivatesNewWindow in app/main_test.go"
Task T005: "Write TestHandleHostRotateStoreError in app/main_test.go"
Task T006: "Write TestHandleHostRotateMethodNotAllowed in app/main_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Verify baseline green
2. Complete Phase 2: Register route + implement handler (`app/main.go`)
3. Complete Phase 3: Write + run the three required unit tests
4. **STOP and VALIDATE**: All three tests pass — manual rotation is working
5. Demo: open leaderboard, click Refresh QR, observe updated QR code

### Incremental Delivery

1. Phases 1 + 2 → Handler live
2. Phase 3 → Unit tests green (US1 done)
3. Phase 4 → Auto-rotation confirmed in compose (US2 done)
4. Phase 5 → Browser sign-off screenshot captured

---

## Notes

- [P] tasks = logically independent; can be executed in parallel
- [Story] label maps each task to its user story from spec.md
- The entire change touches two files: `app/main.go` (2 additions) and `app/main_test.go` (3 additions)
- Total new lines of production code: ~15. Total new test lines: ~40.
- Existing helpers `newTestApp`, `gatetest.FakeWindowStore`, and `appWithErroringStore` cover all test scenarios — no new test infrastructure needed
- Constitution Principle IV requires a browser observation before marking this feature done
