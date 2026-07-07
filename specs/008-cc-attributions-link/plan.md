# Implementation Plan: Add CC Attributions and Leaderboard Link

**Branch**: `008-cc-attributions-link` | **Date**: 2026-07-07 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/008-cc-attributions-link/spec.md`

## Summary

Surface the required CC BY 4.0 attribution for the Moby Dock whale model on the game's start screen, alongside a navigation link to the leaderboard. Both additions are confined to `frontend/game/index.html` (markup) and `frontend/game/style.css` (presentation). No backend (Go) changes, no new services, no new routes.

## Technical Context

**Language/Version**: HTML5, CSS3 (game front-end — no version pinning beyond the existing browser targets)

**Primary Dependencies**: None new. three.js r160 and FontAwesome (both existing) are unchanged.

**Storage**: N/A

**Testing**: Browser validation against the compose stack (constitution Principle IV). No new Go tests — no new Go code.

**Target Platform**: Modern browsers, desktop and mobile viewports. The game targets the same browsers as the existing three.js r160 import-map setup.

**Project Type**: Web application — static game front-end served by the Go backend at `/play`.

**Performance Goals**: No performance impact. The change is static HTML and CSS only.

**Constraints**: Attribution must be visible without scrolling on the start screen at all supported viewport sizes. Must not obstruct the name-input field or Play button.

**Scale/Scope**: Two files edited (`index.html`, `style.css`). No new files in the source tree.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I — Demo-First Delivery | PASS | Attribution appears on the start screen, visible on the projected wall. Leaderboard link aids demo navigation. Both changes are instant demo improvements. |
| II — Compose-Orchestrated Reproducibility | PASS | No new services, no compose changes. The game's static files are already volume-mounted via `./frontend/game:/frontend:ro`. |
| III — Testcontainers for Boundary Tests | N/A | No new Go code or service boundaries. No new tests cross any external boundary. |
| IV — Visible-in-the-Browser Definition of Done | GATE | Feature is entirely visual. Done = observed in browser at `/play`, with attribution and leaderboard link confirmed present on the start screen. See `quickstart.md`. |
| V — Vendored-Code Hygiene | PASS | This feature IS the compliance resolution. `ATTRIBUTION.md` is already correct; this plan surfaces the credit in the running app. No new vendored assets introduced. |

All gates clear. No complexity justification required.

## Project Structure

### Documentation (this feature)

```text
specs/008-cc-attributions-link/
├── plan.md           ← This file
├── research.md       ← Phase 0: decisions on CC requirements, URL, UI placement, CSS
├── quickstart.md     ← Phase 1: validation scenarios
└── tasks.md          ← Phase 2 output (created by /speckit-tasks, not this command)
```

No `data-model.md` — feature introduces no new data entities.
No `contracts/` — feature introduces no new API contracts.

### Source Code (repository root)

```text
frontend/game/
├── index.html    ← Edit: add CC attribution + leaderboard link inside #name-prompt
└── style.css     ← Edit: add styling for #cc-attribution and #leaderboard-link-prompt
```

No other files change.

**Structure Decision**: This feature touches only the existing game front-end static files. The Go backend, Redis, and compose configuration are untouched. The two edited files are already served by the `app` service via the volume mount `./frontend/game:/frontend:ro`.

## Implementation Design

### HTML change (`frontend/game/index.html`)

Inside `#name-prompt`, after the closing `</form>` tag, add two new elements:

1. **Leaderboard link** — a paragraph with a link to `/leaderboard`, opening in a new tab:

   ```html
   <p id="leaderboard-link-prompt">
     <a href="/leaderboard" target="_blank" rel="noopener">View the leaderboard</a>
   </p>
   ```

2. **CC attribution** — a paragraph with the full required credit line, with hyperlinks to the source and the licence:

   ```html
   <p id="cc-attribution">
     <a href="https://sketchfab.com/3d-models/moby-dock-docker-whale-b706010291ca46ad8daca2d4aeb79edd"
        target="_blank" rel="noopener">"Moby Dock (Docker whale)" by Maurice Svay</a>,
     licensed under
     <a href="https://creativecommons.org/licenses/by/4.0/" target="_blank" rel="noopener">CC BY 4.0</a>.
     Scaled and re-oriented for use in this project.
   </p>
   ```

Both are children of `#name-prompt` and disappear automatically when the overlay is dismissed on form submit — no JavaScript change needed.

### CSS change (`frontend/game/style.css`)

Add two rules at the end of the file:

```css
#leaderboard-link-prompt {
  font-size: 0.4em;
  margin: 0;
  text-align: center;
}

#leaderboard-link-prompt a {
  color: #7ecbff;
  text-decoration: underline;
}

#cc-attribution {
  font-size: 0.28em;
  margin: 0;
  text-align: center;
  max-width: 320px;
  opacity: 0.75;
  line-height: 1.5;
}

#cc-attribution a {
  color: inherit;
  text-decoration: underline;
}
```

The `#name-prompt` form already uses `flex-direction: column; gap: 20px` — the new `<p>` elements inherit that gap automatically, spacing them below the Play button without additional margin rules.

Font sizes are relative to the body's `font-size: 2em` (`Press Start 2P` font). At `0.4em`, the leaderboard link renders at ~25.6px equivalent; at `0.28em`, the attribution renders at ~17.9px equivalent — both readable but visually subordinate to the form.

## Complexity Tracking

> No constitution violations. This section intentionally blank.
