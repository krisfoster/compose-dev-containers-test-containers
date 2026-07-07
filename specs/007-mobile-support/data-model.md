# Data Model: Mobile Support

**Feature**: `007-mobile-support`  
**Date**: 2026-07-07

This feature introduces no new persistent data structures. All changes are to the rendering layer (HTML, CSS, JS) and one Go string constant. This document describes the interface contracts between the new UI components and the existing codebase.

---

## 1. Orientation State Machine

The portrait-mode overlay is driven by a simple two-state machine managed in JavaScript.

```
LANDSCAPE ──(innerHeight > innerWidth)──► PORTRAIT
PORTRAIT  ──(innerWidth >= innerHeight)──► LANDSCAPE
```

**State: LANDSCAPE**
- Portrait overlay: `display: none` (hidden)
- Game canvas: visible and receiving input
- Directional controls: visible and receiving touch input

**State: PORTRAIT**
- Portrait overlay: `display: flex` (full-screen, above canvas)
- Game canvas: visible but pointer-events blocked by overlay
- Directional controls: obscured by overlay

**Trigger**: `window.addEventListener('resize', ...)` and `window.addEventListener('orientationchange', ...)` both call the same `checkOrientation()` function.

---

## 2. DOM Elements

### Existing elements (modified)

| Element ID | Current role | Change |
|------------|-------------|--------|
| `#controls` | Directional button container | Gains `touchstart` listeners on each child button |
| `#forward`, `#left`, `#backward`, `#right` | Arrow buttons | Gain `touchstart` + `preventDefault()` handlers |
| `body` | Root layout | Gains viewport meta and `touch-action: pan-x pan-y` reset |

### New elements

| Element ID | Tag | Role |
|------------|-----|------|
| `#rotate-prompt` | `<div>` | Full-screen portrait-mode overlay; `position: fixed; z-index: 100` |

---

## 3. CSS Custom Property Contract

The fallback height stack is expressed as three sequential declarations. No custom property is introduced; the cascade handles the fallback.

```css
/* Applied to html, body, and the canvas wrapper */
height: 100vh;    /* fallback: all browsers */
height: 100svh;   /* override: iOS Safari 15.4+, Chrome 108+ */
height: 100dvh;   /* override: iOS Safari 16.0+, Chrome 108+ */
```

---

## 4. Home Page QR Poller Contract

The home page QR polling is a self-contained function. Its interface with the rest of the page:

**Input**: None (reads `document.getElementById('qr-img').src` to update it).

**Side effects**:
- Updates `img#qr-img.src` to `/qr.png?t=<epoch-ms>` on each tick.
- On `onerror`: leaves `img#qr-img.src` unchanged (previous successful PNG or placeholder).

**Interval**: 4000ms (`setInterval`).

**Invariant**: The poller MUST NOT be cleared or paused; it runs for the lifetime of the page. No teardown is needed because navigating away from the page destroys the page's execution context.

---

## 5. Touch Event Handler Contract

Each directional button registers one handler:

```
touchstart → e.preventDefault() → dispatchMovement(direction)
```

`dispatchMovement(direction)` calls the same internal function that the existing keyboard `keydown` handler calls for the corresponding arrow key. The keyboard path is not modified.

**Direction mapping**:

| Button ID | Direction | Equivalent key |
|-----------|-----------|---------------|
| `#forward` | `forward` | ArrowUp |
| `#backward` | `backward` | ArrowDown |
| `#left` | `left` | ArrowLeft |
| `#right` | `right` | ArrowRight |

---

## 6. Home Page Layout Contract

The `gettingStartedPageHTML` constant in `app/main.go` is updated. The new structure:

```html
<div class="layout">
  <div class="nav-col">
    <!-- existing nav buttons: Play, Host, Leaderboard -->
  </div>
  <div class="qr-col">
    <img id="qr-img" src="/qr.png" alt="QR code to join" width="280" height="280">
  </div>
</div>
```

The `.layout` container uses `display: grid; grid-template-columns: 1fr 1fr` at `min-width: 601px` and `grid-template-columns: 1fr` at `max-width: 600px`.

No changes to Go handler logic; only the string constant changes.
