# Tasks: Phone Touch Controls for Whale Movement

**Input**: Design documents from `specs/021-phone-touch-controls/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, quickstart.md ✓

**Tests**: No test tasks generated — spec does not request TDD; validation is performed manually in the browser against the running compose stack (Constitution Principle IV).

**Organization**: Tasks are grouped by user story. All three user stories are independently implementable because they touch different files (US1: `script.js`; US2: `style.css`; US3: validation only).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1, US2, US3 — maps to user stories in `spec.md`

---

## Phase 1: Setup

**Purpose**: Confirm the baseline game is running before making changes.

- [x] T001 Start the compose stack (`docker compose up`) and open the game in a browser — confirm the whale moves with the D-pad buttons and keyboard arrows, and the game-over screen works; this is the baseline for all validation in later phases

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: No blocking prerequisites exist for this feature — all user story changes are isolated to separate files with no shared infrastructure to build first.

**⚠️ NOTE**: Phase 1 (baseline validation) must pass before moving to user story phases.

---

## Phase 3: User Story 1 — Tap to Move in any Direction (Priority: P1) 🎯 MVP

**Goal**: A phone user can tap anywhere on the game canvas to move the whale one step in the tapped direction. Tapping to the side rotates the whale to face that way first, then jumps. All existing movement rules (boundaries, obstacles, name-guard) apply.

**Independent Test**: Open the game in Chrome DevTools with touch emulation enabled (or on a phone). Enter a name and tap to the right, left, above, and below the whale — verify the whale moves correctly in all four cardinal directions (quickstart.md scenarios 3–6).

### Implementation for User Story 1

- [x] T002 [US1] Add `touchstart` listener to `renderer.domElement` in `frontend/game/script.js`: gate on `window.matchMedia('(pointer: coarse)').matches`, project `whale.position.clone().project(camera)` to screen coords, compute `dx`/`dy` from tap to whale centre, apply 10 CSS-pixel dead zone, resolve to cardinal direction via dominant-axis test (`Math.abs(dx) > Math.abs(dy)` → left/right; else forward/backward based on sign of `dy`), then call `move(direction)` — place this block after `renderer.domElement` is appended to the body, alongside the existing keyboard and button listeners

### Validation for User Story 1

- [ ] T003 [P] [US1] Validate all four tap directions in browser (quickstart.md scenarios 3–6): tap right of whale → whale rotates right and jumps right; tap left → rotates left and jumps left; tap above → moves forward; tap below → rotates and moves backward; also verify lane counter increments on each forward move and boundary blocks work (cannot move past column 0 or 16)

**Checkpoint**: After T002 + T003 pass, US1 is fully functional. Phone users can navigate the whale.

---

## Phase 4: User Story 2 — On-Screen Buttons Hidden on Phone (Priority: P2)

**Goal**: The D-pad button overlay (`#controls`) is invisible on phones and unchanged on desktop. No layout disruption on either platform.

**Independent Test**: Open the game on a phone (or DevTools phone preset) — buttons absent. Open on desktop — buttons present and clickable as before (quickstart.md scenarios 1–2).

### Implementation for User Story 2

- [x] T004 [P] [US2] Add a CSS media query to `frontend/game/style.css` that hides the controls on coarse-pointer devices: `@media (pointer: coarse) { #controls { display: none; } }`

### Validation for User Story 2

- [ ] T005 [P] [US2] Validate button visibility in browser (quickstart.md scenarios 1–2): load game in DevTools with phone preset — `#controls` div is not visible; load game in desktop mode — `#controls` div is visible with all four arrow buttons; also rotate DevTools device to landscape and confirm buttons remain hidden (scenario 1, acceptance scenario 3)

**Checkpoint**: After T004 + T005 pass, US2 is complete. Phone screen is uncluttered; desktop experience is unchanged.

---

## Phase 5: User Story 3 — Normalisation Correctness and Edge Cases (Priority: P3)

**Goal**: Every tap at any angle resolves to exactly one of the four cardinal directions. The dominant-axis algorithm from T002 is already in place; this phase verifies its correctness for diagonal inputs, the dead zone, multi-touch, scroll suppression, and orientation changes.

**Independent Test**: All 11 scenarios in quickstart.md pass on a phone (or DevTools touch emulation).

**Note**: No new implementation code is expected in this phase. If any scenario fails, the fix belongs in `frontend/game/script.js` (touch listener from T002) or `frontend/game/style.css` (touch-action from existing CSS).

### Validation for User Story 3

- [ ] T006 [P] [US3] Validate diagonal tap normalisation (quickstart.md scenario 7): tap at roughly 45° diagonals in all four corners around the whale — confirm whale always moves in one of the four cardinal directions, never diagonally; confirm the direction is consistent (same angle → same result across multiple taps)

