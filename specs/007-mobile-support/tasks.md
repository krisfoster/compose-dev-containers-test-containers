# Tasks: Mobile Support

**Input**: Design documents from `specs/007-mobile-support/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, quickstart.md ✓

**Tests**: No dedicated test tasks — browser-based validation per quickstart.md is the definition of done (Constitution Principle IV). All validation scenarios reference `specs/007-mobile-support/quickstart.md`.

**Organization**: Tasks are grouped by user story. The feature modifies four files: `frontend/game/index.html`, `frontend/game/style.css`, `frontend/game/script.js`, and the `gettingStartedPageHTML` constant in `app/main.go`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies between these tasks)
- **[Story]**: Which user story this task belongs to
- All file paths are repo-relative

---

## Phase 1: Foundational (Blocking Prerequisites)

**Purpose**: Changes that MUST be in place before any user-story work can be validated on a mobile device

**⚠️ CRITICAL**: All three user stories that touch the game page (US1–US3) require T001 and T002. US4 is independent and can begin immediately.

- [X] T001 [P] Add `<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">` to the `<head>` of `frontend/game/index.html` (immediately after the charset meta tag, before the title)
- [X] T002 [P] Add a `window.addEventListener('resize', onResize)` handler in `frontend/game/script.js` — the `onResize` function must call `renderer.setSize(window.innerWidth, window.innerHeight)` and update the camera's orthographic frustum: set `camera.left = window.innerWidth / -2`, `camera.right = window.innerWidth / 2`, `camera.top = window.innerHeight / 2`, `camera.bottom = window.innerHeight / -2`, then call `camera.updateProjectionMatrix()`. Place the handler after the renderer is initialised (after line 191). T001 and T002 are in different files and can be done in parallel.

**Checkpoint**: Viewport meta is present; rotating the device in DevTools now resizes the renderer canvas correctly.

---

## Phase 2: User Story 1 — Tap-to-Move Controls (Priority: P1) 🎯 MVP

**Goal**: A phone player can tap the four directional buttons to move the whale, identically to keyboard arrow keys.

**Independent Test**: Open `http://localhost:8080/play` in Chrome DevTools mobile emulation (landscape), enter a name, tap each directional button, and confirm the whale moves one step per tap with no double-movement. See quickstart.md Scenario 1.

- [X] T003 [P] [US1] Add `touch-action: none` to the `canvas` element selector in `frontend/game/style.css` so the browser does not intercept touch events on the game canvas for its own scroll or zoom gestures. Add it as a new rule: `canvas { touch-action: none; display: block; }` (the `display: block` removes the default inline gap beneath the canvas element).
- [X] T004 [US1] Add `touchstart` event listeners to each of the four directional buttons in `frontend/game/script.js`, placed immediately after the existing `click` handlers at lines 576–582. Each listener must call `e.preventDefault()` (to suppress the synthesised mouse-click sequence mobile browsers generate after touch) and then call `move(direction)` with the corresponding direction string. Example for forward: `document.getElementById('forward').addEventListener('touchstart', (e) => { e.preventDefault(); move('forward'); });`. Add the same pattern for `backward`, `left`, and `right`. T003 (style.css) and T004 (script.js) are in different files and can be done in parallel.

**Checkpoint**: Tapping any directional button on a touch device moves the whale exactly once per tap. No double-movement. Existing keyboard controls still work unchanged.

---

## Phase 3: User Story 2 — Portrait Rotation Prompt (Priority: P2)

**Goal**: When a phone is held in portrait orientation, a full-screen overlay appears asking the player to rotate. When the device is rotated to landscape, the overlay disappears.

**Independent Test**: In Chrome DevTools mobile emulation, switch to portrait. Confirm the overlay appears and the controls are not accessible beneath it. Switch to landscape; confirm the overlay disappears. See quickstart.md Scenario 2.

