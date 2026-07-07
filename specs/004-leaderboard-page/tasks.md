---

description: "Task list for Leaderboard Display Page"
---

# Tasks: Leaderboard Display Page

**Input**: Design documents from `/specs/004-leaderboard-page/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/leaderboard-openapi.yaml, quickstart.md

**Tests**: Included for backend logic, per the constitution's non-negotiable Principle III
(Testcontainers Over Mocks) — the new Redis-touching read method MUST be tested against a real
Redis; ranking/limit/tie-break logic above that boundary is tested via the existing in-memory fake.
The page's auto-refresh and stale-data-on-failure behavior (US2, US3) are browser-only and have no
automated test tasks — they are validated manually per `quickstart.md`, per constitution
Principle IV, matching how prior features treated their own browser-only flows.

**Organization**: Tasks are grouped by user story (US1-US3, matching spec.md's P1/P1/P2
priorities) so each can be implemented and demoed independently once Foundational is done.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1-US3)
- File paths are relative to the repository root unless otherwise noted

## Path Conventions

Extends the existing single Go module at `app/` (modifies `app/internal/leaderboard/` and
`app/main.go`), per `plan.md`'s Project Structure. No new module, service, frontend project, or
top-level directory.

---

## Phase 1: Setup

No new configuration, dependencies, or environment variables are needed for this feature — it
reuses the existing `app` Go module, the existing Redis instance, and the existing `leaderboard`
package, with no new secret (Clarifications: reads are unauthenticated, per FR-013). Proceed
directly to Foundational.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The shared leaderboard read path every user story depends on — the extended
`ScoreStore` read method, its fake, and the `GET` handler branch

**⚠️ CRITICAL**: No user story task can be verified end-to-end until this phase is complete

- [X] T001 [P] Extend the `ScoreStore` interface in `app/internal/leaderboard/store.go` with a
      `Top(ctx context.Context, limit int) ([]Entry, error)` method; implement it on
      `RedisScoreStore` via a full `XRange` read of `leaderboard:scores`, decoding each entry's
      `name`/`score`, sorting by score descending with the Stream entry ID descending as a
      tiebreaker (most recent first), and returning at most `limit` entries (research.md §1,
      data-model.md) — implemented via a shared `RankTop` helper (reverse to newest-first, then a
      stable sort by score descending) so `RedisScoreStore` and the fake rank identically
- [X] T002 [P] Extend `FakeScoreStore` in
      `app/internal/leaderboard/leaderboardtest/fake_store.go` with a matching `Top` method over
      its in-memory `Entries` slice, using the same sort/tiebreak rules as T001, so handler tests
      don't need a real Redis (depends on T001 for the interface shape) — also added a `TopErr`
      field (mirroring the existing `WriteErr`) to exercise the GET handler's failure path
- [X] T003 Write Testcontainers-go tests for the new `Top` method in
      `app/internal/leaderboard/store_test.go`: an empty stream returns an empty slice; multiple
      entries come back sorted by score descending; a tied score is broken by insertion order
      (most recently written ranks first); `limit` truncates correctly (constitution Principle
      III — no mocked Redis client for this test; depends on T001) — 6 new tests, all passing
      against a real Redis via Testcontainers-go
- [X] T004 Extend `ServeHTTP` in `app/internal/leaderboard/handler.go` with a `GET` branch: no
      credential check (FR-013); parse an optional `limit` query parameter, defaulting to 10 and
      clamping to `[1, 50]` per `contracts/leaderboard-openapi.yaml`; call `ScoreStore.Top`; write
      a `200` JSON response shaped `{"standings":[{"rank":1,"name":...,"score":...}, ...]}` with
      1-based `rank` assigned from response order. Leave the existing `POST` branch and the
      "method not allowed" fallback for any other verb unchanged (depends on T001) — the existing
      `ServeHTTP` was refactored into `serveSubmit` (POST) and `serveList` (GET), dispatched by
      method
- [X] T005 [P] Write handler tests in `app/internal/leaderboard/handler_test.go` for the new `GET`
      branch against the fake store (T002): an empty store returns `{"standings":[]}`; multiple
      entries return correctly ranked and ordered; an out-of-range `limit` is clamped rather than
      rejected; no credential header is required for a `200` (depends on T002, T004) — also updated
      the pre-existing `TestHandlerRejectsNonPostMethod` (renamed
      `TestHandlerRejectsUnsupportedMethod`, now exercising `PUT`) since `GET` is no longer
      rejected, and added a 500-on-store-error test

**Checkpoint**: `curl http://localhost:8080/api/leaderboard/scores` returns ranked JSON standings
with no credential; nothing renders it in a browser yet. Foundation ready for user story work.

