# Feature Specification: Phone Touch Controls for Whale Movement

**Feature Branch**: `021-phone-touch-controls`

**Created**: 2026-07-10

**Status**: Draft

**Input**: User description: "When the game is played on phones I want users to be able to use the touch interface of the phone to move the whale game character. Tapping to the side for example rotates the whale to face the direction of the tap and then moves it one jump in that direction. Tapping in front moves the whale one tap in front. The movement directions for the whale remain as they are and taps need to be normalised to the four cardinal directions of the game screen: forward, right, back, left. If played on a phone the on screen buttons for movement are removed / hidden."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Tap to Move in any Direction (Priority: P1)

A player on a phone taps anywhere on the game canvas. The game maps the tap position relative to the whale's current screen position to one of the four cardinal game directions (forward, right, back, left). If the tap is to the side of the whale, the whale rotates to face that direction then jumps one step. If the tap is in front of the whale, the whale jumps one step forward.

**Why this priority**: This is the core touch control mechanic. Without it, phone players cannot navigate the game at all.

**Independent Test**: A phone user can open the game, tap in different directions around the whale, and observe the whale rotating and moving correctly in all four cardinal directions.

**Acceptance Scenarios**:

1. **Given** the game is running on a phone and the whale is visible, **When** a player taps to the right of the whale on screen, **Then** the whale rotates to face right and jumps one step right
2. **Given** the game is running on a phone and the whale is visible, **When** a player taps to the left of the whale on screen, **Then** the whale rotates to face left and jumps one step left
3. **Given** the game is running on a phone and the whale is visible, **When** a player taps above the whale (in front / forward direction), **Then** the whale moves one step forward without rotating
4. **Given** the game is running on a phone and the whale is visible, **When** a player taps below the whale (behind / backward direction), **Then** the whale rotates 180° and jumps one step backward
5. **Given** a tap offset is equidistant between two directions, **When** the tap is processed, **Then** a consistent tie-breaking rule resolves to exactly one of the four cardinal directions (no ambiguity or double-move)

---

### User Story 2 - On-Screen Buttons Hidden on Phone (Priority: P2)

A player opens the game on a phone. The keyboard/on-screen movement buttons that are shown on desktop are automatically hidden so they do not clutter the phone screen or interfere with touch input.

**Why this priority**: Without hiding the buttons, the touch area is visually confusing and may overlap controls. The button-hiding also signals that touch is the intended input mode.

**Independent Test**: Open the game on a mobile device (or narrow viewport matching phone dimensions). Verify the movement button overlay is not visible anywhere on screen.

**Acceptance Scenarios**:

1. **Given** a player opens the game on a phone (touch-capable device), **When** the game loads, **Then** the on-screen movement buttons are not rendered or are fully hidden from view
2. **Given** a player opens the game on a desktop browser, **When** the game loads, **Then** the on-screen movement buttons remain visible and functional as before
3. **Given** a player rotates their phone from portrait to landscape, **When** the orientation change completes, **Then** the movement buttons remain hidden and touch controls remain active

---

### User Story 3 - Tap Normalisation to Four Cardinal Directions (Priority: P3)

Any tap at any angle around the whale is snapped to the closest of the four cardinal screen directions (0°, 90°, 180°, 270°). Diagonal or arbitrary-angle taps do not produce diagonal movement — they round to the nearest cardinal.

**Why this priority**: The whale's movement model is grid-based (forward/right/back/left). Normalisation must be correct so players experience predictable, satisfying control.

**Independent Test**: Tap at 45° angles (diagonals) around the whale and confirm the whale always moves in one of the four cardinal directions, never diagonally.

**Acceptance Scenarios**:

1. **Given** a tap lands at a 45° diagonal from the whale, **When** the direction is normalised, **Then** the tap resolves to one of the two adjacent cardinal directions via a deterministic rule
2. **Given** the whale is at the edge of the playfield, **When** a player taps in the direction of the boundary, **Then** the movement is blocked in the same way as keyboard/button input would be blocked

