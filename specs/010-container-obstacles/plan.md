# Implementation Plan: Container-Themed Obstacles

**Branch**: `010-container-obstacles` | **Date**: 2026-07-07 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/010-container-obstacles/spec.md`

## Summary

Replace the voxel truck obstacles in the Crossy Whale game with two CC BY 4.0 licensed Sketchfab GLB models — "Container Ship" by RM02 and "Container" by Willy Decarpentrie — using the same async `GLTFLoader` pattern already established for the Moby Dock whale. Both models appear in truck lanes; each truck-lane vehicle randomly uses one of the two models (or falls back to the existing voxel truck if either model is absent). The start screen `#cc-attribution` paragraph gains two additional attribution links, and `ATTRIBUTION.md` gains entries for both new models.

## Technical Context

**Language/Version**: JavaScript ES2020 modules (no transpile step); HTML5

**Primary Dependencies**: three.js r160 (ES module importmap, already loaded), `GLTFLoader` addon (already imported in `script.js`)

**Storage**: N/A — static files served by the existing compose stack

**Testing**: Browser-based, observed in the running compose app (Principle IV)

**Target Platform**: Modern browser (desktop + mobile) via compose stack (`docker compose up`)

**Project Type**: Browser game — static frontend under `frontend/game/`

**Performance Goals**: Frame rate with loaded GLB models within 10% of voxel fallback on tested devices (SC-005)

**Constraints**: No build step; no runtime CDN fetch for models (must be vendored in `frontend/game/models/`); models must be standard glTF 2.0 binary (Sketchfab default)

**Scale/Scope**: Two new model files; changes confined to `frontend/game/` (script.js, index.html, ATTRIBUTION.md, models/)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Principle | Status | Notes |
|------|-----------|--------|-------|
| Demo-first delivery | I | **PASS** — swapping voxel trucks for recognisable container models is a direct visual improvement observable within two seconds on the projected wall. | |
| Compose-orchestrated | II | **PASS** — models are vendored static files; no new service or runtime dependency added. | |
| Testcontainers for boundary tests | III | **PASS** — no new backend or service boundary. Change is purely frontend. | |
| Visible-in-browser definition of done | IV | **REQUIRED** — container models must be observed rendering in the running compose stack before shipping. | |
| Vendored-code hygiene | V | **REQUIRED** — ATTRIBUTION.md must be updated with full entries for both new models in the same commit that adds the files; CC BY 4.0 credit must appear on a demo-visible surface (start screen). Failure to do this is a ship blocker. | |
| Stack change? | Stack | **PASS** — no new runtime dependency. GLTFLoader already in the importmap; three.js r160 already used for the whale model. | |

**Complexity violations**: None. No constitution amendments required.

## Project Structure

### Documentation (this feature)

```text
specs/010-container-obstacles/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (changes to repository root)

```text
frontend/game/
├── index.html           # Extend #cc-attribution paragraph with two new credits
├── script.js            # Add model constants, preload function, modify Truck() / Lane('truck')
├── ATTRIBUTION.md       # Add entries for Container Ship (RM02) and Container (Willy Decarpentrie)
└── models/
    ├── moby-dock.glb        # Existing — unchanged
    ├── container-ship.glb   # NEW — copied from host ~/Downloads
    └── container.glb        # NEW — copied from host ~/Downloads
```

**Structure Decision**: Single-project web app; all changes are in the existing `frontend/game/` surface. No new directories beyond a model file drop.

## Complexity Tracking

No constitution violations to justify.
