# Research: Phone Touch Controls for Whale Movement

## How to project the whale's 3D position to 2D screen coordinates

**Decision**: Use THREE.js's built-in `Vector3.project(camera)` method, which converts a world-space position into NDC (Normalised Device Coordinates) for any camera type including `OrthographicCamera`. Convert NDC to screen pixels with `screenX = (ndc.x + 1) / 2 * innerWidth` and `screenY = (-ndc.y + 1) / 2 * innerHeight`.

**Rationale**: This is the canonical Three.js approach and works correctly for the existing `OrthographicCamera`. The whale `THREE.Group` tracks its world position in `whale.position`, which is always current at the moment of a touch event (even mid-animation — the position is the rendered position at that frame).

**Alternatives considered**: Using the whale's DOM bounding rect (impossible — it is a WebGL object with no DOM element), or using a raycaster in reverse (overly complex; raycasters cast rays *from* the screen *into* the scene — the reverse projection is the simpler `project()` call above).

---

## How to detect "phone / coarse pointer" without user-agent sniffing

**Decision**: Use the CSS Media Queries Level 4 pointer interaction feature: `window.matchMedia('(pointer: coarse)').matches`. This returns `true` on touchscreen phones and tablets, and `false` on desktop mice/trackpads.

**Rationale**: User-agent sniffing is fragile and widely considered an anti-pattern. `pointer: coarse` is the modern, specification-defined way to detect "a pointing device with limited accuracy" — exactly a finger on a touchscreen. It is supported in all modern mobile browsers (Chrome, Safari, Firefox, Samsung Internet) and is already partially addressed in the codebase (`manifest.json` targets mobile). The detection runs once at game load, matching the single-session nature of the game.

**Alternatives considered**:
- `'ontouchstart' in window` — works on most phones but also matches laptops with touchscreens that have a precise trackpad. `pointer: coarse` is more semantically correct.
- User-agent string — brittle, requires maintenance.
- Always showing tap controls (in addition to buttons) — clutters the desktop experience and can cause conflicting inputs.

---

## How to map a 2D tap vector to one of four cardinal game directions

**Decision**: Compute the 2D screen-space vector from the whale's projected screen position to the tap coordinates (dx, dy). Resolve to a cardinal direction by taking the **dominant axis**: if `|dx| > |dy|`, the direction is left or right; otherwise it is forward or backward. Sign determines which: positive dx → right, negative dx → left, negative dy → forward (screen Y increases downward, so tapping above the whale gives negative dy), positive dy → backward.

**Rationale**: The 45° sector boundary (dominant-axis test) is the simplest tie-breaking rule that feels natural in use — a tap at 30° off horizontal still resolves to the horizontal axis, which is what a user pressing "to the side" would expect. It also matches the convention used by most grid-based mobile games.

**Alternatives considered**:
- Angle-based `Math.atan2` with 45° bucket boundaries — equivalent result but more computationally complex; the dominant-axis test is simpler to read and produces the same 4-sector split.
- Mapping screen directions through the camera matrix to world axes — technically more precise for the angled camera but practically unnecessary: the camera yaw (20°) and roll (10°) are small enough that screen-space quadrants correspond naturally to the game's forward/right/back/left directions in the existing layout.

---

## Whether to block mid-animation taps (queue or drop)

**Decision**: Pass taps directly to the existing `move()` function. The existing `moves[]` queue already handles queued inputs, so mid-animation taps will be queued rather than dropped, consistent with keyboard and button behaviour.

**Rationale**: The `move()` function checks `finalPositions` against pending queued moves before deciding whether to accept the new input, so it is already safe to call with a mid-animation touch. No extra queuing or debounce logic is needed.

**Alternatives considered**: Debouncing or dropping taps during animation — would make the game feel unresponsive on mobile compared to the keyboard experience; inconsistent with documented FR-010.

---

## Whether `canvas { touch-action: none; }` is already set

**Decision**: Already in place. `style.css` already has `canvas { touch-action: none; }`, which prevents browser scroll and zoom on canvas touch events.

**Rationale**: No additional CSS or `e.preventDefault()` calls needed to suppress browser default touch behaviours on the canvas. The `touchstart` listener on the canvas will not trigger scroll even without `e.preventDefault()`.

**Alternatives considered**: Adding `e.preventDefault()` in the JS listener as belt-and-suspenders — acceptable but redundant given the CSS declaration; the CSS approach is cleaner and avoids the "passive listener" warning browsers emit when `preventDefault()` is called without `{ passive: false }`.

---

## Where to attach the touch listener

**Decision**: Attach `touchstart` on `renderer.domElement` (the WebGL canvas), not on `document.body` or `window`.

**Rationale**: The canvas fills the full viewport (`renderer.setSize(window.innerWidth, window.innerHeight)`), so all touches in the play area hit it. Attaching to the body would also capture touches on the `#controls` div (which will be hidden on phones anyway) and on the game-over overlay, requiring extra filtering. Scoping to the canvas is narrower and cleaner.

**Alternatives considered**: `window` or `document` — too broad; `#game-canvas` ID — the canvas has no ID in the current HTML, using `renderer.domElement` is the correct reference.

---

## Zero-offset tap (tapping directly on the whale)

**Decision**: Treat a tap where both `|dx|` and `|dy|` are below a minimum threshold (e.g., 10 CSS pixels) as a no-op — do not call `move()`.

**Rationale**: FR-007 requires this behaviour. A 10px dead zone corresponds to roughly one finger-width precision margin and is small enough not to affect usability.

**Alternatives considered**: No dead zone (treat dx=0 as left because sign is not defined for zero) — non-deterministic and would frustrate players. Larger dead zone (50px) — would require precise central tapping for no reason.