---

## Phase 3: User Story 1 - Viewer sees current leaderboard standings (Priority: P1) 🎯 MVP

**Goal**: A `/leaderboard` page exists, fetches standings once on load, and renders them ranked
highest-to-lowest, with a loading state before the first successful fetch and a clear empty state
when there are no entries (FR-001, FR-002, FR-003, FR-004, FR-008, FR-009, FR-010, FR-011)

**Independent Test**: Seed a few scores via `curl`, open `/leaderboard`, confirm a ranked list of
name/score rows appears; on a fresh stack with no entries, confirm a clear empty state instead of
an error or blank page

### Implementation for User Story 1

- [X] T006 [US1] Add a `GET /leaderboard` route to both `ungatedMux()` and `gatedMux()` in
      `app/main.go`, serving a new inline HTML/CSS/JS response (a new string constant, e.g.
      `leaderboardPageHTML`), mirroring how `hostPageHTML` is already served by `handleHost`
      (FR-001, FR-011)
- [X] T007 [US1] Implement the page's initial render inside that inline script: on load,
      `fetch('/api/leaderboard/scores')` and render each returned standing's rank, name, and
      score as a row; show a loading state before the first response resolves, and a distinct
      empty-state message when `standings` is `[]` (FR-002, FR-003, FR-008, FR-009) (depends on
      T004, T006)
- [X] T008 [P] [US1] Style the standings list, loading state, and empty state within the same
      inline constant's `<style>` block, including layout that degrades gracefully for unusually
      long player names (e.g. truncation/ellipsis CSS) (FR-010) (depends on T006)
- [X] T009 [P] [US1] Write a handler test in `app/main_test.go` confirming `GET /leaderboard`
      returns `200` with an HTML content type on both listeners, with no credential/header
      required (FR-011) (depends on T006) — added
      `TestHandleLeaderboardPageOnBothListeners` and `TestHandleLeaderboardPageWiresUpFetchAndPolling`
- [X] T010 [US1] Run `quickstart.md` Scenario 1 against `docker compose up` (manual browser
      validation, constitution Principle IV) — this sandbox has no browser available (same
      limitation prior features hit), so validation was done via `docker compose up` plus `curl`:
      seeded two scores via `POST /api/leaderboard/scores`, confirmed `GET /leaderboard` serves
      `200 text/html`, and confirmed `GET /api/leaderboard/scores` returns `{"standings":[]}` on a
      fresh stack before seeding and the correctly ranked list after. The rendering/DOM behavior
      itself (the page's JS) was verified by code review and by fetching the served markup
      directly — **awaiting user confirmation in a real browser**.

**Checkpoint**: Opening `/leaderboard` shows current ranked standings, or a clear empty state —
this is the MVP.

---

## Phase 4: User Story 2 - Standings update automatically without a manual reload (Priority: P1)

**Goal**: An already-open `/leaderboard` page keeps refreshing its standings on a recurring
interval, indefinitely, with no manual reload (FR-006)

**Independent Test**: Leave the page open, submit a new score via `curl`, confirm it appears on
the already-open page within about 10 seconds with no reload

### Implementation for User Story 2

- [X] T011 [US2] Wrap User Story 1's fetch-and-render logic in a recurring interval (e.g.
      `setInterval`, on the order of a few seconds) within the leaderboard page's inline script,
      so standings keep refreshing for as long as the page stays open, with no accumulating
      timers across cycles (FR-006, SC-001, SC-002) (depends on T007) — implemented as a single
      `setInterval(refresh, 4000)` call made once at script load (not re-armed per cycle), so
      there is exactly one timer for the page's lifetime
