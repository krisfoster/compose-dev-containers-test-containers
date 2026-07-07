# Research: Container-Themed Obstacles

## Model Format & Loading

**Decision**: Load both new models via the existing `GLTFLoader` pattern already used for `moby-dock.glb`.

**Rationale**: Sketchfab downloads produce standard glTF 2.0 binary (`.glb`) files. The `GLTFLoader` from `three/addons/` is already imported in `script.js`. No new loader or importmap entry is needed.

**Alternatives considered**: Three.js `OBJLoader`, `FBXLoader` — rejected because Sketchfab's preferred export is GLB and the project already has GLTFLoader wired.

---

## Y-up → Z-up Rotation Convention

**Decision**: Apply `{ x: Math.PI / 2, y: 0, z: 0 }` to every new loaded model, identical to the whale.

**Rationale**: glTF spec uses Y-up. The game uses Z-up (ground is the XY plane; vehicles move along X; camera looks down from Z). The same rotation has already been validated in production for `moby-dock.glb`.

**Alternatives considered**: Rotating at export time — rejected because the GLB files are vendored unmodified per constitution Principle V.

---

## Model-to-Lane Assignment

**Decision**: Both models appear in truck lanes. Each truck-lane vehicle randomly chooses container ship (50%) or container (50%). No change to car lanes.

**Rationale**: The user specified "replace the trucks". Using both models adds visual variety without changing lane logic. Car lanes retain existing voxel Car() since no car-themed model was provided.

**Alternatives considered**:
- Container ship for trucks, container for cars — rejected because no container model was downloaded for car lanes, and car lanes are not in scope.
- Only container ship, ignore container model — rejected because the user provided both models and expects both to appear.

---

## Pre-loading Strategy (Shared Model + Clone)

**Decision**: Pre-load each model once at startup into a module-level variable (`containerShipScene`, `containerScene`). When a new truck group is created, clone the cached scene (using three.js `scene.clone(deep=true)`) and add it to the truck group. If the cache variable is `null` at truck-creation time, the voxel fallback is used for that vehicle.

**Rationale**: Creating one `GLTFLoader` per vehicle (which can be 2–4 per lane, dozens over a session) wastes network bandwidth and memory. Cloning a pre-loaded scene is the standard three.js pattern for instancing complex models.

**Alternatives considered**:
- One load per vehicle group (whale pattern): inappropriate for repeated instantiation; the whale is created once.
- `InstancedMesh`: powerful but requires all instances to share the same geometry/material, which is not guaranteed for multi-mesh GLB files. Overkill for this game's obstacle count.

---

## Target Lengths (Scaling)

**Decision**:
- `CONTAINER_SHIP_MODEL_TARGET_LENGTH = 200` world units (the longest bounding-box axis of the loaded scene is scaled to this value)
- `CONTAINER_MODEL_TARGET_LENGTH = 100` world units

**Rationale**: The existing truck voxel occupies roughly `100 × zoom = 200` world units along X (base is `100×zoom = 200`, cabin is offset). Hit-detection uses `105 × zoom = 210` for trucks. Scaling the container ship to 200 world units makes it fill the lane visually while keeping the existing 210 unit hit-detection box (which is deliberately generous for fair gameplay). The container is a single box — 100 world units (roughly half the truck) gives it a distinct, smaller silhouette that reads as a parked container rather than a moving truck, adding visual variety.

**Alternatives considered**: Using 105 × zoom = 210 world units to match the hit box exactly — rejected because leaving a slight margin between the visual model and the hit-detection box is the existing convention (the whale model is smaller than its hit box too).

---

## Hit-Detection Box

**Decision**: Retain `{ car: 60, truck: 105 }` in the `vechicleLength` lookup. No change.

**Rationale**: Hit detection is an axis-aligned 1D range check along X. The values were tuned for playability. Changing them risks making the game harder or easier for reasons unrelated to the visual swap. The new models are scaled to fit within the existing box.

---

## Model File Naming

**Decision**:
- `frontend/game/models/container-ship.glb` — for "Container Ship" by RM02
- `frontend/game/models/container.glb` — for "Container" by Willy Decarpentrie

**Rationale**: Lowercase kebab-case matches the existing `moby-dock.glb` convention. Names are descriptive and stable.

**Note on source files**: The user downloaded these models to `~/Downloads` on the host machine. The sandbox does not have direct access to the host `~/Downloads` path. Implementation will require the user to copy the files into the repo or confirm the exact filenames from Sketchfab's download so the implementation task can instruct the copy step precisely.

---

## Attribution Display (Start Screen)

**Decision**: Extend the existing `#cc-attribution` paragraph in `index.html` with two new `<a>` link pairs — one for Container Ship (RM02) and one for Container (Willy Decarpentrie). Separate entries with a line break or period delimiter to match the existing Moby Dock entry style.

**Rationale**: Issue 12 (now done) already established `#cc-attribution` as the designated credits surface. Extending it is the least-surprise approach.

**Alternatives considered**: A separate `<details>` element for "more credits" — rejected; the existing paragraph is already the designated surface and adding a third entry is still readable.

---

## ATTRIBUTION.md Update

**Decision**: Add two new sections to `frontend/game/ATTRIBUTION.md` following the existing Moby Dock section format (title, author, source URL, licence, downloaded date, modifications note, required credit line).

**Rationale**: Principle V requires ATTRIBUTION.md to be updated in the same commit that vendors the model files.

---

## Fallback Behaviour

**Decision**: If either model file is missing or fails to load, the corresponding vehicle silently uses the voxel `Truck()` shape. The fallback is logged to `console.info` (same as the whale pattern).

**Rationale**: The spec requires 100% graceful fallback (SC-002). The whale pattern already demonstrates this approach is sufficient.
