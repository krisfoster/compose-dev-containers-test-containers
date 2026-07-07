# Feature Specification: Container-Themed Obstacles

**Feature Branch**: `010-container-obstacles`

**Created**: 2026-07-07

**Status**: Draft

**Input**: User description: "look at issue 6 in issues.md. I have downloaded a couple of extra models in ~/Downloads. Add them to the game to replace the trucks. The CC attributions need to be added to the start screen of the app."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See Container-Themed Vehicles in the Game (Priority: P1)

A player starts a game session and encounters obstacles that look like container ships and shipping containers rather than generic voxel trucks. The obstacles still move and behave the same way — they cross the lanes and the player must dodge them — but they are now visually themed around the Docker/container world.

**Why this priority**: The visual swap is the core of this feature. Everything else depends on it being in place first.

**Independent Test**: Can be fully tested by starting a game and observing that vehicle obstacles in truck lanes display the container ship and/or container 3D models rather than the built-in voxel truck shapes.

**Acceptance Scenarios**:

1. **Given** a player opens the game and begins play, **When** a truck-lane row appears, **Then** the obstacles on that row display a 3D container ship or shipping container model instead of the voxel truck shape.
2. **Given** the container ship or container model fails to load (network error, missing file), **When** the game starts, **Then** the game falls back to the existing voxel truck shape with no crash or visible error to the player.
3. **Given** a player plays on a mobile device, **When** container-ship obstacles appear, **Then** the models render at an appropriate size and do not cause performance degradation compared to the voxel fallback.

---

### User Story 2 - CC Attributions Visible on the Start Screen (Priority: P2)

A player arrives at the game start screen (the name-entry view) and can see attribution credits for the 3D models used in the game. The attributions satisfy the CC BY 4.0 licence requirements for the Container Ship and Container models, as well as the existing Moby Dock whale credit already required by issue 12.

**Why this priority**: The CC BY 4.0 licence on both new models legally requires visible attribution before any public showing. P2 only because P1 must ship before the attribution is meaningful.

**Independent Test**: Can be fully tested by loading the game start screen and confirming that each required credit line is present and legible without launching a game round.

**Acceptance Scenarios**:

1. **Given** the game start screen is open, **When** a player views the page, **Then** a credits section lists "Container Ship" by RM02 with a link to `https://skfb.ly/prrCt`, licensed CC BY 4.0.
2. **Given** the game start screen is open, **When** a player views the page, **Then** the credits section also lists "Container" by Willy Decarpentrie with a link to `https://skfb.ly/FZOL`, licensed CC BY 4.0.
3. **Given** the start screen is viewed on a small mobile screen, **When** the credits are rendered, **Then** they are readable without horizontal scrolling and do not obscure the name-entry form.

---

### User Story 3 - Attribution Records Updated (Priority: P3)

The project's ATTRIBUTION.md is updated to record the new models with all required metadata so that future contributors can trace the provenance of every asset without inspecting the game at runtime.

**Why this priority**: Needed for licence compliance and project hygiene (constitution Principle V), but not visible to end players and therefore lowest priority after the in-game and in-UI credits.

**Independent Test**: Can be fully tested by reading `frontend/game/ATTRIBUTION.md` and confirming entries for both new models.

**Acceptance Scenarios**:

1. **Given** a developer reads `frontend/game/ATTRIBUTION.md`, **When** they look for the Container Ship entry, **Then** they find: title, author (RM02), source URL, licence (CC BY 4.0), download date, and a note of any transformations applied in-game.
2. **Given** a developer reads `frontend/game/ATTRIBUTION.md`, **When** they look for the Container entry, **Then** they find the same metadata fields for Willy Decarpentrie's model.

---

### Edge Cases

- What happens when both new model files are missing at page load? (The game must still be fully playable using the voxel fallback.)
- How should hit-detection box sizes be adjusted if the loaded model's bounding box differs significantly from the current voxel truck size?
- What if one model loads but the other does not? (Each model loads independently; partial load is acceptable.)
- How are the attribution links displayed on very narrow screens (< 320 px wide)?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The game MUST display 3D model obstacles (container ship and/or shipping container) in truck-lane rows when the model files are present and successfully loaded.
- **FR-002**: The game MUST fall back to the existing voxel truck shape when a model file cannot be loaded, with no player-visible error.
- **FR-003**: Each loaded 3D model MUST be scaled and oriented so that it fits within the lane width and matches the existing obstacle movement behaviour.
- **FR-004**: The start screen MUST display a visible attribution section that credits "Container Ship" by RM02 (link: `https://skfb.ly/prrCt`, CC BY 4.0) and "Container" by Willy Decarpentrie (link: `https://skfb.ly/FZOL`, CC BY 4.0).
- **FR-005**: The attribution section on the start screen MUST be legible on both desktop and mobile viewport sizes.
- **FR-006**: `frontend/game/ATTRIBUTION.md` MUST be updated with an entry for each new model, covering: title, author, source URL, licence, download date, and any in-game transformations.
- **FR-007**: The new model files MUST be vendored into the repository under `frontend/game/models/` so the game works offline (from a fresh clone + `docker compose up`), not fetched at runtime from an external URL.

### Key Entities

- **Container Ship model**: The GLB file for "Container Ship" by RM02 (CC BY 4.0), vendored locally. Used as the primary obstacle replacement for truck-lane vehicles.
- **Container model**: The GLB file for "Container" by Willy Decarpentrie (CC BY 4.0), vendored locally. Used as an additional or alternative obstacle for truck-lane vehicles.
- **Truck obstacle**: The current voxel truck shape built from BoxGeometry. Retained as a fallback when model loading fails.
- **Start screen credits section**: A UI element on the name-entry screen that lists all CC-licensed asset attributions with author names and source links.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In a running game session, truck-lane obstacles visually display container ship or container models rather than voxel trucks 100% of the time when the model files are present.
- **SC-002**: When model files are absent, the game starts and runs to completion without any player-visible error, at a rate of 100%.
- **SC-003**: The start screen attribution section is visible without scrolling on viewports ≥ 375 px wide.
- **SC-004**: Both new model attributions (Container Ship by RM02, Container by Willy Decarpentrie) are present and correct in `ATTRIBUTION.md` before the feature ships.
- **SC-005**: Frame rate during game play with the loaded 3D models is within 10% of frame rate with the voxel fallback on the same device.

## Assumptions

- The user has downloaded the Container Ship and Container GLB files from Sketchfab and they are available at `~/Downloads` on the host machine; they will be copied into `frontend/game/models/` as part of implementation.
- The GLB files are in a format compatible with three.js `GLTFLoader` (standard glTF 2.0 binary, no proprietary extensions required).
- The container ship model is used for truck-lane obstacles; the container model may be used for the same lanes or for an additional obstacle type — exact placement is decided at implementation time.
- The existing attribution section added in feature 008 (CC attributions + leaderboard link) is the surface to extend; a new section is not required.
- The voxel truck fallback is preserved; this feature does not delete that code path.
- Mobile performance is acceptable given that the whale GLB already loads and renders at acceptable frame rates on tested devices.