- [X] T012 [US2] Run `quickstart.md` Scenario 2 (manual browser validation): confirm a newly
      submitted score appears on an already-open page within 10 seconds, and that refreshing
      continues indefinitely with the page left idle — same no-browser limitation as T010; the
      4-second poll interval leaves comfortable margin under the 10-second bound (SC-001), and the
      wiring test (T009) confirms `setInterval` is present in the served script.
      **Awaiting user confirmation in a real browser.**

**Checkpoint**: User Stories 1 and 2 together deliver a genuinely live-updating wall display.

---

## Phase 5: User Story 3 - Display stays usable when the leaderboard data is briefly unavailable (Priority: P2)

**Goal**: A failed refresh leaves the last successfully retrieved standings on screen instead of
clearing the display or showing an error (FR-007)

**Independent Test**: While the page is open, make the leaderboard data temporarily unreachable
(e.g. stop Redis), confirm the last-known standings remain visible, then restore access and
confirm refreshing resumes automatically

### Implementation for User Story 3

- [X] T013 [US3] Wrap each polling fetch in the leaderboard page's inline script (T011) in error
      handling that leaves the currently rendered standings untouched on failure — no clearing, no
      error UI — simply retrying on the next interval tick (FR-007) (depends on T011) — implemented
      via a `fetch().then().catch()` chain where the `catch` never touches `listEl.innerHTML`
- [X] T014 [US3] Run `quickstart.md` Scenario 3 (manual browser validation): `docker compose stop
      redis` while the page is open, confirm standings persist on screen; `docker compose start
      redis`, confirm refreshing resumes with current data on the next poll — validated the
      server-side half directly against a live `docker compose up` stack: with 3 scores seeded,
      `docker compose stop redis` made `GET /api/leaderboard/scores` return `500`, then
      `docker compose start redis` restored the exact same 3 ranked entries on the next `GET`. The
      client-side "don't clear the DOM on a failed fetch" behavior was verified by code review
      (the `.catch` block never assigns to `listEl.innerHTML`) — **awaiting user confirmation in a
      real browser**.

**Checkpoint**: All three user stories are functional together — the display is booth-ready.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Bring the rest of the repo's documentation in line with the new feature and confirm
the coverage bar established by prior features still holds

- [X] T015 [P] Update `README.md`'s "Running the app" section to link
      `specs/004-leaderboard-page/quickstart.md` alongside the existing 001/002/003 links
- [X] T016 Run `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out` in
      `app/`; confirm the extended `leaderboard` package still meets the project's existing ≥80%
      statement coverage bar, adding tests for any gap — added `Top`/`TopErr` tests directly in
      `leaderboardtest` (previously only exercised cross-package, showing false 0% coverage);
      total statement coverage across `app/` is 91.1%, `leaderboard` package is 97.5%,
      `leaderboardtest` is 100%, all above the 80% bar