- [ ] T007 [P] [US3] Validate dead zone (quickstart.md scenario 8): tap as close to the whale's centre as possible — confirm the whale does not move and does not rotate; the 10px dead zone in `frontend/game/script.js` must suppress the event

- [ ] T008 [P] [US3] Validate multi-touch handling (quickstart.md scenario 9): touch with two fingers simultaneously — one to the right, one to the left of the whale — confirm only one directional move is triggered (the first touch point), not two moves or a cancel; verify `e.touches[0]` is the only touch point used in `frontend/game/script.js`

- [ ] T009 [P] [US3] Validate scroll suppression (quickstart.md scenario 10): tap rapidly on the canvas while the game is active — confirm the page does not scroll or zoom; the existing `canvas { touch-action: none; }` in `frontend/game/style.css` is responsible — confirm it is still present and the canvas element has no inline style overriding it

- [ ] T010 [P] [US3] Validate portrait and landscape orientation (quickstart.md scenario 11): play several taps in portrait orientation, then rotate phone to landscape and play more — confirm tap controls work in both orientations and buttons remain hidden in both; confirm the existing `onResize` handler in `frontend/game/script.js` keeps the canvas filling the viewport so projected whale coordinates remain accurate

**Checkpoint**: After all T006–T010 pass, US3 is complete. Touch controls are correct for all input angles, orientations, and edge cases.

---

## Final Phase: Polish & End-to-End Validation

**Purpose**: Full end-to-end run of the quickstart and verification that the feature has not regressed any existing behaviour.

- [ ] T011 Run the complete quickstart.md validation sequence (all 11 scenarios) on a physical phone (not just DevTools emulation) to confirm real touch events behave correctly, and confirm the sbx port-publish step works (quickstart.md "Exposing the game to a phone" section)

- [ ] T012 [P] Confirm no regression in desktop behaviour: keyboard arrow keys still move the whale; D-pad buttons still work; game-over screen appears and retry resets correctly; leaderboard score is submitted on game over

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: No tasks — Phase 1 completion is the only gate
- **US1 (Phase 3)**: Depends on Phase 1 baseline passing — no dependency on US2 or US3
- **US2 (Phase 4)**: Depends on Phase 1 baseline passing — no dependency on US1 or US3; T004 can start in parallel with T002
- **US3 (Phase 5)**: Depends on T002 (the touch listener must exist) — validation only, no new code
- **Polish (Final Phase)**: Depends on all user story phases passing

### User Story Dependencies

- **US1 (P1)**: Independent — `frontend/game/script.js` only
- **US2 (P2)**: Independent — `frontend/game/style.css` only; can start immediately after T001, in parallel with T002
- **US3 (P3)**: Depends on T002 (US1 implementation) — validation tasks can all run in parallel once T002 is done

### Parallel Opportunities

- T004 (US2 CSS) and T002 (US1 JS) touch different files — run in parallel after T001 passes
- T003 (US1 validation) and T005 (US2 validation) can run in parallel once T002 and T004 are done respectively
- All T006–T010 (US3 validations) can run in parallel once T002 is done

---

## Parallel Example

```text
After T001 (baseline confirmed):

  ┌─ T002 [US1] Add touchstart listener (script.js)
  │     └─ T003 [US1] Validate four-direction taps
  │
  └─ T004 [US2] Add CSS media query (style.css)       ← parallel with T002
        └─ T005 [US2] Validate button hiding

After T002 done:

  ┌─ T006 [US3] Validate diagonal normalisation
  ├─ T007 [US3] Validate dead zone                   ← all parallel
  ├─ T008 [US3] Validate multi-touch
  ├─ T009 [US3] Validate scroll suppression
  └─ T010 [US3] Validate orientation

After all US phases pass:
  T011 Physical phone end-to-end
  T012 Desktop regression check                      ← parallel with T011
```

---

## Implementation Strategy

### MVP First (User Story 1 + 2 Only)

1. Complete Phase 1: Baseline validation (T001)
2. Complete Phase 3 + Phase 4 in parallel: T002 + T004, then T003 + T005
3. **STOP and VALIDATE**: Both US1 and US2 work independently
4. Demo to booth presenter — phones can now play

### Incremental Delivery

1. T001 → T002 + T004 (parallel) → T003 + T005 (parallel) — core feature working
2. T006–T010 (parallel) — edge case correctness confirmed
3. T011 + T012 — production-ready on physical hardware

### Notes

- T002 is the only implementation task with non-trivial code. Refer to `research.md` for the exact projection formula and `data-model.md` for the direction resolution table.
- US3 has no new code tasks — if any scenario T006–T010 fails, the fix is a one-line correction to the listener in T002.
- The existing `canvas { touch-action: none; }` in `style.css` already prevents scroll (T009). Verify it is present before writing any `e.preventDefault()` code.
- All validation must be done in a running browser against the compose stack, not by static analysis (Constitution Principle IV).
