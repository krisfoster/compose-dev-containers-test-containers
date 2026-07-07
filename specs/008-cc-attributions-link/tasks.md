# Tasks: Add CC Attributions and Leaderboard Link

**Input**: Design documents from `specs/008-cc-attributions-link/`

**Prerequisites**: plan.md, spec.md, research.md, quickstart.md

**Tests**: No test tasks — the feature is validated visually in the browser per constitution Principle IV. No new Go code is introduced; no service boundaries are crossed. Validation scenarios are in `quickstart.md`.

**Organization**: Tasks are grouped by user story. US1 (CC Attribution) is P1 and must ship first; US2 (Leaderboard Link) is P2 and is independently testable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on in-progress tasks)
- **[Story]**: Which user story this task belongs to (US1, US2)

---

## Phase 1: Setup

**Purpose**: Confirm the baseline — game start screen loads correctly before any edits.

- [x] T001 Start compose stack (`docker compose up --build -d`) and confirm `/play` loads in a browser with the name-entry overlay visible and the whale model present

**Checkpoint**: Baseline confirmed — HTML edits can begin

---

## Phase 2: User Story 1 — CC Attribution on Start Screen (Priority: P1) 🎯 MVP

**Goal**: Display the required CC BY 4.0 credit for the Moby Dock whale model on the game's start screen, visible to every player before they begin.

**Independent Test**: Load `http://localhost:8080/play`, confirm the attribution paragraph is visible below the Play button with title, author, Sketchfab link, and CC BY 4.0 licence link — all before entering a name or clicking Play.

### Implementation

- [x] T002 [US1] In `frontend/game/index.html`, after the closing `</form>` tag inside `#name-prompt`, add the CC attribution paragraph:
  ```html
  <p id="cc-attribution">
    <a href="https://sketchfab.com/3d-models/moby-dock-docker-whale-b706010291ca46ad8daca2d4aeb79edd"
       target="_blank" rel="noopener">"Moby Dock (Docker whale)" by Maurice Svay</a>,
    licensed under
    <a href="https://creativecommons.org/licenses/by/4.0/" target="_blank" rel="noopener">CC BY 4.0</a>.
    Scaled and re-oriented for use in this project.
  </p>
  ```

- [x] T003 [P] [US1] In `frontend/game/style.css`, append CSS rules for `#cc-attribution` (font-size: 0.28em, text-align: center, max-width: 320px, opacity: 0.75, line-height: 1.5, link color inherits white)

- [x] T004 [US1] Browser-validate US1: load `/play`, confirm attribution paragraph is visible below Play button, both links open new tabs (Sketchfab and CC BY 4.0 licence page), text does not overlap form or Play button, readable without scrolling on desktop viewport

**Checkpoint**: US1 complete — CC attribution is visible on the start screen. Constitution Principle V compliance gap is closed.

---

## Phase 3: User Story 2 — Leaderboard Link on Start Screen (Priority: P2)

**Goal**: Give players a one-click path from the game start screen to the leaderboard.

**Independent Test**: Load `http://localhost:8080/play`, confirm a clearly labelled leaderboard link appears between the Play button and the attribution, clicks open the leaderboard page in a new tab, and the game tab stays on the start screen.

### Implementation

- [x] T005 [US2] In `frontend/game/index.html`, between the closing `</form>` and the `<p id="cc-attribution">` added in T002, insert the leaderboard link paragraph:
  ```html
  <p id="leaderboard-link-prompt">
    <a href="/leaderboard" target="_blank" rel="noopener">View the leaderboard</a>
  </p>
  ```

- [x] T006 [P] [US2] In `frontend/game/style.css`, append CSS rules for `#leaderboard-link-prompt` (font-size: 0.4em, text-align: center, margin: 0, link color: #7ecbff with underline)

- [x] T007 [US2] Browser-validate US2: load `/play`, confirm leaderboard link appears above the attribution, clicks open `/leaderboard` in a new tab, original tab stays on the start screen, link is visible without scrolling

**Checkpoint**: US2 complete — players can navigate to the leaderboard from the start screen in one click.

---

## Phase 4: Polish & Cross-Cutting Concerns

**Purpose**: Mobile and end-to-end validation across both user stories.

- [x] T008 [P] Mobile viewport validation: resize browser to 375×667 (iPhone SE equivalent), confirm name-entry form, Play button, leaderboard link, and CC attribution are all visible without scrolling and without any element overlap or horizontal overflow

- [x] T009 Start-screen dismiss validation: enter any name and press Play, confirm `#name-prompt` overlay (and its children, including attribution and leaderboard link) dismiss correctly and the game starts normally — no layout regression

- [x] T010 Run all five validation scenarios from `specs/008-cc-attributions-link/quickstart.md` end-to-end and confirm each passes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **US1 (Phase 2)**: Depends on Phase 1 baseline confirmation
  - T002 and T003 can run in parallel (different files: HTML and CSS)
  - T004 depends on both T002 and T003 complete
- **US2 (Phase 3)**: Depends on T002 (inserts HTML relative to the element T002 adds)
  - T005 depends on T002
  - T006 can run in parallel with T004 or T005 (CSS file, independent of HTML state)
  - T007 depends on T005 and T006 complete
- **Polish (Phase 4)**: Depends on all US1 and US2 tasks complete

### User Story Dependencies

- **US1 (P1)**: No dependency on US2 — independently testable after T004
- **US2 (P2)**: HTML insertion (T005) depends on T002 having added `#cc-attribution` as the anchor point; CSS (T006) is independent

### Parallel Opportunities

- T002 (HTML) and T003 (CSS) — different files, both adding US1 content
- T006 (CSS for leaderboard link) can start as soon as T001 completes
- T008 and T009 can run in parallel (different validation dimensions)

---

## Parallel Example: US1

```bash
# T002 and T003 can launch simultaneously:
Task T002: "Add #cc-attribution paragraph to frontend/game/index.html"
Task T003: "Add #cc-attribution CSS rules to frontend/game/style.css"
# T004 runs after both complete.
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: Setup (T001)
2. Complete Phase 2: US1 (T002 → T003 in parallel → T004)
3. **STOP and VALIDATE**: Attribution visible and correct in browser
4. Ship if only the licence compliance gap needs to close

### Incremental Delivery

1. Phase 1 (T001) → baseline confirmed
2. Phase 2 (T002–T004) → CC attribution live, Principle V satisfied → demo-ready
3. Phase 3 (T005–T007) → leaderboard link added → full feature complete
4. Phase 4 (T008–T010) → polish validated → ship

---

## Notes

- [P] tasks touch different files (HTML vs CSS) and carry no cross-task dependency
- [US1] and [US2] labels map to spec.md User Story 1 and User Story 2
- Both user stories edit the same two files (`index.html`, `style.css`) — no branching required within the feature
- The Go backend, Redis, and compose configuration are untouched
- No new vendored assets are introduced — no `ATTRIBUTION.md` update needed
- Commit once after all tasks pass quickstart.md validation (per constitution: commits only when user asks)
