# Tasks: Container-Themed Obstacles

**Input**: Design documents from `specs/010-container-obstacles/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, quickstart.md ✓

**Tests**: No test tasks — spec does not request automated tests; validation is browser-based per constitution Principle IV.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

---

## Phase 1: Setup (Model Files)

**Purpose**: Vendor the two Sketchfab GLB files into the repository. These are prerequisite for US1 (in-game rendering). US2 and US3 do not depend on this phase and can be started in parallel.

> **⚠️ USER ACTION REQUIRED**: The model files live in `~/Downloads` on the host machine. The sandbox cannot access the host Downloads folder. The user must copy them into the repo manually (or confirm the exact Sketchfab download filenames so the copy command can be documented).

- [x] T001 Copy the "Container Ship" GLB file from `~/Downloads` on the host machine into `frontend/game/models/container-ship.glb` in the repository (exact source filename depends on Sketchfab download — see research.md Model File Naming section)
- [x] T002 Copy the "Container" GLB file from `~/Downloads` on the host machine into `frontend/game/models/container.glb` in the repository

**Checkpoint**: `ls frontend/game/models/` shows `moby-dock.glb`, `container-ship.glb`, and `container.glb`.

---

## Phase 2: User Story 1 — Container Models in Game (Priority: P1) 🎯 MVP

**Goal**: Truck-lane obstacles display 3D container ship or container models (randomly selected per vehicle), falling back to the existing voxel truck when models are unavailable.

**Independent Test**: Start the game, move the whale forward several lanes. Observe that truck-lane obstacles are 3D container-themed models. Console shows `[container-ship] loaded` and `[container] loaded`. With model files removed, voxel trucks render and no errors appear.

### Implementation for User Story 1

- [x] T003 [US1] Add four model constants at the top of `frontend/game/script.js` (after the existing `WHALE_MODEL_*` constants): `CONTAINER_SHIP_MODEL_URL = './models/container-ship.glb'`, `CONTAINER_SHIP_MODEL_TARGET_LENGTH = 200`, `CONTAINER_MODEL_URL = './models/container.glb'`, `CONTAINER_MODEL_TARGET_LENGTH = 100`; also add two module-level null-initialized variables `let containerShipScene = null;` and `let containerScene = null;`
- [x] T004 [US1] Implement `preloadVehicleModels()` in `frontend/game/script.js`: use the existing `GLTFLoader` (already imported) to load `CONTAINER_SHIP_MODEL_URL` and `CONTAINER_MODEL_URL` each in their own `loader.load()` call; on success apply `{ x: Math.PI / 2, y: 0, z: 0 }` rotation, enable `castShadow`/`receiveShadow` on all meshes via `traverse`, auto-scale longest bounding-box axis to the respective `TARGET_LENGTH`, center on XY and place bottom at z=0 (matching the whale model loading pattern in `tryLoadWhaleModel`), then assign `gltf.scene` to `containerShipScene` / `containerScene`; on error call `console.info` with a friendly message and keep the variable null
- [x] T005 [US1] Call `preloadVehicleModels()` once at startup in `frontend/game/script.js`, immediately after the existing `tryLoadWhaleModel(whale)` call
- [x] T006 [US1] Modify the `Truck()` constructor in `frontend/game/script.js`: if `containerShipScene !== null` or `containerScene !== null`, pick one of the non-null options at random (50/50 when both are loaded, or whichever is available when only one loaded), clone it with `.clone(true)`, wrap in a `new THREE.Group()`, and return it; if both are null return the existing voxel truck group unchanged — the existing voxel `Truck()` body becomes the fallback branch

**Checkpoint**: With model files present, truck-lane obstacles display 3D container models. With files absent, voxel trucks render and console shows info-level fallback messages. Game plays normally in both cases.

---

## Phase 3: User Story 2 — CC Attributions on Start Screen (Priority: P2)

**Goal**: The game start screen visibly credits "Container Ship" by RM02 and "Container" by Willy Decarpentrie with CC BY 4.0 links, satisfying the licence requirement before any public showing.

**Independent Test**: Load the game start screen (before entering a name). Confirm both new attribution entries appear alongside the existing Moby Dock credit, all links open correct Sketchfab pages, and layout is readable at 375 px width.

### Implementation for User Story 2

- [x] T007 [US2] Extend the `#cc-attribution` paragraph in `frontend/game/index.html` to add two new credit lines after the existing Moby Dock entry: (1) a hyperlinked `"Container Ship" by RM02` linking to `https://skfb.ly/prrCt`, followed by a CC BY 4.0 link; (2) a hyperlinked `"Container" by Willy Decarpentrie` linking to `https://skfb.ly/FZOL`, followed by a CC BY 4.0 link — match the existing entry's link structure and separators

**Checkpoint**: Start screen shows three attribution entries. All six links (three titles, three licence links) are clickable and point to the correct URLs. No layout overflow at 375 px viewport width.

---

## Phase 4: User Story 3 — Attribution Records in ATTRIBUTION.md (Priority: P3)

**Goal**: `frontend/game/ATTRIBUTION.md` contains complete provenance entries for both new models so future contributors can trace asset origins without inspecting the app at runtime.

**Independent Test**: Read `frontend/game/ATTRIBUTION.md` and confirm both new entries are present with all required fields.

### Implementation for User Story 3

