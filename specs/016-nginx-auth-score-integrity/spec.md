# Feature Specification: nginx Auth Request for Score Integrity

**Feature Branch**: `016-nginx-auth-score-integrity`

**Created**: 2026-07-09

**Status**: Draft

**Input**: User description: "review @specs/016-nginx-auth-score-integrity/handoff.md and create a specification using it"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Score Submission Requires QR Grant (Priority: P1)

A game player who arrived via the QR code and holds a valid session grant can submit their score
at the end of a game. A visitor who never scanned the QR code — or whose grant has expired — is
denied at the network boundary before the score endpoint ever processes the request.

**Why this priority**: Score integrity is the primary goal. If a grant-less request can reach the
score endpoint at all, the feature has not delivered its core promise.

**Independent Test**: Send a score submission request with and without a valid QR session grant.
With a valid grant: submission is accepted and appears in the leaderboard. Without a grant: the
submission is rejected with a 401 before reaching any game-logic handler.

**Acceptance Scenarios**:

1. **Given** a player holds a valid QR session grant, **When** they complete a game and their
   score is submitted, **Then** the submission succeeds and the score appears on the leaderboard.

2. **Given** a visitor has no session grant, **When** a score submission is attempted (even with
   a correctly shaped payload), **Then** the request is rejected with a 401 and no score is
   recorded.

3. **Given** a player's session grant has expired, **When** they attempt to submit a score,
   **Then** the request is rejected with a 401 and no score is recorded.

---

### User Story 2 - No Extractable Token in the Browser (Priority: P1)

A player who opens the browser's developer tools or views the game page source sees no embedded
secret, API key, or reusable credential that could be copied and used to craft arbitrary score
submissions outside the game.

**Why this priority**: Eliminating the visible token closes the primary attack vector identified in
the research phase. It is equally foundational to the feature as grant-gated submission.

**Independent Test**: Load the game page after feature delivery. Inspect the page source and all
JavaScript globals. Confirm no leaderboard credential appears anywhere.

**Acceptance Scenarios**:

1. **Given** the game page has loaded, **When** a player searches the page source and all
   JavaScript globals for a leaderboard credential, **Then** no token, secret, or equivalent
   credential is found.

2. **Given** the game is running, **When** a player intercepts the score submission network
   request, **Then** the request carries no separate API secret header — only the standard session
   cookie that authorises game access.

---

### User Story 3 - Game Access and Score Submission Use One Credential (Priority: P2)

The same QR session grant that lets a player load and play the game is the only credential they
need to submit a score. There is no separate token for the score endpoint. Obtaining game access
automatically grants score submission capability; losing game access (expired grant) automatically
revokes it.

**Why this priority**: A single trust boundary is simpler to reason about, harder to misconfigure,
and matches the access model players actually experience — one QR scan, one continuous session.

**Independent Test**: Scan the QR code on a fresh device. Play the game. Confirm the score
submits successfully without any additional credential exchange. Expire the grant (wait for TTL or
revoke it). Confirm a score submission then fails.

**Acceptance Scenarios**:

1. **Given** a player has scanned the QR code and holds a valid grant, **When** they complete a
   game, **Then** the score submission succeeds using only that grant — no additional token
   exchange occurs.

2. **Given** a player's grant is the only credential present, **When** the grant expires, **Then**
   any further score submission attempt is rejected — there is no fallback credential path.

---

### User Story 4 - Dead Auth Path Removed (Priority: P2)

The previous token-based score authentication mechanism — both the server-side validation and the
browser-side token injection — is fully removed. No vestigial token fields, environment variables,
or config entries remain that could cause confusion or be accidentally re-enabled.

**Why this priority**: Shipping both auth paths simultaneously creates a false sense of security
and leaves misleading code in place. Clean removal prevents future re-introduction of the
vulnerability.

**Independent Test**: After delivery, search the entire codebase and configuration for any
reference to the old leaderboard API secret or token mechanism. Confirm zero matches. Verify the
compose stack starts and operates correctly without the removed configuration.

**Acceptance Scenarios**:

1. **Given** the feature is shipped, **When** the codebase is searched for references to the
   old credential mechanism, **Then** no references remain in source code, templates, or
   configuration.

2. **Given** the compose stack starts without any leaderboard API secret configured, **When** a
   player with a valid QR grant submits a score, **Then** it succeeds — the removed variable is
   not required.

---

### Edge Cases

