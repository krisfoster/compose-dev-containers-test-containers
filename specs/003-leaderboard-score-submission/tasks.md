---

description: "Task list for Player Name Entry, Game Over Score Display, and Leaderboard Score Submission"
---

# Tasks: Player Name Entry, Game Over Score Display, and Leaderboard Score Submission

**Input**: Design documents from `/specs/003-leaderboard-score-submission/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/leaderboard-openapi.yaml, quickstart.md

**Tests**: Included for backend logic, per the constitution's non-negotiable Principle III
(Testcontainers Over Mocks) — the new `leaderboard` package's Redis-touching code MUST be tested
against a real Redis; validation and credential logic above that boundary are made testable via a
`ScoreStore` interface, so unit/handler tests run against a fast in-memory fake instead. The
browser-only flow (name prompt, Game Over display, Replay) has no automated test tasks — it is
validated manually per `quickstart.md`, per constitution Principle IV, matching how
`002-qr-gated-access` treated its camera-scan flow.

**Organization**: Tasks are grouped by user story (US1-US5, matching spec.md's P1/P1/P1/P2/P2
priorities) so each can be implemented and demoed independently once Foundational is done.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1-US5)
- File paths are relative to the repository root unless otherwise noted

## Path Conventions

Extends the existing single Go module at `app/` (adds `app/internal/leaderboard/`) and the existing
static frontend at `frontend/game/`, per `plan.md`'s Project Structure. No new module, service, or
top-level directory.

---

## Phase 1: Setup

**Purpose**: Add the one new piece of shared configuration every later phase depends on

- [X] T001 [P] Add `LEADERBOARD_API_SECRET` to `.env.example`, with a short explanatory comment
      mirroring the existing `GRANT_COOKIE_SECRET` entry
- [X] T002 [P] Add `LEADERBOARD_API_SECRET` (default `dev-only-change-me`, sourced from `.env`) to
      the `app` service's `environment` block in `docker-compose.yml`

**Checkpoint**: `docker compose config` shows the new variable wired through; no new dependencies,
services, or Go module changes needed (reuses `go-redis` and `testcontainers-go`, already vendored
by `002-qr-gated-access`).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The shared leaderboard write path every user story depends on — the Redis-backed
store (behind a testable interface), request validation, the credential check, the HTTP handler,
and the credential-injection change to how the game page is served

**⚠️ CRITICAL**: No user story task can be verified end-to-end until this phase is complete

- [X] T003 Define a `ScoreStore` interface (write one name+score entry) and its Redis
      Stream-backed implementation (`XADD leaderboard:scores`) in
      `app/internal/leaderboard/store.go`, per `data-model.md`'s Leaderboard Entry — this
      interface is what makes T006's handler and later stories' tests testable without a live
      container
- [X] T004 [P] Provide an in-memory fake `ScoreStore` implementation in
      `app/internal/leaderboard/leaderboardtest/fake_store.go` — a regular (non-test) file in its
      own subpackage, not `store_fake_test.go`, mirroring `app/internal/gate/gatetest`'s pattern
      from `002-qr-gated-access` (Go does not allow importing another package's `_test.go` symbols,
      and this fake must be usable both by `leaderboard`'s own tests and by `app/main_test.go`)
- [X] T005 [P] Write a Testcontainers-go test for the Redis Stream-backed `ScoreStore` (write via
      `XADD`, read back via `XRANGE`, confirm fields match) in
      `app/internal/leaderboard/store_test.go` (constitution Principle III — no mocked Redis
      client for this test; depends on T003)
- [X] T006 Implement request validation (name: trim whitespace, reject empty, reject >32 chars;
      score: reject missing, negative, or non-integer values) and the credential check
      (constant-time comparison against the configured `LEADERBOARD_API_SECRET`), plus the
      `POST /api/leaderboard/scores` handler wiring validation → credential check →
      `ScoreStore.Write`, returning `201`/`400`/`401` per `contracts/leaderboard-openapi.yaml`, in
      `app/internal/leaderboard/handler.go` (depends on T003)
- [X] T007 Wire the new handler onto both `ungatedMux()` and `gatedMux()` in `app/main.go` at
      `/api/leaderboard/scores`, deliberately not wrapped in `gate.Middleware` (research.md §2 —
      the credential check is this feature's own, independent protection) (depends on T006)
- [X] T008 Switch `handlePlayIndex` in `app/main.go` from `http.ServeFile` to an `html/template`
      render of `frontend/game/index.html` that injects the configured `LEADERBOARD_API_SECRET`
      into an inline `window.__LEADERBOARD_TOKEN__ = "...";` script tag (research.md §4); add the
      corresponding template placeholder to the `<head>` of `frontend/game/index.html`
- [X] T009 [P] Write a handler test in `app/main_test.go` confirming a request to the game's index
      page on either listener returns a response body containing the configured token value
      (depends on T008)

**Checkpoint**: `docker compose up` serves a game page that carries the write credential, and
`POST /api/leaderboard/scores` is live on both listeners, fully validating and authorizing
requests and writing to Redis — but nothing in the frontend calls it yet, and there is no name
prompt or Game Over screen. Foundation ready for user story work.

---

## Phase 3: User Story 1 - Player enters their name before playing (Priority: P1) 🎯 MVP

**Goal**: The game will not start until the player has entered a non-empty display name (FR-001,
FR-002, FR-003)

**Independent Test**: Load the game, confirm a name prompt appears before any gameplay is
possible, confirm submitting empty/whitespace input is rejected, then confirm a valid name starts
the game immediately

### Implementation for User Story 1

- [X] T010 [US1] Add name-prompt overlay markup to `frontend/game/index.html` (lands after T008 —
      same file, different section)
- [X] T011 [US1] Implement the name-prompt flow in `frontend/game/script.js`: block game start
      until a trimmed, non-empty name (≤32 chars) is entered; retain it in a module-level variable
      for reuse by later stories
- [X] T012 [P] [US1] Style the name-prompt overlay in `frontend/game/style.css`
- [X] T013 [US1] Run `quickstart.md` Scenario 1 against `docker compose up` (manual browser
      validation, constitution Principle IV) — this sandbox has no browser available (headless
      Chromium download is blocked by network policy), so implementation-time verification was
      limited to served markup/token checks and a hand-trace of `script.js`'s `nameEntered` guard.
      **Confirmed by the user in a real browser**: the game asks for a name before play starts.

**Checkpoint**: User Story 1 is fully functional and independently demoable — this is the MVP.

---

## Phase 4: User Story 2 - Game Over screen shows the player their score (Priority: P1)

**Goal**: On death, gameplay stops and a "Game Over" screen shows only the player's own score for
that attempt (FR-004, FR-005)

**Independent Test**: Play until death and confirm a "Game Over" screen appears showing the
numeric score for that attempt, with no other players' data or ranking visible

### Implementation for User Story 2

- [X] T014 [US2] Add Game Over overlay markup (score display only) to `frontend/game/index.html`
- [X] T015 [US2] Wire the Game Over display in `frontend/game/script.js`: on death (the existing
      `gameOver` flag becoming true), stop gameplay and show only this attempt's score
- [X] T016 [P] [US2] Style the Game Over overlay in `frontend/game/style.css`
- [X] T017 [US2] Run `quickstart.md` Scenario 2 (manual browser validation) — same no-browser
      limitation as T013 at implementation time; verified then by hand-tracing the
      `!gameOver`-guarded collision block in `script.js`. **Confirmed by the user in a real
      browser**: the Game Over screen appears on death and shows the score.

**Checkpoint**: User Stories 1 and 2 both work independently — the full local play loop (name →
play → death → own score shown) is visible end to end.

---

## Phase 5: User Story 3 - Score and name are recorded to the leaderboard store (Priority: P1)

**Goal**: The moment a player dies, their name and score are submitted to the Redis-backed
leaderboard store with no extra action from the player, and repeat attempts never overwrite a
prior entry (FR-006, FR-007, FR-008)

**Independent Test**: Play a round to completion, then confirm — independently of the game UI —
that a new entry with the matching name and score exists in the leaderboard store; repeat under a
different name and under the same name, and confirm neither overwrites another entry

### Tests for User Story 3

- [X] T018 [P] [US3] Write handler tests in `app/internal/leaderboard/handler_test.go` against the
      fake `ScoreStore` (T004): a valid request (correct credential, valid name/score) is accepted
      (`201`) and produces exactly one write; two valid submissions under the identical name
      produce two separate entries, neither overwriting the other (FR-007, FR-008)

### Implementation for User Story 3

- [X] T019 [US3] Implement the score-submission call in `frontend/game/script.js`: on death,
      `POST` `{name, score}` to `/api/leaderboard/scores` with the `X-Leaderboard-Token` header
      read from `window.__LEADERBOARD_TOKEN__`; best-effort — a failed or slow request never
      blocks or delays the Game Over display (FR-006; edge case: store temporarily unavailable)
- [X] T020 [US3] Run `quickstart.md` Scenario 3, including the `redis-cli XREVRANGE`/`XLEN` checks
      (manual browser validation plus direct store inspection) — validated the store-inspection
      half directly against a live `docker compose up` stack: two `curl` `POST`s to
      `/api/leaderboard/scores` with the real `X-Leaderboard-Token` (one under a repeated name)
      both returned `201`, and `redis-cli XLEN leaderboard:scores` / `XRANGE leaderboard:scores - +`
      showed both entries present with correct `name`/`score` fields, neither overwriting the
      other (FR-007/FR-008 confirmed against real Redis). The browser half (playing to death and
      seeing the same score reflected) has the same no-browser limitation as T013/T017; the
      `script.js` call site (`submitScore(playerName, currentLane)`) was traced by hand to confirm
      it fires with the correct arguments at the same point the Game Over score is set.

**Checkpoint**: User Stories 1-3 together deliver the feature's core value — capture, display, and
durable recording.

---

## Phase 6: User Story 4 - Replay restarts the game without losing the entered name (Priority: P2)

**Goal**: A Replay control on the Game Over screen immediately starts a new attempt, reusing the
already-entered name with no re-prompt (FR-009, FR-010)

**Independent Test**: From the Game Over screen, activate Replay and confirm a fresh attempt
starts immediately using the same name, with no name prompt shown again

### Implementation for User Story 4

- [X] T021 [US4] Add a Replay control to the Game Over overlay markup in `frontend/game/index.html`
- [X] T022 [US4] Implement Replay handling in `frontend/game/script.js`: resets game state and
      starts a new attempt immediately on activation, reusing the already-entered name with no
      prompt shown
- [X] T023 [P] [US4] Style the Replay control in `frontend/game/style.css`
- [X] T024 [US4] Run `quickstart.md` Scenario 4 (manual browser validation) — same no-browser
      limitation as T013 at implementation time; verified then by confirming the pre-existing
      `#retry` handler never touches `playerName`/`nameEntered`, so Replay keeps the name by
      construction. **Confirmed by the user in a real browser** as part of the overall playthrough.