- [x] T008 [P] [US3] Add a "Container Ship" section to `frontend/game/ATTRIBUTION.md` with: title, author (RM02), source URL (`https://skfb.ly/prrCt`), licence (CC BY 4.0), download date (today: 2026-07-07), modifications note (Y-up→Z-up rotation, scaled to 200 world units, centered on XY, bottom at z=0), and the required credit line: `"Container Ship" (https://skfb.ly/prrCt) by RM02 is licensed under Creative Commons Attribution (http://creativecommons.org/licenses/by/4.0/).`
- [x] T009 [P] [US3] Add a "Container" section to `frontend/game/ATTRIBUTION.md` with: title, author (Willy Decarpentrie), source URL (`https://skfb.ly/FZOL`), licence (CC BY 4.0), download date (today: 2026-07-07), modifications note (Y-up→Z-up rotation, scaled to 100 world units, centered on XY, bottom at z=0), and the required credit line: `"Container" (https://skfb.ly/FZOL) by Willy Decarpentrie is licensed under Creative Commons Attribution (http://creativecommons.org/licenses/by/4.0/).`

**Checkpoint**: `frontend/game/ATTRIBUTION.md` has entries for Container Ship (RM02) and Container (Willy Decarpentrie), each with all six metadata fields and the required credit line verbatim.

---

## Phase 5: Polish & End-to-End Verification

**Purpose**: Run the full quickstart.md validation suite against the running compose stack and fix any issues before ship.

- [ ] T010 Start the compose stack (`docker compose up --build`) and verify Scenario 1 from `specs/010-container-obstacles/quickstart.md`: truck-lane obstacles render as 3D container models in the browser; console shows load-success messages
- [ ] T011 [P] Verify Scenario 2 from quickstart.md: with model files temporarily removed, game runs without errors, voxel trucks appear, console shows info-level fallback messages; restore files after
- [x] T012 [P] Verify Scenario 3 from quickstart.md: start screen shows all three attribution credits with correct links; layout is intact at 375 px width
- [x] T013 [P] Verify Scenario 4 from quickstart.md: ATTRIBUTION.md entries for both new models are complete and correct
- [ ] T014 Verify Scenario 5 from quickstart.md: frame rate with models loaded is within 10% of voxel fallback on a test device

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No prior dependencies — start immediately; required for Phase 2 (US1) but NOT for Phases 3 or 4
- **Phase 2 (US1)**: Depends on Phase 1 (model files must be present in repo); independent of Phases 3 and 4
- **Phase 3 (US2)**: No prior dependencies — can start in parallel with Phase 1 and Phase 2
- **Phase 4 (US3)**: No prior dependencies — can start in parallel with any other phase
- **Phase 5 (Polish)**: Depends on all of Phases 1–4 being complete

### User Story Dependencies

- **US1 (P1)**: Requires Phase 1 complete (model files in repo). No dependency on US2 or US3.
- **US2 (P2)**: No dependencies on other stories. Can start immediately.
- **US3 (P3)**: No dependencies on other stories. Can start immediately.

### Within Phase 2 (US1)

Tasks T003 → T004 → T005 → T006 are sequential (all modify `script.js`).

### Parallel Opportunities

- T008 and T009 (ATTRIBUTION.md entries) can run in parallel (different sections of the same file, but the sections don't conflict — take care with adjacent edits)
- T011, T012, T013 (quickstart verification Scenarios 2, 3, 4) can be run in parallel
- US2 (T007) and US3 (T008, T009) can be worked entirely in parallel with Phase 1 + Phase 2

---

## Parallel Execution Example: Fastest Path to Ship

```text
# Stream 1: Model loading (US1)
T001 → T002 → T003 → T004 → T005 → T006

# Stream 2: HTML credits (US2) — starts immediately, no waiting
T007

# Stream 3: ATTRIBUTION.md records (US3) — starts immediately, no waiting
T008 + T009 (parallel within stream)

# After all three streams complete:
T010 → T011/T012/T013 (parallel) → T014
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: copy model files (T001, T002)
2. Complete Phase 2: script.js changes (T003–T006)
3. **Validate**: truck lanes show container models in the running app
4. Ship US1, then layer US2 and US3

### Full Ship (All Stories)

1. Phase 1 + Phase 2 (US1) in one stream
2. Phase 3 (US2) + Phase 4 (US3) in parallel stream
3. Phase 5: full quickstart.md verification
4. Commit T001–T014 in one commit (model files + ATTRIBUTION.md + script.js + index.html together, per constitution Principle V — attribution must ship in the same commit as the vendored asset)

### Important Note on T001/T002

The model file copy must happen on the **host machine** (or by the user confirming the filenames and initiating the copy). The sandbox cannot read `~/Downloads`. This should be the very first step; once the GLB files are in `frontend/game/models/`, all remaining tasks run inside the sandbox normally.

---

## Notes

- No automated tests generated — this feature is validated by direct browser observation (constitution Principle IV)
- `[P]` tasks operate on different files or non-conflicting sections; use caution with T008/T009 (adjacent ATTRIBUTION.md edits)
- Commit model files + ATTRIBUTION.md updates in the same commit as the code changes (constitution Principle V — attribution debt is a ship blocker)
- The hit-detection box (`truck: 105` world units) is intentionally **not** changed — see research.md Hit-Detection Box section
