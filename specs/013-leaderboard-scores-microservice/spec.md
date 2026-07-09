# Feature Specification: Leaderboard Scores Microservice

**Feature Branch**: `013-leaderboard-scores-microservice`

**Created**: 2026-07-08

**Status**: Draft

**Input**: User description: "create a specification for serving up the leader baord score to the leader board page as its own micro service, written in go. The leaderboard page uses a react js script tocall the api and display the leader board list. Using the same mechanisn as used for updating the list of commits, you should make sure that the leaderboard is updated dynamically."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Leaderboard Displays Live Score Standings (Priority: P1)

A booth attendee looking at the leaderboard wall projection sees the current score standings. When a player completes a run and submits a score, the standings update automatically without requiring a page reload, so the projected display stays current throughout the demo.

**Why this priority**: This is the core demo-visible outcome. Without live score updates on the leaderboard, the feature delivers no value. All other stories depend on this foundation.

**Independent Test**: Can be tested by running the compose stack, submitting a score via the game, and observing the leaderboard standings update within a few seconds — delivering the key visible demo moment.

**Acceptance Scenarios**:

1. **Given** the leaderboard page is open in a browser, **When** a player submits a new score, **Then** the updated standings appear on the leaderboard within 5 seconds without a page reload.
2. **Given** the leaderboard has standings loaded, **When** a presenter views the board after several minutes of inactivity, **Then** the standings still reflect the current state (no stale data).
3. **Given** the compose stack is running, **When** the leaderboard page first loads, **Then** any existing standings are immediately displayed.

---

### User Story 2 - Empty State Feedback (Priority: P2)

A presenter opens the leaderboard at the start of a booth session before any scores have been recorded. Instead of a blank or broken UI, the leaderboard clearly communicates that no scores are available yet.

**Why this priority**: An empty standings column with no explanation looks broken during a live demo. Clear empty-state messaging prevents confusion and sets audience expectations.

**Independent Test**: Can be tested in isolation by pointing the leaderboard at a fresh Redis instance with no score data and verifying that the "no scores yet" message renders in place of the standings list.

**Acceptance Scenarios**:

1. **Given** the leaderboard microservice has no score data to return, **When** the leaderboard React component requests scores, **Then** a clear, user-friendly message is displayed in place of the standings list (e.g., "No scores yet — be the first to play!").
2. **Given** the empty state message is displayed, **When** a score is subsequently submitted, **Then** the empty state message is replaced by the standings automatically.

---

### User Story 3 - Compose-Orchestrated Service Startup (Priority: P3)

A developer or presenter with a fresh clone of the repository runs `docker compose up` and the leaderboard scores microservice starts alongside all other services, with no additional manual steps required.

**Why this priority**: Consistent with the project's Compose-Orchestrated Reproducibility principle. If the service requires host-side setup beyond Docker Desktop and git, it violates the core constraint and becomes a demo liability.

**Independent Test**: Can be tested by performing a fresh clone on a machine with only Docker Desktop installed, running `docker compose up`, and confirming the leaderboard scores microservice endpoint is reachable and returning valid responses.

**Acceptance Scenarios**:

1. **Given** a machine with only Docker Desktop and git installed, **When** `docker compose up` is run from a fresh clone, **Then** the leaderboard scores microservice starts successfully and is reachable from the leaderboard page.
2. **Given** the compose stack is running, **When** the leaderboard scores microservice endpoint is called directly, **Then** it returns score data in a structured format (not HTML).

---

### Edge Cases

