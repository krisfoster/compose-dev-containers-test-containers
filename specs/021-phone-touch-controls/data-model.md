# Data Model: Phone Touch Controls for Whale Movement

This feature introduces no new persistent data and no new server-side entities. All logic is
ephemeral, computed at runtime within a single browser session.

## Runtime Entities

### TapEvent (transient)

Represents a single finger-down event on the game canvas.

| Field | Source | Description |
|-------|--------|-------------|
| `clientX` | `TouchEvent.touches[0].clientX` | Horizontal pixel position of the tap, relative to the viewport |
| `clientY` | `TouchEvent.touches[0].clientY` | Vertical pixel position of the tap, relative to the viewport |

Only the first touch point (`touches[0]`) is used. Subsequent simultaneous touches are ignored
(FR-008).

---

### WhaleScreenPosition (computed)

The projected 2D screen position of the whale at the moment of a tap. Derived from the whale's
current 3D world position and the active camera.

| Field | How computed | Description |
|-------|-------------|-------------|
| `screenX` | `(ndc.x + 1) / 2 * innerWidth` | Whale centre, x-axis, in CSS pixels |
| `screenY` | `(-ndc.y + 1) / 2 * innerHeight` | Whale centre, y-axis, in CSS pixels (y=0 is top) |

NDC is obtained via `whale.position.clone().project(camera)`.

---

### TapVector (computed)

The 2D vector from the whale's screen position to the tap position.

| Field | Formula | Sign convention |
|-------|---------|----------------|
| `dx` | `tap.clientX − whale.screenX` | Positive = tap is to the right of the whale |
| `dy` | `tap.clientY − whale.screenY` | Positive = tap is below the whale (screen Y increases downward) |

---

### CardinalDirection (resolved)

One of four string values matching the existing `move()` API:

| Value | Tap condition |
|-------|--------------|
| `'forward'` | `\|dy\| ≥ \|dx\|` and `dy < 0` (tap is above whale on screen) |
| `'backward'` | `\|dy\| ≥ \|dx\|` and `dy > 0` (tap is below whale on screen) |
| `'left'` | `\|dx\| > \|dy\|` and `dx < 0` (tap is left of whale on screen) |
| `'right'` | `\|dx\| > \|dy\|` and `dx > 0` (tap is right of whale on screen) |
| _(no-op)_ | `\|dx\| < 10` and `\|dy\| < 10` (tap is within the dead zone centred on the whale) |

The dead zone radius is 10 CSS pixels. This entity resolves to no-op rather than a direction when
the tap is within the dead zone (FR-007).

---

## Existing Entities Modified

### `move(direction: string)` (existing function in `script.js`)

No changes to this function. The touch handler calls `move()` with a `CardinalDirection` value,
which is already one of the four strings the function accepts. All existing validity checks
(boundary detection, forest obstacle checks, `nameEntered` guard) continue to apply unmodified.

---

## Touch UI State

### Device Type (computed once at page load)

| Field | Value | Description |
|-------|-------|-------------|
| `isTouchDevice` | `window.matchMedia('(pointer: coarse)').matches` | True on phones; drives controls visibility |

When `isTouchDevice` is `true`:
- The `#controls` element is hidden (CSS or JS)
- The canvas `touchstart` listener is registered

When `isTouchDevice` is `false`:
- No changes to existing behaviour

No state is persisted between page loads; the detection runs fresh on each load.
