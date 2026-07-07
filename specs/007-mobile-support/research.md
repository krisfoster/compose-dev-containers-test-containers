# Research: Mobile Support

**Feature**: `007-mobile-support`  
**Date**: 2026-07-07

---

## Decision 1 — Touch Event Strategy: `touchstart` + `preventDefault` vs `pointerdown`

**Decision**: Use `touchstart` with `e.preventDefault()` on each directional button.

**Rationale**: `pointerdown` is the modern, unified API and handles both mouse and touch. However, the existing directional buttons in `index.html` already receive synthetic `click` events on desktop; using `touchstart` + `preventDefault()` on mobile is the minimal, non-breaking addition. `touchstart` fires before the browser generates a synthetic `mousedown`/`click` from a touch sequence; calling `preventDefault()` on it suppresses those synthetic events, preventing the double-fire that would otherwise move the whale twice per tap. `pointerdown` would require refactoring the existing `click` event handlers or using `pointer-events: none` tricks to avoid conflict — more risk for the same outcome.

**Alternatives considered**:
- **`pointerdown` (unified)**: Cleaner long-term but requires reworking the existing click handler wiring to avoid duplicate fires; out-of-scope risk for this feature.
- **CSS `touch-action: manipulation`**: Disables double-tap zoom and reduces 300ms tap delay on some Android browsers, but does not prevent the synthetic mouse event sequence on iOS Safari. Complementary to `touchstart` + `preventDefault()`, not a replacement.

**Implementation note**: The three.js canvas itself must also set `touch-action: none` in CSS to prevent the browser from intercepting touch events for scrolling or zooming while the user is touching the canvas area.

---

## Decision 2 — Viewport Height Units: `dvh` / `svh` / `vh` Fallback Stack

**Decision**: Use `100dvh` as the primary value, `100svh` as the first fallback, and `100vh` as the final fallback, expressed as a CSS custom property or via `@supports`.

**Rationale**: On mobile browsers, `100vh` is historically equal to the viewport height including the retractable browser chrome (address bar). When the chrome is visible, `100vh` overflows the visible area by the height of the chrome, causing a white strip or scroll. `dvh` (dynamic viewport height) tracks the viewport as the chrome changes — it equals the available height when the chrome is visible and the full height when it retracts. `svh` (small viewport height) is always the chrome-visible height (the smallest stable viewport), which prevents overflow at the cost of some unused space when the chrome retracts. `dvh` is ideal; `svh` is a safe conservative fallback for browsers that have `svh` but not `dvh` (a narrow window); `vh` is the last resort.

Browser support (as of 2026-07): `dvh` — iOS Safari 16.0+, Chrome 108+; `svh` — iOS Safari 15.4+, Chrome 108+; `vh` — universal.

```css
/* Fallback stack */
min-height: 100vh;
min-height: 100svh;
min-height: 100dvh;
```

CSS processes all three lines; the last valid one wins, so `dvh` takes effect on browsers that support it.

**Alternatives considered**:
- **JavaScript resize listener to set `--vh` custom property**: The classic workaround (setting `document.documentElement.style.setProperty('--vh', ...)` on resize). Works everywhere but requires JS, fires on every resize, and is fragile to changes in browser chrome behaviour. The native CSS units are now broadly available enough to make this hack unnecessary.
- **`100svh` only**: Always uses the smallest viewport height. Correct but wastes a few pixels of screen space when the browser chrome retracts. Acceptable but `dvh` is a strictly better choice where available.

---

## Decision 3 — Orientation Detection: `resize` Event vs `orientationchange`

**Decision**: Listen to `window.addEventListener('resize', checkOrientation)` as the primary trigger, with `window.addEventListener('orientationchange', checkOrientation)` as a secondary trigger. Detect portrait by comparing `window.innerWidth < window.innerHeight`.

