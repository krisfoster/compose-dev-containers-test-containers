# Feature Specification: Secure Microservice Port Exposure

**Feature Branch**: `020-secure-service-port-exposure`

**Created**: 2026-07-09

**Status**: Draft

**Input**: User description: "create a specification for issue 2 in arch-issues.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Score Integrity Protected from Direct Port Access (Priority: P1)

An attacker or curious attendee with access to the demo host machine cannot submit arbitrary scores by calling the scores service directly on its internal port — all score submissions must flow through the nginx authentication gate.

**Why this priority**: Score data integrity is central to the demo's leaderboard feature. If anyone at the booth can submit fake scores by bypassing the auth gate, the projected leaderboard becomes meaningless. This is the only write-path exposure in the system.

**Independent Test**: Can be fully tested by attempting a `POST /scores` directly to port 8083 from the host machine after the fix is applied. The test delivers value by confirming the auth gate cannot be bypassed regardless of the scores service being up.

**Acceptance Scenarios**:

1. **Given** the compose stack is running, **When** a host-machine client sends `POST /scores` directly to the scores service port, **Then** the connection is refused (port not reachable from host).
2. **Given** the compose stack is running, **When** a score is submitted through the nginx front door (`POST /api/leaderboard/scores`) with a valid `cw_grant` cookie, **Then** the score is accepted and appears on the leaderboard.
3. **Given** the compose stack is running, **When** a score is submitted through the nginx front door without a valid `cw_grant` cookie, **Then** the submission is rejected with a 401 response.

---

### User Story 2 - Read-Only Service Ports Protected from Host Access (Priority: P2)

The commits and QR services, while read-only, are not directly reachable from the host machine. Their data is only accessible via the nginx routing layer, preventing unintended data exposure during demos.

**Why this priority**: While commits and QR data have no write path, exposing raw microservice ports leaks internal API shape, enables scraping of session data, and is inconsistent with the principle that nginx is the single public ingress. Fixing this closes the exposure without affecting any demo functionality.

**Independent Test**: After applying the fix, attempting to reach `http://localhost:8082` (commits) or `http://localhost:8084` (qr-service) from the host machine results in a refused connection. All leaderboard and QR functionality continues to work normally through the nginx front door.

**Acceptance Scenarios**:

1. **Given** the compose stack is running, **When** a host-machine client attempts to reach the commits service directly on port 8082, **Then** the connection is refused.
2. **Given** the compose stack is running, **When** a host-machine client attempts to reach the QR service directly on port 8084, **Then** the connection is refused.
3. **Given** the compose stack is running, **When** the leaderboard page loads via the nginx front door, **Then** the commits feed SSE stream works correctly.
4. **Given** the compose stack is running, **When** the host page requests a QR code via the nginx front door, **Then** the QR image is served correctly.

---

### Edge Cases

- What happens if a developer needs direct port access for debugging? The fix should document that temporary port access can be restored via a compose override file or explicit profile, not by reverting the security change.
- What if a Compose service that routes through nginx itself needs to reach the microservices internally? Internal service-to-service communication over the Docker network (using service names, not host ports) is unaffected by changing `ports:` to `expose:`.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The scores service MUST NOT publish port 8083 to the host network; it MUST only be reachable by other services on the internal Compose network.
- **FR-002**: The commits service MUST NOT publish port 8082 to the host network; it MUST only be reachable by other services on the internal Compose network.
- **FR-003**: The QR service MUST NOT publish port 8084 to the host network; it MUST only be reachable by other services on the internal Compose network.
- **FR-004**: All functionality currently accessible through the nginx front door (score submission, commit feed, QR image) MUST continue to work after the port change.
- **FR-005**: The nginx service MUST remain the sole compose service with ports published to the host (port 80).
- **FR-006**: The change MUST be documented so future contributors understand that the port restriction is intentional and how to temporarily restore direct access if needed for debugging.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After applying the change, a connection attempt to `localhost:8083`, `localhost:8082`, and `localhost:8084` from the host machine is refused in under 1 second (connection refused, not timeout).
- **SC-002**: All demo flows — score submission via QR code, leaderboard display, commit feed — continue to function end-to-end without any change to user-visible behaviour.
- **SC-003**: The compose stack starts and reaches a demoable state via `docker compose up` with no additional configuration required.
- **SC-004**: A reviewer can determine from the compose file and surrounding documentation that the port restriction is deliberate, without needing to read commit history.

## Assumptions

- The nginx service is and remains the single public ingress at port 80; this assumption is already codified in the project constitution's Permanent Routing Layer carve-out.
- Internal service-to-service communication uses Docker Compose service DNS names (e.g., `scores-service:8083` from nginx's perspective), not host-published ports — changing `ports:` to `expose:` does not affect this communication.
- Developer convenience via direct port access is not a required day-to-day workflow; if it is needed occasionally, a compose override file is the appropriate mechanism.
- No changes to Go service code are required — this is purely a Compose configuration change.
- The fix applies to all three microservices (scores, commits, QR) for consistency, even though only the scores service has a write path.