**Checkpoint**: All P1 stories plus Replay are functional and demoable together.

---

## Phase 7: User Story 5 - Arbitrary clients cannot write scores to the leaderboard (Priority: P2)

**Goal**: A score-write request without a valid credential is rejected and nothing is recorded
(FR-012)

**Independent Test**: Send a score-submission request directly to the API without the game's
credential and confirm it is rejected with no entry recorded; confirm a normal play-through still
succeeds

### Tests for User Story 5

- [X] T025 [P] [US5] Write handler tests in `app/internal/leaderboard/handler_test.go` confirming
      a request with a missing credential header and a request with an invalid credential are both
      rejected `401` with no `ScoreStore` write, and that both responses are identical (FR-012 —
      no distinguishing leak, consistent with `002-qr-gated-access`'s gate error responses)
- [X] T026 [P] [US5] Write handler tests confirming validation failures (empty/whitespace-only
      name, name >32 chars, missing score, negative score, non-integer score) are rejected `400`
      with no `ScoreStore` write (FR-002, FR-003)

### Implementation for User Story 5

- [X] T027 [US5] Run `quickstart.md` Scenario 5 and the Validation-failure spot check against
      `docker compose up` (curl-based manual validation) — run against a live stack: a request
      with no `X-Leaderboard-Token` and one with a garbage token both returned `401` with
      `XLEN leaderboard:scores` unchanged; a request with the real token from `.env` returned
      `201`; an empty-name request and a negative-score request both returned `400` with no new
      stream entry. All outcomes matched the quickstart's expectations exactly.

**Checkpoint**: All five user stories are independently functional and demoable together.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Bring the rest of the repo's documentation in line with the new feature and confirm
the coverage bar established by `002-qr-gated-access` still holds

- [X] T028 [P] Update `README.md`'s "Running the app" section to link
      `specs/003-leaderboard-score-submission/quickstart.md` alongside the existing 001/002 links
- [X] T029 Run `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out` in
      `app/`; confirm the new `leaderboard` package meets this project's existing ≥80% statement
      coverage bar (established by `002-qr-gated-access`'s plan.md), adding tests for any gap
- [X] T030 Run the full `quickstart.md` validation (all five scenarios plus the validation-failure
      spot check) end-to-end against a fresh `docker compose up`, confirming no regressions across
      US1-US5 — all API/store-level checks (Scenarios 3, 5, and the validation-failure spot check)
      were run against a fresh `docker compose up` at implementation time and matched expectations
      exactly (see T020, T027). The full `go test ./...` suite (all packages, including
      Testcontainers-go Redis tests) passes, and total statement coverage across `app/` is 90.1%
      (`go tool cover -func`), above the project's 80% bar. Browser-driven Scenarios 1, 2, and 4
      (name prompt, Game Over display, Replay) couldn't be run from this sandbox (no browser
      available; headless Chromium's download is blocked by network policy) and were hand-traced
      instead at implementation time. **The user has since run the app locally and confirmed all
      three**: the game asks for a name, shows the score at Game Over, and the flow works
      end-to-end — closing out this feature's constitution Principle IV requirement.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories (the store, validation,
  credential check, handler, and index-page credential injection all live here)
- **User Stories (Phase 3-7)**: All depend on Foundational completion
  - US1, US2 are both P1 and independent of each other — either can go first
  - US3 (P1) depends on US2's death handling existing to hang its submission call off of, and on
    Foundational's endpoint
  - US4 (P2) depends on US2's Game Over screen existing (it adds a control to it) and US1's stored
    name
  - US5 (P2) has no new implementation of its own — see the Note on Sequencing below — and can be
    written any time after Foundational, though it is most meaningfully run after US3 exists
- **Polish (Phase 8)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational — no dependency on other stories
- **User Story 2 (P1)**: Can start after Foundational — no dependency on US1's implementation,
  though a full manual runthrough naturally plays through US1 first
- **User Story 3 (P1)**: Depends on US2 (needs the death/Game Over trigger point to hang the
  submission call off of) and on Foundational's endpoint
- **User Story 4 (P2)**: Depends on US1 (the name to retain) and US2 (the screen it adds a control
  to)
- **User Story 5 (P2)**: Depends only on Foundational (its tests exercise the handler directly);
  no dependency on US1-US4's frontend work

### Within Each User Story

- Tests (where included) before the implementation tasks they cover
- Foundational pieces (`ScoreStore`, validation, credential check, handler) before anything that
  routes through them
- Story complete before moving to the next priority, if working sequentially

### Parallel Opportunities

- T001 and T002 (Setup) can run in parallel
- T004 and T005 (Foundational) can run in parallel with each other once T003 lands; T009 can run
  once T008 lands
- Within each user story, tasks marked [P] (styling, or tests against an already-landed fake) can
  proceed alongside that story's other tasks
- US1, US2, and US5 can be staffed to different people once Foundational is merged; US3 and US4
  each have a real dependency on an earlier story's frontend work (see above), so are not fully
  parallel with them

---

## Parallel Example: Foundational Phase

```bash
# After T003 lands, these can proceed together:
Task: "Provide an in-memory fake ScoreStore in app/internal/leaderboard/leaderboardtest/fake_store.go"
Task: "Write a Testcontainers-go test for the Redis Stream-backed ScoreStore in app/internal/leaderboard/store_test.go"
```

## Parallel Example: User Story 5

```bash
Task: "Handler tests: missing/invalid credential both rejected 401, identical response, no write"
Task: "Handler tests: validation failures (name/score bounds) rejected 400, no write"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: run `quickstart.md` Scenario 1
5. At this point the name prompt works, but nothing is recorded anywhere yet and there is no Game
   Over screen — acceptable for validating the prompt in isolation, not yet a demoable loop

### Incremental Delivery

1. Setup + Foundational → the write API and credential mechanism exist, nothing in the UI calls
   them yet
2. Add US1 → name prompt works (MVP for that slice)
3. Add US2 → the play loop now shows the player their own result (demoable local loop)
4. Add US3 → results are durably recorded — the feature's core value is now delivered
5. Add US4 → repeat play is frictionless
6. Add US5 → the write endpoint is verified safe to leave reachable
7. Polish → docs catch up, coverage bar re-verified

### Note on Sequencing for User Story 5

Because US5's core protection (the credential and validation checks) is already implemented by
Foundational's handler (T006) — a request without a valid credential, or with invalid data, is
rejected from the moment the endpoint exists — US5's own tasks only add the dedicated tests and
manual validation that prove that protection holds, mirroring how `002-qr-gated-access` treated its
own already-fail-closed gate decision in its User Story 2. There is no separate "turn on
protection" step to sequence.

### Note on File Overlap in `frontend/game/index.html`

T008 (Foundational), T010 (US1), T014 (US2), and T021 (US4) all touch
`frontend/game/index.html`, each adding a distinct, non-overlapping section (the credential script
tag; the name-prompt overlay; the Game Over overlay; the Replay control within it). These are a
coordination point to sequence rather than truly parallelize if one person owns the file, but they
do not conflict in intent.

---

## Notes

- [P] tasks touch different files, or exercise independent behaviors against a shared fake/test
  file without blocking each other's authorship
- [Story] labels map tasks to spec.md's user stories for traceability
- Redis-touching tests MUST use Testcontainers-go, never a mocked client (constitution Principle
  III, non-negotiable); tests against the `ScoreStore` interface from the handler's perspective use
  the in-memory fake instead, which is not the same thing as mocking Redis itself
- Commit after each task or logical group
- The full name-prompt → play → death → Game Over → submission → Replay flow cannot be automated
  end-to-end — it is the Definition of Done per constitution Principle IV and is called out
  explicitly as its own tasks (T013, T017, T020, T024, T027, T030) rather than folded silently into
  implementation tasks
