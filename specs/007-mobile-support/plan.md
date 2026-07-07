# Implementation Plan: Mobile Support

**Branch**: `007-mobile-support` | **Date**: 2026-07-07 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/007-mobile-support/spec.md`

## Summary

Add mobile playability to Whale Runner: tap-to-move touch controls on the existing directional buttons, a full-screen landscape layout using `dvh`/`svh`/`vh` CSS fallback stack, and a portrait-mode overlay that prompts phone users to rotate before they can play. Update the home page landing screen (`/`) to a two-column layout with the QR code on the right and an auto-refreshing QR image (4-second `setInterval` polling `/qr.png`, matching the leaderboard page's cadence). No new Go packages or npm dependencies are introduced; all changes touch three frontend files (`index.html`, `style.css`, `script.js`) and one Go string constant (`gettingStartedPageHTML` in `main.go`).

## Technical Context

**Language/Version**: Go 1.25 (for `main.go` constant update); JavaScript ES modules, vanilla DOM APIs (no new frameworks or build steps)

**Primary Dependencies**:
- three.js r160 via ES-module importmap (already present — unchanged)
- No new runtime dependencies

**Storage**: N/A — no new persistent state. The QR poller reads `/qr.png`, which is already served by the existing Go handler.

**Testing**: Visual validation in a browser against the compose stack (Constitution Principle IV). Chrome DevTools mobile emulation covers the orientation and layout scenarios; real-device testing validates touch event handling. No Go unit tests are affected; no new Go test coverage is required (the changed Go code is a string constant, not logic).

**Target Platform**: Mobile browsers — iOS Safari 15.4+ and Android Chrome 108+ — in landscape orientation. Desktop browsers are not regressed (changes are additive or guarded by media queries / feature detection).

**Project Type**: Frontend enhancement (HTML/CSS/JS) + Go string constant update. No new services, routes, or API endpoints.

**Performance Goals**: Touch response within one animation frame (~16ms at 60fps). Portrait prompt appears/disappears within one `resize` event cycle. QR image refresh within 5 seconds of a window rotation.

**Constraints**:
- MUST NOT introduce new Go packages or npm dependencies
- MUST NOT modify existing keyboard arrow-key handling in `script.js`
- MUST NOT break the desktop game experience on any existing route (`/`, `/play`, `/host`, `/leaderboard`)
- `touch-action: none` on the three.js canvas to prevent browser scroll/zoom capture during gameplay
- `touchstart` handlers MUST call `e.preventDefault()` to suppress synthetic mouse events (FR-011)

**Scale/Scope**: Four files changed, zero new files, zero new services. Estimated diff: ~80 lines HTML/CSS/JS, ~40 lines Go string constant.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

**Principle I — Demo-First Delivery**: PASS. Mobile support directly increases the number of booth attendees who can join and play without presenter intervention. Tap-to-move controls and the QR on the home page are immediately visible demo improvements: more people can pick up their phone and play, and the presenter's landing page shows the QR code without switching tabs to `/host`.

**Principle II — Compose-Orchestrated Reproducibility**: PASS. No new compose services are introduced. The existing `docker compose up` path is unchanged. The home page QR polling reads the existing `/qr.png` endpoint; no new routes or handlers are added.

**Principle III — Testcontainers Over Mocks**: PASS. This feature touches no service boundaries. The existing Go leaderboard and gate tests are unchanged. No new Go test coverage is required because the changed Go code is a string constant (`gettingStartedPageHTML`), not testable logic.

**Principle IV — Visible-in-Browser Definition of Done**: PASS. All six quickstart scenarios require a running browser against the compose stack. The definition of done is: a player completes a full game on a real or emulated mobile device using tap controls, the portrait prompt appears and disappears on rotation, and the home page QR auto-updates.

**Principle V — Vendored-Code Hygiene**: PASS. No new third-party code or assets are introduced. three.js is already used via CDN importmap. Any "rotate device" icon in the portrait prompt will be Unicode or inline SVG — no external asset — so no `ATTRIBUTION.md` entry is needed.

**Technology Stack — Constitution Amendment**: NOT REQUIRED. No new runtime dependency is added to the compose stack. `dvh`/`svh` are native CSS units, not a new library. The Go string constant update does not add a package.

*Post-Phase-1 re-check*: All five principles still PASS. The data-model confirms no new entities, no new persistence layer, and no new service contracts beyond the existing `/qr.png` endpoint. The home page QR poller is client-side JS only.

## Project Structure

### Documentation (this feature)

```text
specs/007-mobile-support/
├── plan.md              # This file
├── research.md          # Touch events, dvh units, orientation detection, QR polling, home page layout
├── data-model.md        # DOM element contracts, state machine, touch handler mapping
├── quickstart.md        # Six end-to-end validation scenarios
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
frontend/game/
├── index.html    # Add viewport meta, portrait-mode overlay div (#rotate-prompt)
├── style.css     # dvh/svh/vh fallback stack, portrait overlay styles, #controls
│                 #   visibility, touch-action: none on canvas
└── script.js     # touchstart handlers on #forward/#left/#backward/#right,
                  #   checkOrientation() function, resize + orientationchange listeners

app/
└── main.go       # Update gettingStartedPageHTML constant: 2-col CSS Grid layout,
                  #   QR image element (#qr-img), 4s setInterval QR poller with onerror guard
```

No changes to:
- `docker-compose.yml`
- `app/internal/` (gate, leaderboard, qrcode packages)
- `app/go.mod` / `app/go.sum`
- `frontend/game/models/`

**Structure Decision**: All frontend changes are in-place edits to the three existing files under `frontend/game/`. The Go change is a single string constant in `main.go` — no new file, no new handler. This is the minimal, lowest-risk surface area for the feature.

## Complexity Tracking

> No Constitution Check violations requiring justification.
