# Feature Specification: Mobile Support

**Feature Branch**: `007-mobile-support`

**Created**: 2026-07-07

**Status**: Draft

**Input**: Issue #1 from issues.md: "Make the game work better on phones: tap-to-move controls for the whale, and full-screen landscape play (prompt the user to rotate their phone when in portrait mode). Also, update the home page so that the QR code is displayed on the right hand side (2-col layout) and ensure that it refreshes identically to how it does on the QR code page."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Tap-to-Move Controls on Mobile (Priority: P1)

A booth attendee scans the QR code on their phone and opens the game. Directional buttons are visible on the game screen and respond to taps, moving the whale forward, backward, left, and right without requiring a keyboard.

**Why this priority**: This is the core playability requirement. Without responsive tap controls, mobile users cannot play the game at all. All other mobile improvements depend on the game being playable first.

**Independent Test**: Open the game on a physical or emulated phone, tap each directional button, and confirm the whale moves in the corresponding direction.

**Acceptance Scenarios**:

1. **Given** the game is loaded on a mobile browser in landscape orientation, **When** the player taps the forward/backward/left/right buttons, **Then** the whale moves one step in the tapped direction with the same responsiveness as keyboard arrow keys.
2. **Given** a player is mid-game on mobile, **When** they tap a directional button rapidly, **Then** each tap registers as a discrete movement (no double-fire, no missed taps from touch event aliasing).
3. **Given** the game is loaded on mobile, **When** the name prompt is visible, **Then** the directional control buttons are not visible or interactive — they only appear once the game starts.

---

### User Story 2 - Portrait-Mode Rotation Prompt (Priority: P2)

A player opens the game on their phone in portrait orientation. Instead of seeing a broken or tiny game, they see a clear prompt asking them to rotate their device to landscape. When they rotate, the prompt disappears and the game fills the screen.

**Why this priority**: Without a rotation prompt, players in portrait mode see a broken game experience. The prompt eliminates confusion and guides the player to a good play state.

**Independent Test**: Open the game on a phone in portrait orientation and confirm the rotation prompt appears. Rotate the phone and confirm the game is now visible and playable.

**Acceptance Scenarios**:

1. **Given** a phone browser in portrait orientation, **When** the game page loads, **Then** a full-screen overlay appears instructing the player to rotate their device; the three.js canvas and directional controls are not interactable beneath it.
2. **Given** the rotation prompt is showing, **When** the user rotates the device to landscape, **Then** the prompt disappears and the game canvas fills the viewport.
3. **Given** the player is mid-game in landscape, **When** they rotate back to portrait, **Then** the rotation prompt reappears immediately (game does not crash or glitch beneath the overlay).

---

### User Story 3 - Full-Screen Landscape Play (Priority: P2)

A player on a phone in landscape orientation has the game canvas and controls fill the entire viewport with no white space, address bar overflow, or scroll bars. The experience uses the full screen.

**Why this priority**: Even with functional tap controls, a game that does not properly fill the screen looks unpolished and is harder to play. Full-screen layout is essential for the demo-quality bar.

**Independent Test**: Load the game in landscape on a phone; confirm no scroll bars, no white edges, and the three.js canvas fills the full viewport.

**Acceptance Scenarios**:

1. **Given** the game is loaded in landscape on a mobile browser, **When** the player looks at the screen, **Then** the game canvas fills the full viewport width and height with no scroll bars or address-bar overflow.
2. **Given** the full-screen landscape layout is active, **When** the directional control buttons are displayed, **Then** they are positioned and sized so they are easy to tap without obscuring the entire play area.
3. **Given** a phone whose browser shows and hides an address bar dynamically, **When** the player is in the game, **Then** the game layout fills the visible viewport area and does not overflow beneath the browser chrome bar.

---

### User Story 4 - Home Page QR Code in Two-Column Layout (Priority: P3)

The landing page at `/` is redesigned with a two-column layout: navigation links on the left, the QR code image on the right. The QR code auto-refreshes on the same polling interval used on the `/host` page, so the home page always shows the current code without a manual reload.

**Why this priority**: The home page QR is a secondary presenter convenience — important for the booth experience but the game being playable is the critical path. P3 reflects that it does not gate mobile play.

**Independent Test**: Open `/` on the presenter's laptop and confirm a 2-column layout with the QR image on the right; wait for the QR window to be rotated from `/host` and confirm the home page image updates automatically.

**Acceptance Scenarios**:

1. **Given** an active QR window exists, **When** a presenter opens `/`, **Then** the page displays a two-column layout: navigation links (Play / Host / Leaderboard) on the left, and the current QR code image on the right.
2. **Given** the home page is open, **When** the QR code is rotated via `/host/rotate`, **Then** within the next polling interval the home page QR image updates to show the new code without a page reload.
3. **Given** no active QR window exists yet (first load before `/host` has been opened), **When** the presenter opens `/`, **Then** the QR column shows a placeholder or empty state rather than a broken image; no JavaScript error is thrown.
4. **Given** the home page is open on a narrow screen (e.g. tablet portrait), **When** the viewport width is below a responsive breakpoint, **Then** the layout collapses to a single column so neither the links nor the QR code are cut off.

---

### Edge Cases

- What happens if the player's phone does not fire `orientationchange` events (some older Android browsers use `resize` instead)? Both events should be listened to so the portrait prompt appears and disappears reliably.
- What if the player has a foldable phone that changes orientation by unfolding rather than rotating? The portrait prompt should respond to viewport dimensions (height > width), not solely the `orientation` API.
- Does the rotation prompt interfere with the name-entry form if it appears while the prompt is showing?
- What if `/qr.png` returns 503 (no active window) when the home page polls it? The image should not flicker to a broken state; the last successful image should remain displayed until a new one loads.
- Does the home page QR polling keep working when the presenter's browser is left open for a full conference day (memory leak via `setInterval`)?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The game MUST respond to tap (touch) events on the four directional control buttons and move the whale by one step per tap, identically to how a keyboard arrow-key press does.
- **FR-002**: The game page MUST scale correctly on mobile device screens without the user needing to manually zoom or scroll horizontally.
- **FR-003**: When the device is held in portrait orientation, the game page MUST display a full-screen overlay prompting the user to rotate their device to landscape. The game controls and play area MUST remain inaccessible while the prompt is shown.
- **FR-004**: When the device is in landscape orientation, the portrait-mode overlay MUST be hidden and the game canvas MUST fill the full visible viewport.
- **FR-005**: The game layout MUST fill the visible viewport height on mobile, including on browsers that show and hide an address bar dynamically, without overflowing or leaving white gaps.
- **FR-006**: The directional control buttons MUST remain visible and usable in landscape mode and MUST NOT be obscured by the game canvas on any modern smartphone screen in landscape.
- **FR-007**: The landing page at `/` MUST be updated to a two-column layout in which navigation links appear on the left and the QR code image appears on the right at medium-and-wider screen widths.
- **FR-008**: The home page QR image MUST automatically refresh at a fixed interval and update when the active QR code changes, matching the refresh behaviour of the dedicated host display page.
- **FR-009**: If no active QR code is available (e.g. the presenter has not yet opened the host page), the home page MUST show a placeholder rather than a broken image, and MUST display the QR image as soon as one becomes available without a manual reload.
- **FR-010**: On narrow viewports (e.g. a phone held in portrait), the home page two-column layout MUST collapse to a single column so content is not clipped or overflowed.
- **FR-011**: Tapping a directional button MUST produce exactly one movement — no duplicate movements from a single tap, regardless of the mobile browser's touch-to-click event sequence.

### Key Entities

- **Portrait Rotation Prompt**: A full-screen overlay that appears when the device is held in portrait orientation, instructing the player to rotate; disappears automatically when the device is in landscape.
- **Mobile Tap Controls**: The four directional buttons on the game screen that respond to finger taps, producing the same movement as keyboard arrow keys.
- **Home Page QR Display**: The live QR code image on the landing page that auto-refreshes to show the current scannable code without a manual reload.
- **Responsive Home Layout**: The two-column layout of the landing page that places navigation links on the left and the QR code on the right, collapsing to a single column on narrow screens.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A player on a modern smartphone in landscape orientation can complete a full game (name entry → play → game over → replay) using only tap controls, with no keyboard required.
- **SC-002**: The portrait-mode rotation prompt appears immediately when the device is rotated to portrait and disappears immediately when rotated back to landscape, with no visible lag.
- **SC-003**: The game play area fills 100% of the visible viewport in landscape with no white gaps, scroll bars, or content cut off on a representative modern smartphone.
- **SC-004**: The home page QR code image updates automatically within 5 seconds of the QR code being rotated on the host page, without a manual page reload.
- **SC-005**: A player completes the full mobile game flow with no error messages or broken states visible in the browser.

## Assumptions

- The four directional control buttons already exist in the game's HTML; this feature adds touch responsiveness to them without removing or changing existing keyboard controls.
- The target device is a modern smartphone (iOS 15+, Android 10+) in landscape orientation. Tablet support is a beneficial side effect of good responsive design but is not an explicit validation target.
- The home page is a self-contained page served by the existing Go backend; no new backend services or routes are required.
- The QR code refresh behaviour on the home page matches the mechanism already used on the dedicated host display page — no new backend endpoints are needed.
- No new third-party libraries or build tools are introduced; all changes use capabilities already present in the project.
- Portrait orientation is detected by comparing viewport width and height — this approach works correctly on foldable phones and browsers that do not fire orientation-change events.
