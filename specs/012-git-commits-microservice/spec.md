# Feature Specification: Git Commits Microservice

**Feature Branch**: `012-git-commits-microservice`

**Created**: 2026-07-08

**Status**: Draft

**Input**: User description: "Create a specification for splitting out the API that serves the git commits into its own go based micro service. On the leader board the commits should be pulled by a react js component that pulls the commits from the API. There must either be a polling, or push mechanism, for the api to pass more commits dynamically. The new go micro service must be brought up by docker compose up. The go api will not serve any html. If the react component calls the api and there are no commits, then it displays a message letting the user know that there are no commits."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Leaderboard Displays Live Commit Feed (Priority: P1)

A booth attendee looking at the leaderboard wall projection sees a live feed of recent git commits. When new commits arrive, the commit list updates automatically without requiring a page reload, so the projection stays fresh throughout the demo.

**Why this priority**: This is the core demo-visible outcome. Without a working commit feed on the leaderboard, the feature has no value. All other stories depend on this being in place first.

**Independent Test**: Can be tested by running the compose stack, making a git commit on any connected repository, and observing the leaderboard update the commit list within a few seconds — delivering a visible, real-time demo moment.

**Acceptance Scenarios**:

1. **Given** the leaderboard page is open in a browser, **When** a new git commit is pushed to the tracked repository, **Then** the commit appears in the leaderboard's commit list within 5 seconds without a page reload.
2. **Given** the leaderboard has commits loaded, **When** a presenter views the board after several minutes of inactivity, **Then** the commit list still reflects the current state (no stale data).
3. **Given** the compose stack is running, **When** the leaderboard page first loads, **Then** any existing commits are immediately displayed.

---

### User Story 2 - Empty State Feedback (Priority: P2)

A presenter opens the leaderboard at the start of a booth session before any commits have been recorded. Instead of a blank or broken UI, the leaderboard clearly communicates that no commits are available yet.

**Why this priority**: An empty UI with no explanation looks like a bug during a live demo. Clear empty-state messaging prevents presenter embarrassment and sets audience expectations.

**Independent Test**: Can be tested in isolation by pointing the leaderboard at a repository with no commits in the tracked window and verifying that the "no commits" message renders in place of the commit list.

**Acceptance Scenarios**:

1. **Given** the commits microservice has no commit data to return, **When** the leaderboard component requests commits, **Then** a clear, user-friendly message is displayed in place of the commit list (e.g., "No commits yet — make your first commit to see it here!").
2. **Given** the empty state message is displayed, **When** a commit is subsequently recorded, **Then** the empty state message is replaced by the commit entry automatically.

---

### User Story 3 - Compose-Orchestrated Service Startup (Priority: P3)

A developer or presenter with a fresh clone of the repository runs `docker compose up` and the commits microservice starts alongside all other services, with no additional manual steps required.

**Why this priority**: Consistent with the project's Compose-Orchestrated Reproducibility principle. If the service requires host-side setup beyond Docker Desktop and git, it violates the core constraint and becomes a demo liability.

**Independent Test**: Can be tested by performing a fresh clone on a machine with only Docker Desktop installed, running `docker compose up`, and confirming the commits microservice endpoint is reachable and returning valid responses.

**Acceptance Scenarios**:

1. **Given** a machine with only Docker Desktop and git installed, **When** `docker compose up` is run from a fresh clone, **Then** the commits microservice starts successfully and serves commit data.
2. **Given** the compose stack is running, **When** the commits microservice URL is called directly, **Then** it returns commit data in a structured format (not HTML).

---

### Edge Cases

- What happens when the commits microservice is temporarily unavailable? The leaderboard component should handle the connection failure gracefully and attempt to reconnect or retry, rather than displaying an error that looks broken on the projected wall.
- What happens when the commit list grows very large? The display should be bounded (e.g., show the N most recent commits) to prevent the leaderboard from becoming unreadable.
- What happens when the same commit is received more than once? The component should deduplicate commits to prevent duplicates in the displayed list.
- How does the system behave if the leaderboard page is loaded before the commits microservice finishes starting? The component should wait and retry rather than showing a permanent error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The commits data MUST be served by a dedicated microservice that is separate from any existing backend service and provides no HTML output.
- **FR-002**: The commits microservice MUST be defined as a service in `docker-compose.yml` so it starts with `docker compose up` with no additional host-side steps.
- **FR-003**: The commits microservice MUST expose an endpoint that returns structured commit data (not HTML) consumable by the leaderboard frontend.
- **FR-004**: The leaderboard MUST include a dedicated component responsible for fetching and displaying commit data from the commits microservice.
- **FR-005**: The leaderboard commits component MUST update the commit list dynamically — either via polling on a fixed interval or via a push mechanism (e.g., Server-Sent Events or WebSockets) — without requiring a full page reload.
- **FR-006**: When the commits microservice returns no commit data, the leaderboard commits component MUST display a clear, user-friendly message informing the user that no commits are available.
- **FR-007**: When commit data is available, the leaderboard commits component MUST display a bounded list of recent commits (most recent first).
- **FR-008**: The commits microservice MUST NOT serve any HTML content — all responses MUST be structured data only.
- **FR-009**: The leaderboard commits component MUST handle transient microservice unavailability gracefully without displaying an unrecoverable error state.
- **FR-010**: The commits microservice MUST be reachable by the leaderboard frontend within the compose network.

### Key Entities

- **Commit**: A single git commit record surfaced by the microservice. Attributes include at minimum: commit hash (short form), author name, commit message (subject line), and timestamp. No raw implementation storage details are exposed.
- **Commit Feed**: The ordered collection of recent commits returned by the microservice, bounded to a configurable or fixed maximum count, ordered by most-recent-first.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new git commit appears on the leaderboard within 5 seconds of being recorded, with no manual page refresh required.
- **SC-002**: The leaderboard renders a clear "no commits" message when zero commits are available — this is verifiable in under 30 seconds of inspection by a non-technical observer.
- **SC-003**: `docker compose up` from a fresh clone brings the commits microservice to a ready state within 60 seconds, with no additional host-side commands required.
- **SC-004**: The commit list on the leaderboard remains current (not stale) after 10+ minutes of continuous display, as would occur during a full booth session.
- **SC-005**: The leaderboard commits component recovers and resumes displaying data within 15 seconds after a transient microservice restart, without a page reload.

## Assumptions

- The existing leaderboard page is already rendered in the browser; this feature replaces or augments the current static or server-rendered commit display with a dynamic React component.
- The commits microservice reads git commit data from the same repository the project currently tracks for the `/api/commits` endpoint; sourcing logic is ported to the new service.
- Polling interval (if polling is chosen over push) defaults to 5 seconds — frequent enough for a live demo, infrequent enough to avoid overwhelming the service.
- The maximum number of commits displayed is bounded at a sensible default (e.g., 20 most recent) to keep the leaderboard readable on a projected screen; this may be made configurable in the implementation plan.
- The commits microservice communicates with the leaderboard frontend over the compose internal network; no external tunnel or separate authentication is required for this service-to-service call.
- The leaderboard frontend is already using or capable of using React components; no major frontend framework change is required.
- The existing `/api/commits` endpoint (if present in the current backend) will be retired or left in place as a compatibility shim — the implementation plan will decide which; this spec assumes the new microservice becomes the authoritative source.