- [X] T005 [P] [US2] Add a `<div id="rotate-prompt">` element to `frontend/game/index.html`. Place it immediately before the closing `</body>` tag (after the `<div class="credits">` block). The div should contain a phone rotation icon (use a Unicode emoji such as `📱↻` or a short phrase like `↻`) and an instruction line: `<div id="rotate-prompt"><span class="rp-icon">↻</span><p>Please rotate your device to landscape</p></div>`.
- [X] T006 [P] [US2] Add CSS for `#rotate-prompt` in `frontend/game/style.css`. The overlay must be: `position: fixed; inset: 0; z-index: 100; display: none; flex-direction: column; align-items: center; justify-content: center; background: rgba(0, 0, 0, 0.92); color: white; font-family: inherit; text-align: center; gap: 16px;`. Add a `.rp-icon` rule: `font-size: 3em;`. The element starts hidden (`display: none`) and is shown by JS. T005 (index.html) and T006 (style.css) are in different files and can be done in parallel.
- [X] T007 [US2] Add a `checkOrientation()` function in `frontend/game/script.js` (placed after the resize handler added in T002). The function sets `document.getElementById('rotate-prompt').style.display` to `'flex'` when `window.innerWidth < window.innerHeight`, and to `'none'` otherwise. Call `checkOrientation()` from inside the existing `onResize` function (T002) so it runs on every resize/rotation event. Also call `checkOrientation()` once immediately after the function is defined so the correct state is applied on page load.

**Checkpoint**: Portrait overlay appears instantly on portrait orientation; disappears instantly on landscape. Game canvas is not interactive while the overlay is visible.

---

## Phase 4: User Story 3 — Full-Screen Landscape Play (Priority: P2)

**Goal**: In landscape orientation, the game canvas fills the entire visible viewport with no white gaps, scroll bars, or browser-chrome overflow.

**Independent Test**: In Chrome DevTools mobile emulation (iPhone 14 landscape, 844×390), confirm the canvas occupies the full viewport with no scroll bars or white edges. See quickstart.md Scenario 3.

- [X] T008 [US3] Update `frontend/game/style.css` to apply the mobile viewport height fallback stack. Find the `body` rule and add the three height declarations as sequential lines: `height: 100vh; height: 100svh; height: 100dvh;`. Do the same for the `html` selector (add or create the rule). Also ensure `margin: 0; overflow: hidden;` remain on `body` (they already exist — confirm, do not remove). The three height lines use CSS cascade: each line overrides the previous one on browsers that support the newer unit, so `dvh` wins on modern browsers and `vh` is the safe fallback on older ones.

**Checkpoint**: Game canvas fills the entire visible viewport in landscape. No white strip at the bottom from the retractable browser address bar. No scroll bars.

---

## Phase 5: User Story 4 — Home Page QR Code Layout (Priority: P3)

**Goal**: The landing page at `/` shows a two-column layout (nav links left, QR code right) and the QR image auto-refreshes when the active code changes.

**Independent Test**: Open `http://localhost:8080/` in a browser. Confirm 2-column layout. Rotate the QR code via `/host/rotate` and confirm the home page image updates within 5 seconds. See quickstart.md Scenarios 4, 5, and 6.

- [X] T009 [US4] Rewrite the `gettingStartedPageHTML` string constant in `app/main.go` (lines 215–234) to use a two-column CSS Grid layout. The `<style>` block must define: `.layout { display: grid; grid-template-columns: 1fr 1fr; gap: 2rem; align-items: start; }` and `@media (max-width: 600px) { .layout { grid-template-columns: 1fr; } }`. The `<body>` must contain a `<div class="layout">` with two children: a `<div class="nav-col">` holding the three existing buttons (Play, Host, Leaderboard), and a `<div class="qr-col">` containing `<img id="qr-img" src="/qr.png" alt="QR code to join" width="280" height="280">`. Preserve all existing button styles and links. Keep the same page title and `<h1>Crossy Whale</h1>` heading.
- [X] T010 [US4] Add a JavaScript `<script>` block inside `gettingStartedPageHTML` in `app/main.go` (inside the `<body>`, after the layout div) that polls `/qr.png` on a 4-second `setInterval`. On each tick: create a new `Image` object, set its `onload` to update `document.getElementById('qr-img').src` to the new timestamped URL (e.g. `/qr.png?t=` + Date.now()), set its `onerror` to do nothing (leaving the currently displayed image untouched), then set the `Image` object's `src` to the timestamped URL to trigger the fetch. This pattern avoids flashing a broken image when `/qr.png` returns 503. T010 depends on T009 (the `#qr-img` element must exist before the script references it).

**Checkpoint**: Two-column layout visible at full width. QR image updates automatically after a `/host/rotate` without page reload. On narrow screens the layout collapses to single column.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: End-to-end validation of all scenarios and a final console-error sweep.

