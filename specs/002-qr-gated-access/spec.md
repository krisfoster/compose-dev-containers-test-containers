# Feature Specification: QR-Gated Public Access to Crossy Whale

**Feature Branch**: `002-qr-gated-access`

**Created**: 2026-07-06

**Status**: Draft

**Input**: User description: "I want to add a hosted qr code that when a user views it with their camera on a phone opens the crossy whale app (that shoudl be the name of the game btw). The QR code has to be seen for a user to access the game - so in some way it has to secure things. Please do some research on how best to do this. The game should not be accessible over the public end point, using ngrok, unless the user has accessed the app through the qr coder"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Attendee scans the QR code and lands in the game (Priority: P1)

An attendee at the booth points their phone camera at a displayed QR code. Their phone recognizes
it as a link and offers to open it; tapping it opens Crossy Whale directly in their phone's
browser, ready to play, with no URL to type and no app to install.

**Why this priority**: This is the entire point of the feature — a booth attendee's only realistic
path into the game is a camera scan. If this doesn't work smoothly, nothing else matters.

**Independent Test**: Display the current QR code, scan it with a phone camera app (not a
dedicated QR reader), and confirm the phone's browser opens directly into a playable game screen.

**Acceptance Scenarios**:

1. **Given** a QR code is currently displayed at the booth, **When** an attendee scans it with
   their phone camera, **Then** their phone browser opens and shows the Crossy Whale game.
2. **Given** an attendee has just scanned the QR code and reached the game, **When** they play and
   finish a round, **Then** they are not asked to scan again to keep playing or to play another
   round in the same visit.
3. **Given** several attendees scan the QR code around the same time, **When** each of them plays,
   **Then** each has their own independent, concurrently active game instance that does not
   interfere with anyone else's, and each is distinguishable from the others.

---

### User Story 2 - Public access is blocked without a valid scan (Priority: P1)

Someone who has not scanned the current QR code — whether they guessed the public URL, found an
old link, or are simply poking around — cannot reach the playable game over the public (ngrok)
endpoint. They see a clear message telling them to find and scan the QR code instead of the game
or a broken/generic error page.

**Why this priority**: This is the security requirement driving the feature. Without it, the
public URL is just an open door and the QR code is decorative.

**Independent Test**: With public access enabled and a QR code currently displayed, open the
public URL directly in a fresh, unauthenticated browser (no prior scan) and confirm the game does
not load — an explanatory "scan the QR code" screen appears instead.

**Acceptance Scenarios**:

1. **Given** public access is enabled, **When** someone opens the bare public URL in a browser
   that has never gone through a valid QR link, **Then** they see a message directing them to scan
   the QR code, not the game itself.
2. **Given** no QR code has ever been generated for the current run, **When** anyone opens the
   public URL, **Then** the game is not reachable.
3. **Given** the local/presenter machine is running without public access enabled, **When** the
   presenter opens the game locally, **Then** they can play without scanning any QR code.

---

### User Story 3 - Presenter rotates the QR code to cut off stale access (Priority: P2)

The presenter can trigger a new QR code on demand. The moment they do, the previous code — whether
still on display, screenshotted, or shared — stops granting access. This lets the presenter close
the booth for the day, start a new session, or recover from a code being shared outside the
intended audience, without restarting the whole app.

**Why this priority**: A QR code that only ever expires on a timer is not enough control for a
live, presenter-run event; the presenter needs an immediate manual override.

**Independent Test**: With a QR code currently valid, trigger rotation, then attempt to use the
previous QR code's link — confirm it no longer grants access, while a scan of the newly displayed
code does.

**Acceptance Scenarios**:

1. **Given** a QR code is currently active and has already been used by some attendees, **When**
   the presenter rotates it, **Then** the previous code's link no longer grants new access.
2. **Given** the presenter has just rotated the QR code, **When** a new attendee scans the newly
   displayed code, **Then** they reach the game normally.
3. **Given** the presenter rotates the QR code, **When** attendees who already gained access before
   the rotation continue playing, **Then** their in-progress access is not interrupted.

---

### Edge Cases

- What happens when a QR code's validity period lapses on its own (no manual rotation) after
  being left displayed for a long time (e.g., overnight)? It must stop granting new access
  automatically, the same as a manual rotation would.
- What happens when someone forwards a screenshot of the QR code's underlying link to a friend who
  is not physically at the booth? The friend can gain access exactly as if they had scanned it
  themselves, as long as that code is still currently valid — the gate gives someone who was shown
  the code the same access as someone who scanned it directly, and the code going stale (via
  timeout or rotation) is what limits how far that sharing can spread. This is a deliberate,
  documented trade-off, not a gap to close in this feature.
- What happens when the public tunnel is unavailable but a QR code is still displayed? Scanning
  should fail the same way any other attempt to reach the unavailable public endpoint would, with
  no special QR-related error.
- What happens when a player's granted access lapses while they are mid-game? Their current game
  round is not interrupted; only a later page reload or new action against the public endpoint
  would require a fresh, valid scan.
