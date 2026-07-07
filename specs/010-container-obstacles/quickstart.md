# Quickstart Validation Guide: Container-Themed Obstacles

This guide describes how to verify the feature works end-to-end from a fresh checkout.

## Prerequisites

1. Docker Desktop running.
2. The two model files copied into the repo:
   - `frontend/game/models/container-ship.glb` (downloaded from https://skfb.ly/prrCt)
   - `frontend/game/models/container.glb` (downloaded from https://skfb.ly/FZOL)
3. No leftover containers from a previous session: `docker compose down -v` (optional clean slate).

## Start the stack

```bash
docker compose up --build
```

Wait until the frontend service reports it is serving (nginx or the Go backend, whichever hosts `frontend/game/`).

## Scenario 1 — Container models appear in-game (SC-001, FR-001, FR-003)

1. Open the game in a browser (e.g. `http://localhost:8080` — check compose output for the actual port).
2. Enter a name and press **Play**.
3. Move the whale forward several lanes.
4. **Expected**: When a truck lane appears, the vehicles are visually a container ship or a shipping container (3D models), not the flat voxel truck shape.
5. **Expected**: Obstacles move across the lane at the normal speed and direction.
6. Open the browser dev console — **no errors** should appear. The log should include messages like `[container-ship] loaded ./models/container-ship.glb` and `[container] loaded ./models/container.glb`.

## Scenario 2 — Voxel fallback when models are missing (SC-002, FR-002)

1. Temporarily rename or remove `frontend/game/models/container-ship.glb` and `frontend/game/models/container.glb`.
2. Reload the game, enter a name, and play.
3. **Expected**: The game loads and plays normally. Truck-lane obstacles show the voxel truck shape.
4. **Expected**: The dev console shows `console.info` messages like `[container-ship] no model at ./models/container-ship.glb, keeping voxel truck.` — no thrown errors, no blank obstacles.
5. Restore the model files after this test.

## Scenario 3 — Attributions on the start screen (FR-004, FR-005, SC-003)

1. Load the game start screen (before entering a name).
2. **Expected**: The attribution area shows three credits:
   - "Moby Dock (Docker whale)" by Maurice Svay — CC BY 4.0 link
   - "Container Ship" by RM02 — link to https://skfb.ly/prrCt — CC BY 4.0 link
   - "Container" by Willy Decarpentrie — link to https://skfb.ly/FZOL — CC BY 4.0 link
3. **Expected**: All credit links are clickable and open the correct Sketchfab pages in a new tab.
4. Resize the browser to 375 px wide. **Expected**: All attribution text remains visible without horizontal scrolling.

## Scenario 4 — ATTRIBUTION.md records (SC-004)

Open `frontend/game/ATTRIBUTION.md` and confirm:
- An entry for "Container Ship" by RM02: title, author, source URL (`https://skfb.ly/prrCt`), licence (CC BY 4.0), download date, and modifications note.
- An entry for "Container" by Willy Decarpentrie: same fields with source URL `https://skfb.ly/FZOL`.

## Scenario 5 — Performance baseline (SC-005)

1. Play the game with both models loaded. Use the browser's performance profiler or the FPS counter (if available) to observe frame rate during active gameplay with truck lanes on screen.
2. Temporarily remove the model files and repeat.
3. **Expected**: Frame rate with models loaded is within 10% of the voxel baseline on the test device.

## Definition of Done (Principle IV)

The feature is **not** complete until Scenarios 1, 3, and 4 above have been observed passing in the running app in a browser. Scenarios 2 and 5 must also pass. Screenshots or a screen recording of the container-model obstacles in a truck lane are the accepted artefact.