- [ ] T011 Run quickstart.md Scenario 1 (tap controls — real or emulated device) and Scenario 2 (portrait prompt — DevTools emulation); confirm both pass before marking this task done.
- [ ] T012 [P] Run quickstart.md Scenario 3 (full-screen landscape) and Scenario 4 (home page 2-col layout); confirm both pass.
- [ ] T013 [P] Run quickstart.md Scenario 5 (QR auto-refresh) and Scenario 6 (QR missing state — restart the stack to clear Redis before testing); confirm both pass.
- [ ] T014 Open `http://localhost:8080/play` in Chrome DevTools mobile emulation, run the full game flow (name entry → play → game over → replay), and confirm zero JavaScript errors appear in the browser console (DevTools → Console tab).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 1)**: No dependencies — T001 and T002 can start immediately and in parallel
- **US1 (Phase 2)**: Requires T001 and T002 complete. T003 and T004 can then run in parallel with each other (different files)
- **US2 (Phase 3)**: Requires T001 and T002 complete. T005 and T006 can run in parallel (different files); T007 depends on both T005 and T006 being done
- **US3 (Phase 4)**: Requires T001 and T002 complete. T008 is independent of US1 and US2 tasks (different section of style.css, no JS dependency)
- **US4 (Phase 5)**: Fully independent of all other phases — T009 and T010 only touch `app/main.go`. T009 must complete before T010.
- **Polish (Phase 6)**: Depends on all user story phases being complete

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational — no dependency on US2, US3, US4
- **US2 (P2)**: Can start after Foundational — no dependency on US1, US3, US4
- **US3 (P2)**: Can start after Foundational — no dependency on US1, US2, US4
- **US4 (P3)**: Can start immediately (no Foundational dependency) — no dependency on US1, US2, US3

### Within Each User Story

- US1: T003 and T004 can be done in parallel (different files); T004 should be verified in DevTools after
- US2: T005 and T006 in parallel, then T007 after both
- US3: T008 is a single task
- US4: T009 then T010 (same file, T010 extends T009's structure)

### Parallel Opportunities

- T001 ∥ T002 (index.html vs script.js)
- T003 ∥ T004 (style.css vs script.js)
- T005 ∥ T006 (index.html vs style.css)
- T012 ∥ T013 (independent validation scenarios)
- US4 can run in parallel with all of US1–US3 (different file, no shared state)

---

## Parallel Example: User Story 2

```text
# After T001 and T002 are complete, launch both in parallel:
Task T005: Add #rotate-prompt div to frontend/game/index.html
Task T006: Add #rotate-prompt CSS to frontend/game/style.css

# Once both T005 and T006 are done:
Task T007: Add checkOrientation() to frontend/game/script.js and wire to events
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Foundational (T001, T002)
2. Complete Phase 2: US1 (T003, T004)
3. **STOP and VALIDATE**: Open the game in Chrome DevTools mobile emulation — tap each directional button and confirm whale movement
4. Ship/demo tap controls independently; US2–US4 are additive

### Incremental Delivery

1. Foundational (T001, T002) → Validates that the game renders correctly after device rotation
2. US1 (T003, T004) → Tap controls working → **Demo: booth attendees can play on their phones**
3. US2 (T005–T007) → Portrait prompt → **Demo: phones held in portrait show a clear rotation prompt**
4. US3 (T008) → Full-screen landscape → **Demo: game fills the entire phone screen**
5. US4 (T009, T010) → Home page QR layout → **Demo: presenter's landing page shows QR code on right**

### Parallel Strategy (two agents)

- **Agent A**: T001 → T002 → T003 → T004 → T007 (all script.js and index.html changes)
- **Agent B**: T009 → T010 (app/main.go changes — completely independent)
- Then Agent A: T005 → T006 → T008 (style.css and remaining HTML)
- Merge and validate together in Phase 6

---

## Notes

- [P] tasks = different files, no inter-task dependencies — safe to run concurrently
- [Story] label maps each task to its user story for traceability
- US4 (app/main.go) is fully isolated from US1–US3 (frontend/game/) — always safe to do in parallel or in any order
- Existing keyboard controls (lines 584–601 of script.js) must remain untouched; touchstart handlers are additive
- The resize handler (T002) and checkOrientation() (T007) must both live in script.js — sequence them so checkOrientation() is defined before it is called from inside onResize
- Validation uses browser DevTools, not automated tests — Constitution Principle IV
