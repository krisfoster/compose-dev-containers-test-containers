# Feature Specification: Fix Host QR Rotate Route

**Feature Branch**: `019-fix-host-rotate-route`

**Created**: 2026-07-09

**Status**: Draft

**Input**: User description: "create a specification for issue 1 in arch-issues.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Manual QR Code Rotation (Priority: P1)

The booth presenter is on stage showing the live demo. They notice a stray URL has leaked into
chat or that a group's QR window is about to expire. They click the "Refresh QR" button on the
leaderboard screen to immediately invalidate the current join window and display a fresh one.

**Why this priority**: The "Refresh QR" button is already visible on the leaderboard page and
already wired up to send a request — it is already part of the presenter workflow. The bug means
the button does nothing, silently. Fixing this is the highest-value correctness change because it
restores a feature the presenter expects to exist.

**Independent Test**: Can be fully tested by loading the leaderboard page in a browser, clicking
"Refresh QR," and observing that the QR code image updates to a new value. Delivers demonstrable
control over the active join window.

**Acceptance Scenarios**:

1. **Given** the leaderboard is visible with an active QR code, **When** the presenter clicks
   "Refresh QR," **Then** the page sends a rotation request, the request succeeds, and the QR code
   image updates to reflect the new join window.
2. **Given** the leaderboard is visible with an active QR code, **When** the presenter clicks
   "Refresh QR," **Then** the previous join window is invalidated (participants joining via the old
   QR code are no longer accepted after rotation).
3. **Given** the rotation backend is unavailable, **When** the presenter clicks "Refresh QR,"
   **Then** the button does not crash the page, the QR image is not updated, and the presenter can
   retry.

---

### User Story 2 - Automatic QR Rotation (Priority: P2)

The demo runs unattended on a projected screen. Every 60 seconds the join window automatically
refreshes so that a fresh QR code is displayed, limiting how long any single code remains valid.

**Why this priority**: Auto-rotation is already wired in the leaderboard's JavaScript timer. The
same missing route that breaks manual rotation also breaks the timer. Fixing the route repairs
both flows simultaneously.

**Independent Test**: Can be fully tested by watching the leaderboard page for 60 seconds and
confirming the QR code image changes without any manual action. Delivers the intended automatic
window cycling.

**Acceptance Scenarios**:

1. **Given** the leaderboard has been open for 60 seconds, **When** the auto-rotate timer fires,
   **Then** the rotation request succeeds and the QR code image updates automatically.
2. **Given** the page has been open through multiple 60-second cycles, **When** each timer fires,
   **Then** each rotation produces a distinct new QR code.

---

### Edge Cases

- What happens when the rotation request is sent while a previous rotation is still in flight?
  (Duplicate simultaneous rotations should not corrupt state — last-write-wins or idempotent
  behaviour is acceptable.)
- What happens if the join store is unavailable at rotation time? (The QR image must not update;
  the current window remains active.)
- What happens when a player scans a QR code that was valid before a rotation? (Post-rotation
  attempts should be rejected by the existing gate; this is existing behaviour, not new scope.)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST accept a POST request to the `/host/rotate` path and treat it as a
  command to invalidate the current join window and activate a new one.
- **FR-002**: On a successful rotation, the system MUST respond with a status indicating success
  so the leaderboard page can refresh the QR code image.
- **FR-003**: On a rotation failure (e.g., join store unavailable), the system MUST respond with a
  status indicating failure so the leaderboard page can suppress the image refresh.
- **FR-004**: The `/host/rotate` endpoint MUST reject non-POST requests with a response indicating
  the method is not allowed.
- **FR-005**: A new join window activated by rotation MUST behave identically to windows activated
  by any other mechanism — the same TTL, the same gate enforcement, the same QR code generation
  path.
- **FR-006**: The rotation endpoint MUST have automated tests covering: successful rotation,
  store-error handling, and method-not-allowed rejection.

### Key Entities

- **Join Window**: A time-bounded token that authorises a player to enter the game. One window is
  active at a time. Rotation creates a new active window and implicitly supersedes the previous one.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Clicking "Refresh QR" on the leaderboard page produces a visibly updated QR code
  within 2 seconds in the live demo environment.
- **SC-002**: The 60-second auto-rotate timer updates the QR code on every cycle without any
  manual action, observable across at least three consecutive cycles.
- **SC-003**: All three automated test scenarios (success, store error, method rejection) pass in
  the CI test suite.
- **SC-004**: A player who scans a QR code from before the most recent rotation is not granted
  access to the game (existing gate behaviour confirmed unbroken after the fix).

## Assumptions

- The leaderboard page's JavaScript already sends `POST /host/rotate` in the right places; no
  frontend changes are needed beyond what the fixed backend enables.
- The existing join-window store interface already exposes an `Activate` operation that accepts a
  TTL and returns the new window. No store changes are required.
- The 60-second rotation interval is acceptable as-is; this spec does not propose changing it.
- Rotation is unauthenticated at the application layer — it is reachable by anyone who can load
  the leaderboard page, which is the same trust boundary that already applies to the leaderboard.
- The definition of done requires the fix to be observed working in a running browser session
  against the compose stack (per Constitution Principle IV).