- What happens when a player's QR grant expires mid-game (between game start and score
  submission)?
- How does the system behave if the grant-validation endpoint is unreachable at submission
  time?
- What if a player submits multiple scores in rapid succession — does each submission get
  individually validated?
- What happens if a player has a valid grant but submits a malformed score payload?
- Does grant expiry during an in-progress game produce a confusing user-facing error, and is
  that acceptable for the booth demo use case?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST reject any score submission request that does not carry a valid,
  unexpired QR session grant, returning a 401 response before the score is processed.

- **FR-002**: The system MUST accept score submission requests that carry a valid, unexpired QR
  session grant, forwarding them for normal score processing.

- **FR-003**: The game page MUST NOT embed, expose, or inject any leaderboard API secret, token,
  or equivalent credential into the browser environment (page source, JavaScript globals, or
  response headers).

- **FR-004**: Score submission requests MUST carry only the session grant cookie for
  authentication — no separate API secret header is required or accepted.

- **FR-005**: The grant-validation path used for score submission MUST reuse the same grant
  validity rules already applied when a player loads the game — same HMAC verification, same
  time-window TTL enforcement.

- **FR-006**: The service MUST expose an internal grant-check endpoint that can be called as part
  of the score submission authorisation flow, returning 200 for a valid grant and 401 for an
  absent, invalid, or expired one.

- **FR-007**: The grant-check endpoint MUST be unreachable by external clients directly — it is
  only invocable as an internal sub-request during request processing.

- **FR-008**: All environment configuration entries and source-code constructs belonging solely to
  the removed token mechanism MUST be deleted; none may be left commented out or set to empty
  strings as dead code.

- **FR-009**: The application MUST operate correctly as a single-port service after the removal of
  the second (gated) internal listener, with all routes handled through one port.

- **FR-010**: The routing layer's grant enforcement MUST delegate all validation logic to the Go
  application; the routing layer itself MUST contain no grant-checking business logic.

### Key Entities

- **QR Session Grant**: A short-lived, cryptographically signed credential issued when a player
  scans the event QR code. Carries the player's identity and an expiry window. Present as a cookie
  in all subsequent requests from that player's device.

- **Grant-Check Endpoint**: An internal-only endpoint exposed by the application that accepts an
  incoming request (with its cookies forwarded), validates the session grant, and returns 200 or
  401.

- **Score Submission Request**: A POST request from a game client containing the player's score
  payload, sent after a game round ends.

- **Leaderboard API Secret** *(to be retired)*: The current embedded token used to authenticate
  score submissions. This entity ceases to exist after the feature ships.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A score submission attempt without a QR session grant is rejected 100% of the time
  before reaching the score-recording layer.

- **SC-002**: A score submission with a valid QR session grant succeeds and appears on the
  leaderboard within the same latency envelope as before this change.

- **SC-003**: The game page source contains zero references to a leaderboard credential or API
  secret after delivery.

- **SC-004**: The compose stack reaches demoable state with zero configuration of a leaderboard
  API secret — the environment variable is absent from all compose files.

- **SC-005**: The codebase contains zero references to the removed leaderboard credential
  mechanism after delivery (measured by automated search).

- **SC-006**: A legitimate player — one who scanned the QR code and holds a current grant — can
  submit a score without any additional credential step; the QR grant is the sole authorisation
  required.

## Assumptions

- The QR session grant (`cw_grant` cookie) is already issued and managed by the existing gate
  system; this feature reuses it without changing how grants are issued or revoked.
- Grant validity rules (HMAC verification and time-window TTL) are not changing as part of this
  feature — the same rules apply to both game-page access and score submission.
- The routing layer already proxies `/api/leaderboard/scores` to the Go application; this feature
  adds authorisation to that existing proxy route.
- Session-level score integrity (preventing a legitimate grant-holder from hand-crafting an
  inflated score) is explicitly out of scope for this feature — that requires server-side play
  token tracking and is a follow-on effort.
- The booth demo does not require a meaningful error experience for players whose grants expire
  mid-game; a 401 response is an acceptable outcome for that edge case.
- The two-listener (dual-port) architecture in the Go application exists solely to separate gated
  from ungated routes; once the routing layer handles that separation via grant-check sub-requests,
  the second listener becomes redundant and is removed.
- All changes — grant-check endpoint addition, token mechanism removal, dual-listener removal —
  ship as a single atomic change; a half-migrated state where both auth paths exist simultaneously
  is not acceptable.