- [X] T017 Run the full `quickstart.md` validation (all three scenarios plus the read-endpoint
      spot checks) end-to-end against a fresh `docker compose up`, confirming no regressions — ran
      against a freshly built `docker compose up -d --build app redis`: empty-state `GET`, seeding
      3 scores, ranked `GET` and `limit`-bounded `GET`, the Redis-outage/recovery cycle (T014), and
      `GET /leaderboard` serving `200 text/html` with the expected `fetch`/`setInterval` wiring —
      all matched expectations exactly. `go test ./...` passes across every package with no
      Testcontainers failures. The stack was torn down (`docker compose down`) after validation.
      Browser-rendered behavior (Scenarios 1-3's visual/DOM assertions) has the same no-browser
      sandbox limitation noted in T010/T012/T014 — **the user should confirm these in a real
      browser to close out constitution Principle IV**, matching how 003 handled the same gap.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No tasks — nothing new to configure
- **Foundational (Phase 2)**: BLOCKS all user stories (the extended `ScoreStore`, its fake, and
  the `GET` handler branch all live here)
- **User Stories (Phase 3-5)**: All depend on Foundational completion
  - US1 (P1) has no dependency on US2/US3
  - US2 (P1) depends on US1's render function existing to wrap in a polling interval
  - US3 (P2) depends on US2's polling loop existing to wrap in failure handling
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational — no dependency on other stories
- **User Story 2 (P1)**: Depends on US1 (wraps its render logic in an interval) and on
  Foundational's endpoint
- **User Story 3 (P2)**: Depends on US2 (wraps its polling loop in failure handling)

### Within Each User Story

- Foundational pieces (`ScoreStore.Top`, the `GET` handler branch) before anything that routes
  through them
- Story complete before moving to the next priority, if working sequentially

### Parallel Opportunities

- T001 and T002 (Foundational) can run in parallel with each other; T003 depends on T001; T005
  depends on T002 and T004
- T008 and T009 (US1) can proceed alongside T007 once T006 lands
- T015 (Polish) can run in parallel with T016/T017

---

## Parallel Example: Foundational Phase

```bash
# T001 and T002 can proceed together (T002 mirrors T001's interface shape):
Task: "Extend ScoreStore with Top(ctx, limit) in app/internal/leaderboard/store.go"
Task: "Extend FakeScoreStore with a matching Top method in app/internal/leaderboard/leaderboardtest/fake_store.go"
```

## Parallel Example: User Story 1

```bash
# Once T006 lands, these can proceed together:
Task: "Style the standings list, loading, and empty states inline in app/main.go"
Task: "Handler test confirming GET /leaderboard is reachable on both listeners in app/main_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 2: Foundational
2. Complete Phase 3: User Story 1
3. **STOP and VALIDATE**: run `quickstart.md` Scenario 1
4. At this point the page shows a correct ranked snapshot on load, but never updates itself while
   open — acceptable for validating the display in isolation, not yet the "dynamically refreshing"
   feature issue 5 actually asked for

### Incremental Delivery

1. Foundational → the read endpoint exists, nothing renders it yet
2. Add US1 → `/leaderboard` shows a correct snapshot on load (MVP for that slice)
3. Add US2 → the page now updates itself live, the feature's core "dynamically refreshes" value
4. Add US3 → the display survives a brief outage without looking broken on stage
5. Polish → docs catch up, coverage bar re-verified

### Note on File Overlap in `app/main.go`

T006 (US1) adds the new `leaderboardPageHTML` constant and route registrations; T011 (US2) and
T013 (US3) both edit that same constant's inline `<script>` block, each adding a distinct,
non-overlapping piece (the polling interval, then the failure handling within it). These are a
coordination point to sequence rather than truly parallelize if one person owns the file, but they
do not conflict in intent.

---

## Notes

- [P] tasks touch different files, or exercise independent behaviors without blocking each other's
  authorship
- [Story] labels map tasks to spec.md's user stories for traceability
- Redis-touching tests MUST use Testcontainers-go, never a mocked client (constitution Principle
  III, non-negotiable); tests against the `ScoreStore` interface from the handler's perspective use
  the in-memory fake instead, which is not the same thing as mocking Redis itself
- Commit after each task or logical group
- The live, auto-refreshing, outage-resilient display cannot be fully automated end-to-end — it is
  part of the Definition of Done per constitution Principle IV and is called out explicitly as its
  own tasks (T010, T012, T014, T017) rather than folded silently into implementation tasks