**Rationale**: `orientationchange` has historically had bugs: on older iOS Safari, it fires before `window.innerWidth` and `window.innerHeight` have updated to post-rotation values, requiring a `setTimeout` workaround. On some Android browsers it does not fire for foldable-phone unfold events where the viewport changes without a physical rotation. The `resize` event fires reliably after any viewport size change (rotation, fold/unfold, browser chrome toggle) and `window.innerWidth`/`innerHeight` are always accurate inside a `resize` handler. Using `innerWidth < innerHeight` as the portrait test is more reliable than `screen.orientation.angle` (which has inconsistencies on Android) and `window.orientation` (deprecated).

The `orientationchange` listener is kept as a belt-and-suspenders addition: on some browsers `resize` fires a beat later than `orientationchange`, so adding both ensures the prompt appears as quickly as possible.

**Alternatives considered**:
- **CSS-only portrait detection via `@media (orientation: portrait)`**: Elegant for static styles but does not handle the JS-side need to pause/resume game input. A combination CSS + JS approach would be needed anyway; it is simpler to drive everything from one JS event listener.
- **`screen.orientation` API**: The `change` event on `screen.orientation` is the modern replacement for `orientationchange`. Broad support as of 2026, but `resize` + `innerWidth/innerHeight` is still more reliable for the portrait-vs-landscape test on edge cases (foldables, split-screen).

---

## Decision 4 — QR Image Polling on Home Page

**Decision**: Use `setInterval` to update `<img>` `src` with a cache-busting `?t=<timestamp>` query parameter, matching the approach already used on the `/host` page when a rotate happens. Polling interval: 4 seconds (matching the leaderboard page's polling interval for consistency).

**Rationale**: The `/host` page already demonstrates that appending `?t=Date.now()` to `/qr.png` causes the browser to fetch a fresh image. Reusing the same pattern keeps the home page consistent with what the presenter already knows works. A 4-second interval matches the leaderboard page (already established as acceptable polling cadence) and is short enough to feel "live" for a presenter.

**Error handling**: Use the `<img>` element's `onerror` event to detect a 503 (no active window). On error, do not update `img.src` back to the failed URL — simply leave the previous image in place. The next interval tick will try again. This avoids the "broken image" icon and the visual flicker of a failed load.

**Alternatives considered**:
- **Server-Sent Events (SSE) push when the QR rotates**: More efficient (no polling) but requires adding an SSE endpoint to the Go server, which is scope creep for this feature. Polling is correct here.
- **WebSocket**: Same concern as SSE — scope creep.
- **Longer interval (10s+)**: Would miss a QR rotation in booth conditions where the presenter rotates quickly. 4s is a reasonable balance.

---

## Decision 5 — Home Page Two-Column Layout Technology

**Decision**: CSS Grid with `grid-template-columns: 1fr 1fr` at medium+ widths, collapsing to a single column via `@media (max-width: 600px)`.

**Rationale**: CSS Grid is the right tool for a two-column page layout. The breakpoint at 600px ensures narrow phone screens (including portrait mode visits) collapse to a single readable column. The home page is a Go string constant (`gettingStartedPageHTML` in `main.go`), so the full CSS is inlined in the `<style>` block — no external stylesheet to manage.

**Alternatives considered**:
- **Flexbox**: Also appropriate; Grid is a marginally cleaner fit for a defined two-column layout because it controls both axes simultaneously.
- **CSS columns**: Appropriate for flowing text, not for the discrete left/right content split needed here.

---

## Summary of Resolved Unknowns

| Unknown | Resolution |
|---------|------------|
| Touch event API for button taps | `touchstart` + `preventDefault()` on each directional button |
| Viewport height on mobile | `dvh`/`svh`/`vh` fallback stack in CSS |
| Orientation detection | `resize` event + `innerWidth < innerHeight` comparison |
| Home page QR polling | `setInterval` at 4s, `?t=` cache-bust, `onerror` guard |
| Home page layout | CSS Grid 2-col, collapses at 600px breakpoint |