---

### Edge Cases

- What happens when a player taps directly on the whale itself (zero offset)? The tap should be ignored or treated as a no-op with no movement.
- What happens when the player uses multi-touch (two fingers)? Only the first touch point should be used; subsequent simultaneous touches are ignored.
- What if the game is embedded in a scrollable page and a touch swipe is intended as page scroll? Touch events on the game canvas must be consumed so the page does not scroll during gameplay.
- What happens when the phone screen dimensions change mid-game (e.g. browser toolbar auto-hide)? The touch coordinate system must recalculate so taps remain correctly mapped.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: On touch-capable devices (phones), the game MUST accept tap events on the game canvas as movement input for the whale
- **FR-002**: A tap event MUST be translated to exactly one of four cardinal movement directions: forward, right, back, or left — relative to the game screen orientation (not the whale's facing direction)
- **FR-003**: The direction resolution MUST be based on the vector from the whale's current screen position to the tap position, normalised to the nearest of the four cardinal axes (90° sectors)
- **FR-004**: When a resolved tap direction differs from the whale's current facing direction, the whale MUST rotate to face that direction before moving
- **FR-005**: After direction resolution and any required rotation, the whale MUST move exactly one jump/step in the resolved direction — identical in distance and animation to a keyboard/button-triggered move
- **FR-006**: On-screen movement buttons MUST be hidden when the game is detected as running on a touch-capable phone; they MUST remain visible on non-touch (desktop) sessions
- **FR-007**: A tap directly on the whale with no discernible directional offset MUST be treated as a no-op (no movement, no rotation)
- **FR-008**: Only the first touch point in a multi-touch gesture MUST be used; additional simultaneous touches MUST be ignored
- **FR-009**: Touch events on the game canvas MUST be consumed so they do not trigger browser default behaviours (page scroll, zoom) during gameplay
- **FR-010**: The existing whale movement rules (grid boundaries, obstacle collisions, valid move checks) MUST apply identically to touch-triggered moves as to keyboard/button-triggered moves

### Key Entities

- **Tap Event**: A single touchstart or touchend event registered on the game canvas, characterised by its screen coordinates
- **Whale Screen Position**: The rendered 2D position of the whale's centre on the game canvas at the moment of the tap
- **Cardinal Direction**: One of four discrete movement directions — forward, right, back, left — as defined by the existing game movement model
- **Touch Zone**: The full game canvas area that is active for touch input

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A phone player can navigate the whale to any reachable grid position using only tap gestures, without ever touching the keyboard or on-screen buttons
- **SC-002**: 100% of taps in each of the four cardinal half-planes (90° sector each) resolve to the correct cardinal direction
- **SC-003**: The whale responds to a tap within the same frame budget as a keyboard-triggered move (no perceptible extra latency introduced by touch handling)
- **SC-004**: On-screen movement buttons are absent on all tested phone form factors in both portrait and landscape orientation
- **SC-005**: Zero browser scroll or zoom events are triggered by tapping the game canvas during a play session on a phone

## Assumptions

- The game canvas occupies a fixed or full-screen region on the phone display; tap coordinates are measured relative to the canvas element, not the full page
- "Phone" detection is based on the presence of touch input capability (e.g. `window.matchMedia('(pointer: coarse)')` or equivalent), not user-agent string parsing
- The whale's current position is always representable as a 2D screen coordinate at the time of any tap, even during animations (mid-animation taps may queue or be ignored — consistent with existing keyboard behaviour)
- The four cardinal directions map directly to the existing movement directions already implemented in the game; no new movement directions are introduced
- The on-screen movement buttons are a single DOM element or clearly identifiable group that can be shown/hidden via CSS without restructuring the layout
- Portrait and landscape orientations are both supported; the cardinal direction mapping remains relative to the screen (not the device orientation sensor)
- This feature targets phones; tablets with touch screens are out of scope and may receive touch controls or buttons depending on how the detection threshold is set — this boundary is not critical for this feature