- What happens when many attendees scan the same currently-valid QR code at once? All of them gain
  access; there is no cap on how many people one valid code can admit.
- What happens when two or more attendees are playing at the same time? Each holds their own
  independent access grant with its own unique identifier; their gameplay and access do not
  interfere with one another, and their eventual scores can be told apart.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a QR code that, when scanned by a standard phone camera app,
  opens Crossy Whale directly in the phone's browser without the user typing a URL.
- **FR-002**: System MUST display the current QR code somewhere the presenter can show attendees
  (e.g., on a wall/projector view), independent of the game screen itself.
- **FR-003**: System MUST block the game over the public endpoint for any visitor who has not
  reached it through a currently valid QR code link, and instead show a clear message telling them
  to scan the QR code.
- **FR-004**: System MUST NOT apply the QR-scan gate to access on the presenter's local machine
  when the public endpoint is not enabled, so local demoing and rehearsal are never blocked by it.
- **FR-005**: System MUST let a visitor who has already gained valid access keep playing and
  interacting with the game for the remainder of their visit without re-scanning.
- **FR-006**: System MUST expire a given QR code's ability to grant new access automatically after
  a bounded period of time, without requiring the presenter to do anything.
- **FR-007**: System MUST let the presenter manually invalidate the current QR code and produce a
  new one on demand, immediately cutting off new access via the old one.
- **FR-008**: System MUST NOT interrupt a visitor's already-granted access when the presenter
  rotates the QR code or when it expires — rotation and expiry only affect new access attempts.
- **FR-009**: System MUST fail closed: if no QR code has been generated yet for the current run,
  the public endpoint MUST NOT serve the playable game.
- **FR-010**: System MUST support multiple visitors holding independently valid access grants at
  the same time, with no fixed limit on how many concurrent grants are active, and no visitor's
  session affecting another's.
- **FR-011**: System MUST assign each Visitor Access Grant a unique identifier at the time access
  is granted. That identifier MUST remain stable for the remainder of the visit and MUST
  distinguish it from every other concurrently active grant, so that later actions — such as a
  leaderboard score submission — can be attributed to the correct individual instance of play.

### Key Entities

- **QR Access Code**: The current scannable code and the link it encodes. Has a validity period, a
  status (active / expired / rotated out), and is what a phone camera resolves into an entry link
  for Crossy Whale.
- **Visitor Access Grant**: Represents a visitor's device having successfully passed the QR gate.
  Carries a unique identifier assigned at grant time, stable for that visit and distinct from
  every other concurrently active grant. Once granted, it allows continued use of the game for
  that visit, independent of whether the QR Access Code it originated from later expires or is
  rotated. This identifier is what a future leaderboard feature would attribute a score to.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An attendee can go from scanning the QR code with their phone camera to seeing the
  Crossy Whale game loaded in under 10 seconds.
- **SC-002**: 100% of attempts to open the public game URL without a valid QR-code-derived access
  grant are blocked with an explanatory message rather than showing the game.
- **SC-003**: After the presenter rotates the QR code, previously issued codes stop granting new
  access effectively immediately (within a few seconds), with zero app restart required.
- **SC-004**: A QR code that is never manually rotated still stops granting new access on its own
  within its defined validity period, with zero presenter action.
- **SC-005**: The presenter can run and test the game on their local machine with zero QR-scanning
  steps at any point when public access is off.
- **SC-006**: Two or more attendees can play concurrently through independent access grants, each
  identifiable by its own unique ID, with zero cross-visitor interference and zero ID collisions
  observed.

## Assumptions

- The game being secured by this feature is named "Crossy Whale," per the feature request.
- This feature adds the public-access gate on top of the existing local + public (ngrok) hosting
  set up in a prior feature; it does not change how local hosting or the public tunnel itself are
  started.
- "Seen the QR code" is interpreted functionally as "used the current, unexpired link the QR
  encodes" rather than proven through the phone's camera specifically — there is no practical way
  to verify camera use itself. The gate's purpose is to prevent the public URL from being open to
  anyone who finds or guesses it, and to let the presenter cut off access on demand, not to create
  a tamper-proof, un-shareable secret.
- Default QR validity period is short enough to make an old, no-longer-displayed code stop working
  within one typical demo session (on the order of minutes, not hours); the presenter can shorten
  this at any time via manual rotation.
- A visitor's access grant, once obtained, lasts for the remainder of that visit/session and is
  not re-checked against the QR code's current validity on every subsequent action.
- The join/name-entry experience, leaderboard, and wall-display page referenced in the project's
  broader vision are separate concerns; this feature covers generating, displaying, and enforcing
  the QR-based access gate on the public endpoint, plus minting a unique identifier per Visitor
  Access Grant. Recording, storing, and displaying scores against that identifier on an actual
  leaderboard is left to a future feature.
- A single active QR code at a time is sufficient; there is no requirement for multiple
  simultaneously valid codes. Multiple concurrently active Visitor Access Grants (one per playing
  attendee) are required, regardless of how many QR codes have been active over time.
