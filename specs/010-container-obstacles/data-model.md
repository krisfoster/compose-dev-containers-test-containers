# Data Model: Container-Themed Obstacles

This feature is a visual-layer change with no new backend data or persistent state. The "data model" here describes the in-memory game objects and file-system assets involved.

---

## Vendored Asset Records

### ContainerShipAsset

| Field       | Value |
|-------------|-------|
| File path   | `frontend/game/models/container-ship.glb` |
| Title       | "Container Ship" |
| Author      | RM02 |
| Source URL  | https://skfb.ly/prrCt |
| Licence     | CC BY 4.0 |
| Format      | glTF 2.0 binary |
| Modifications | Rotated +90° around X axis (glTF Y-up → game Z-up); longest axis scaled to 200 world units; centered on XY; bottom placed at z=0 |

### ContainerAsset

| Field       | Value |
|-------------|-------|
| File path   | `frontend/game/models/container.glb` |
| Title       | "Container" |
| Author      | Willy Decarpentrie |
| Source URL  | https://skfb.ly/FZOL |
| Licence     | CC BY 4.0 |
| Format      | glTF 2.0 binary |
| Modifications | Rotated +90° around X axis; longest axis scaled to 100 world units; centered on XY; bottom at z=0 |

---

## In-Memory State (script.js globals)

### containerShipScene
- **Type**: `THREE.Group | null`
- **Initial value**: `null`
- **Set by**: `preloadVehicleModels()` — assigned when `GLTFLoader` resolves for `container-ship.glb`
- **Read by**: `Truck()` — if non-null, `.clone(true)` is used as the truck mesh

### containerScene
- **Type**: `THREE.Group | null`
- **Initial value**: `null`
- **Set by**: `preloadVehicleModels()` — assigned when `GLTFLoader` resolves for `container.glb`
- **Read by**: `Truck()` — if non-null, `.clone(true)` may be used as an alternate truck mesh

---

## Modified Game Object: Truck Group

The `Truck()` constructor currently returns a `THREE.Group` built from `BoxGeometry` primitives. After this feature:

1. **If both models are pre-loaded**: returns a `THREE.Group` containing a randomly-selected clone of `containerShipScene` or `containerScene`, scaled and oriented per research decisions.
2. **If models are not yet loaded** (race at startup): returns the existing voxel group (unchanged).
3. **If either model file is missing / load error**: that model is skipped; the other may still be used. If both fail, the voxel group is used.

The hit-detection bounding box (used in the `animate()` loop) is unchanged: `vechicleLength.truck = 105` world units (pre-zoom).

---

## HTML Attribution Record

### `#cc-attribution` paragraph (index.html)

Extended from one entry (Moby Dock) to three entries:

| # | Title | Author | Source URL | Licence |
|---|-------|--------|-----------|---------|
| 1 | Moby Dock (Docker whale) | Maurice Svay | https://sketchfab.com/3d-models/moby-dock-docker-whale-b706010291ca46ad8daca2d4aeb79edd | CC BY 4.0 |
| 2 | Container Ship | RM02 | https://skfb.ly/prrCt | CC BY 4.0 |
| 3 | Container | Willy Decarpentrie | https://skfb.ly/FZOL | CC BY 4.0 |

Each entry renders as a hyperlinked title + author, followed by a CC BY 4.0 link.

---

## Unchanged Entities

| Entity | Status |
|--------|--------|
| `Car()` voxel constructor | Unchanged |
| `Whale()` / `tryLoadWhaleModel()` | Unchanged |
| Lane types (`car`, `truck`, `forest`, `field`) | Unchanged |
| Hit-detection logic (`vechicleLength`) | Unchanged (values preserved) |
| Redis / Go backend / leaderboard | No changes |
