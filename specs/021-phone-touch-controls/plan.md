# Implementation Plan: Phone Touch Controls for Whale Movement

**Branch**: `021-phone-touch-controls` | **Date**: 2026-07-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/021-phone-touch-controls/spec.md`

## Summary

Add tap-to-move touch controls to the browser game for phone users. A tap on the game canvas is
projected against the whale's current screen position, resolved to one of the four cardinal game
directions (forward / right / backward / left) via dominant-axis normalisation, and then fed
directly into the existing `move()` function. On touch-capable devices the on-screen D-pad buttons
are hidden. No new services, build steps, or third-party assets are needed — all changes are
confined to `frontend/game/script.js`, `frontend/game/style.css`, and
`frontend/game/index.html`.

## Technical Context

**Language/Version**: Vanilla JavaScript (ES2020 modules), no transpilation

**Primary Dependencies**: Three.js r160 (loaded via ES-module importmap from unpkg.com); no new dependencies introduced

**Storage**: N/A — pure frontend feature, no persistence

**Testing**: Manual browser validation (Constitution Principle IV) on a physical phone or Chrome DevTools touch emulation; no Go test suite changes required

**Target Platform**: Mobile browsers (Chrome, Safari) on phones; desktop browsers are unaffected

**Project Type**: Browser game (frontend-only modification)

**Performance Goals**: Touch response must be indistinguishable in latency from the existing button `touchstart` handlers — no new async work, no timers

**Constraints**: No new runtime dependencies, no build step, no CDN additions beyond what the page already loads; `touch-action: none` already set on canvas in CSS

**Scale/Scope**: Single player session; feature touches ~30 lines of JS and ~5 lines of CSS

## Constitution Check

*GATE: Must pass before implementation. Re-checked post-design below.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Demo-First Delivery | PASS | Touch controls directly increase the number of people who can play without presenter intervention — phones are the primary device for booth attendees |
| II. Compose-Orchestrated Reproducibility | PASS | No new services; existing compose stack unchanged |
| III. Testcontainers Over Mocks | PASS | No new service boundary crossings; no Go tests affected |
| IV. Visible-in-Browser Definition of Done | PASS | Feature must be validated on a real phone or DevTools touch emulation before closing |
| V. Vendored-Code Hygiene | PASS | No new third-party assets or libraries; Three.js r160 already vendored/attributed |
| Technology Stack | PASS | Vanilla JS + CSS changes to existing `frontend/game/`; no new runtime, database, or build tool added |

**Post-design re-check**: Constitution Check remains PASS. The implementation adds a `touchstart`
listener and a CSS rule; it does not introduce any stack change, new service, or new dependency.

## Project Structure

### Documentation (this feature)

```text
specs/021-phone-touch-controls/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit-tasks — not yet created)
```

### Source Code (repository root)

```text
frontend/game/
├── index.html           # No changes expected
├── script.js            # ADD: touch detection, touchstart listener, direction resolver
└── style.css            # ADD: @media (pointer: coarse) { #controls { display: none; } }
```

**Structure Decision**: All changes are within the existing `frontend/game/` directory. No new
files are created in the source tree.

## Implementation Approach

### 1. Touch device detection and button hiding (CSS-first)

Add a single CSS media query to `style.css`:

```css
@media (pointer: coarse) {
  #controls {
    display: none;
  }
}
```

This hides the D-pad on any device where the primary pointing device is coarse (phone touchscreen).
Desktop browsers with a mouse are unaffected. No JavaScript needed for this part.

### 2. Touch listener in `script.js`

After the renderer is set up and `renderer.domElement` is appended to the body, add:

```js
if (window.matchMedia('(pointer: coarse)').matches) {
  renderer.domElement.addEventListener('touchstart', (e) => {
    const touch = e.touches[0];
    const tapX = touch.clientX;
    const tapY = touch.clientY;

    // Project whale's world position to screen space
    const ndc = whale.position.clone().project(camera);
    const whaleScreenX = (ndc.x + 1) / 2 * window.innerWidth;
    const whaleScreenY = (-ndc.y + 1) / 2 * window.innerHeight;

    const dx = tapX - whaleScreenX;
    const dy = tapY - whaleScreenY;

    // Dead zone: ignore taps directly on the whale
    if (Math.abs(dx) < 10 && Math.abs(dy) < 10) return;

    // Dominant-axis normalisation to four cardinal directions
    let direction;
    if (Math.abs(dx) > Math.abs(dy)) {
      direction = dx > 0 ? 'right' : 'left';
    } else {
      direction = dy < 0 ? 'forward' : 'backward';
    }

    move(direction);
  });
}
```

This is intentionally inlined in `script.js` alongside the existing keyboard and button listeners
rather than extracted into a helper — the file has no module abstraction layer and this matches
the existing code style.

### 3. No changes to `index.html`

The HTML is already correctly set up: `touch-action: none` on canvas is already in the CSS,
the viewport meta tag is already present, and the `#controls` div already has the required ID.

## Complexity Tracking

No constitution violations — complexity tracking section not required.
