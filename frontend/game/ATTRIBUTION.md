# Attribution

This directory contains code and assets from two upstream sources.

## Game code

`index.html`, `script.js`, `style.css`, and `favicon.ico` are vendored from:

- **Repo**: [GeekBoySupreme/crossy-road](https://github.com/GeekBoySupreme/crossy-road)
- **Author**: [GeekBoySupreme](https://github.com/GeekBoySupreme)
- **License**: MIT (see `LICENSE` in this directory, kept intact)
- **Live original**: https://crossy-road.glitch.me/
- **Original commit basis**: master branch as of 2020-08-16

## 3D model

`models/moby-dock.glb` is:

- **Title**: Moby Dock (Docker whale)
- **Author**: Maurice Svay ([@mauricesvay on Sketchfab](https://sketchfab.com/mauricesvay))
- **Source**: https://sketchfab.com/3d-models/moby-dock-docker-whale-b706010291ca46ad8daca2d4aeb79edd
- **License**: [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/)
- **Downloaded**: 2026-07-06 as `moby_dock_docker_whale.glb`, renamed to `moby-dock.glb`
- **Modifications**: none to the model file itself. In-game, it is loaded via `GLTFLoader`, rotated +90° around the X axis to convert from glTF's Y-up to the game's Z-up, auto-scaled to a target longest-axis length of 60 world units, and centered on the XY plane with its bottom sitting at z=0.

Required credit line (surface this somewhere visible in the running app before shipping):

> "Moby Dock (Docker whale)" by Maurice Svay, https://sketchfab.com/3d-models/moby-dock-docker-whale-b706010291ca46ad8daca2d4aeb79edd, licensed under CC BY 4.0. Scaled and re-oriented for use in this project.

## Modifications

- **2026-07-06** — Wired up `GLTFLoader` to load an optional whale model from `./models/moby-dock.glb`. Details:
  - Added `three/addons/` to the importmap in `index.html` so the addon resolves at runtime.
  - Added an `import` for `GLTFLoader` and three tuning constants (`WHALE_MODEL_URL`, `WHALE_MODEL_TARGET_LENGTH = 60`, `WHALE_MODEL_ROTATION = { x: PI/2, y: 0, z: 0 }`) at the top of `script.js`.
  - Added `tryLoadWhaleModel(whaleGroup)`: attempts to load the `.glb`, rotates from glTF's Y-up to the game's Z-up, auto-scales the longest bounding-box axis to `WHALE_MODEL_TARGET_LENGTH` world units, centers the model on its own XY and sits it on z=0, enables `castShadow`/`receiveShadow` on all meshes, then clears the primitive-whale children from the group and adds the loaded scene. On any error (404 included) the primitive whale is kept and a friendly `console.info` is logged.
  - Called `tryLoadWhaleModel(whale)` immediately after the primitive whale is added to the scene, so the load happens once at startup and swaps in when ready.
  - Expected input file: `frontend/game/models/moby-dock.glb`. If the file is missing at page load, the primitive whale stays visible with no visual glitch.
  - Attribution for the model is captured in the "3D model" section above.

- **2026-07-06** — Swapped the chicken character for a voxel Docker whale. Details:
  - Renamed identifiers throughout `script.js`: `Chicken` -> `Whale`, `chicken` -> `whale`, `chickenSize` -> `whaleSize`, `chickenMinX`/`chickenMaxX` -> `whaleMinX`/`whaleMaxX`. `whaleSize` (=15) is still just the hit-detection width, not the visual size.
  - Wrote the `Whale()` constructor to build a side-oriented Moby-Dock-style figure out of BoxGeometry primitives, in Docker blue (0x2496ED):
    - Multi-segment body: a wide middle `body` (16x12x10) plus a smaller `head` (8x10x8) at +X and a narrower `peduncle` (5x8x6) at -X. The stepped silhouette reads as a tapered whale shape rather than a rectangle.
    - Two horizontal `fluke` boxes at the back (3x8x2 each) with a gap between them, evoking a whale tail seen from above.
    - A `fin` (4x3x1.5) protruding from the near-camera side (-Y) partway down the body — the pectoral fin.
    - `eye` (2x1x2, dark) on the -Y face of the head. The camera is positioned at roughly (+X, -Y, +Z), so -Y is the camera-facing side.
    - Three shipping containers (red, yellow, green) stacked in a row along the top of the middle body.
  - Visual body width along X is wider than the hit-detection width (32 vs 30 world units — very close). Overall whale footprint is roughly 60x40x28 world units.

- **2026-07-06** — Upgraded three.js from r99 (global UMD build) to r160 via ES-module importmap. Details:
  - `index.html`: replaced the r99 `<script>` tag with an importmap for `three` (unpkg) and changed `script.js` to `type="module"`.
  - `script.js`: added `import * as THREE from 'three';` at the top.
  - `script.js`: renamed `BoxBufferGeometry` -> `BoxGeometry` (11 sites) and `PlaneBufferGeometry` -> `PlaneGeometry` (1 site); the `*BufferGeometry` aliases were removed in r144.
  - `script.js`: added `const` to previously-implicit globals (`hemiLight`, `dirLight`, `backLight`, `height`, and a `positionY` in the `backward` case) so ES-module strict mode does not throw.
  - `script.js`: preserved the original visual look by setting `THREE.ColorManagement.enabled = false`, `renderer.outputColorSpace = LinearSRGBColorSpace`, and `renderer.useLegacyLights = true`. These three toggles restore r99 behaviour under r160 defaults. `useLegacyLights` was removed in r165, so future three.js upgrades will need to retune lighting rather than keep this shim.

## Runtime notes

- `index.html` loads `three.js r160` from `unpkg.com` via an importmap.
- It also loads a FontAwesome kit via `kit.fontawesome.com`. Icons are for the on-screen tap controls. Non-critical if the kit disappears.