- What happens when the leaderboard scores microservice is temporarily unavailable? The React component should handle the connection failure gracefully and attempt to reconnect, rather than displaying a permanent error on the projected wall.
- What happens when the scores list grows very large? The display is bounded by a configurable maximum (default 10) so the leaderboard remains readable regardless of total submission count.
- What happens if the leaderboard page is loaded before the microservice finishes starting? The component should wait and retry rather than showing a permanent error.
- What happens if a score update arrives that does not change the standings order? The component should re-render with the updated data without flicker or visible disruption.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Score standing data MUST be served by a dedicated microservice that is separate from the existing `app` backend and provides no HTML output.
- **FR-002**: The leaderboard scores microservice MUST be defined as a service in `docker-compose.yml` so it starts with `docker compose up` with no additional host-side steps.
- **FR-003**: The leaderboard scores microservice MUST expose an endpoint that returns structured score standing data (not HTML) consumable by the leaderboard frontend.
- **FR-004**: The leaderboard scores microservice MUST expose a streaming endpoint (using Server-Sent Events, matching the pattern used by the commits microservice) so the React component can receive live score updates without polling.
- **FR-004a**: The leaderboard scores microservice MUST subscribe to a Redis pub/sub channel and push an SSE event to all connected clients when a score change notification is received on that channel.
- **FR-004b**: The `app` service MUST publish a notification to the agreed Redis pub/sub channel immediately after successfully persisting a new score, so the leaderboard scores microservice is alerted in real time.
- **FR-005**: The leaderboard page MUST include a React component responsible for fetching and displaying score standings from the leaderboard scores microservice.
- **FR-006**: The leaderboard React component MUST update the standings list dynamically via the SSE stream — without requiring a full page reload — whenever a new score is submitted.
- **FR-007**: When the leaderboard scores microservice returns no score data, the React component MUST display a clear, user-friendly message informing the user that no scores are available.
- **FR-008**: When score data is available, the React component MUST display a bounded list of top standings (most points first). The maximum number of displayed entries MUST be configurable via an environment variable on the microservice, with a default of 10.
- **FR-009**: The leaderboard scores microservice MUST NOT serve any HTML content — all responses MUST be structured data only.
- **FR-010**: The React component MUST handle transient microservice unavailability gracefully without displaying an unrecoverable error state.
- **FR-011**: The leaderboard scores microservice MUST be reachable by the leaderboard React component within the compose network.
- **FR-012**: The leaderboard scores microservice MUST read score data from the same Redis instance used by the existing `app` service, so that scores submitted via the game are immediately visible on the leaderboard.
- **FR-013**: The existing score submission endpoint (`POST /api/leaderboard/scores` on the `app` service) MUST continue to function, with the addition that it publishes a notification to the agreed Redis pub/sub channel after each successful score write (FR-004b). Score writes remain with the `app` service.
- **FR-014**: The existing score read endpoint (`GET /api/leaderboard/scores`) and all supporting score-read code in the `app` service MUST be removed as part of this feature, with the new microservice becoming the sole provider of score standing data.
- **FR-015**: The leaderboard page's existing polling-based score fetch (JavaScript in the `app` service's leaderboard page template) MUST be removed and replaced by the new React component.

### Key Entities

- **Score Entry**: One row per player representing their single highest score. Attributes include at minimum: player name and best score value. A player who submits multiple scores appears only once, with their top score.
- **Standings List**: The ordered collection of per-player best-score entries returned by the microservice, bounded to a configurable or fixed maximum count, ordered by highest-score-first.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A newly submitted score appears on the leaderboard standings within 5 seconds of submission, with no manual page refresh required.
- **SC-002**: The leaderboard renders a clear "no scores" message when zero scores are recorded — verifiable in under 30 seconds of inspection by a non-technical observer.
- **SC-003**: `docker compose up` from a fresh clone brings the leaderboard scores microservice to a ready state within 60 seconds, with no additional host-side commands required.
- **SC-004**: The standings remain current (not stale) after 10+ minutes of continuous display, as would occur during a full booth session.
- **SC-005**: The leaderboard React component recovers and resumes displaying data within 15 seconds after a transient microservice restart, without a page reload.

## Assumptions

- The existing leaderboard page is already rendered by the `app` service; this feature replaces the current standings column's data-fetching mechanism with a React component backed by the new microservice, and removes the now-redundant read path from `app`.
- The leaderboard scores microservice connects to the same Redis instance as the `app` service to read score data and subscribe to the score-change pub/sub channel; it does not own score writes.
- The SSE streaming pattern used by the commits microservice (`GET /commits/stream`) is reused as the model for this microservice's live update stream; the trigger source is Redis pub/sub rather than polling.
- The React component for scores is vendored under `frontend/leaderboard/` alongside the existing commits component, following the Leaderboard React carve-out in the project constitution.
- The leaderboard scores microservice is a read-only service; all score submission logic remains in the `app` service.
- The maximum number of displayed standings is configurable via an environment variable on the microservice, defaulting to 10.

## Clarifications

### Session 2026-07-08

- Q: Should this feature include extracting and removing the existing score read endpoint (`GET /api/leaderboard/scores`) and supporting code from the `app` service? → A: Yes — full extraction. The new microservice becomes the sole provider of score read data; the existing read path in `app` is removed as part of this feature.
- Q: When a player submits multiple scores, should the leaderboard show each submission or only their best? → A: One row per player showing their highest score (best-score aggregation).
- Q: How many standings rows should the leaderboard display at maximum? → A: Configurable via environment variable on the microservice, defaulting to 10.
- Q: How should the microservice detect new scores to trigger SSE pushes? → A: Redis pub/sub — `app` publishes a notification to a Redis channel on each successful score write; the microservice subscribes and pushes SSE to all connected clients on receipt.
