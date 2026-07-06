# Feature Specification: Host Web App with Public Ngrok Access

**Feature Branch**: `001-host-webapp-ngrok`

**Created**: 2026-07-06

**Status**: Draft

**Input**: User description: "The web app needs to be hosted on a webserver. Define a docker compose file to bring up a webserver that servers the web app. Add support for ngrok, is there a container we can pull and use, to support hosting the web app on a publically accessible url"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Presenter hosts the app locally with one command (Priority: P1)

A presenter on-site at a booth starts the demo from a fresh clone of the repository with a single
command, and the web app is immediately viewable in a browser on the presenter's own machine, with
no manual install steps beyond the container runtime.

**Why this priority**: This is the foundation every other scenario depends on. If the app cannot be
reliably served locally, there is no demo regardless of public access.

**Independent Test**: On a clean machine with only the container runtime installed, run the
startup command and confirm the web app loads in a browser pointed at the local address within a
short, predictable time.

**Acceptance Scenarios**:

1. **Given** a fresh clone of the repository, **When** the presenter runs the single startup
   command, **Then** the web app is reachable in a browser on the presenter's machine without any
   additional setup step.
2. **Given** the web app is already running, **When** the presenter stops and restarts it, **Then**
   it becomes reachable again without code changes or reconfiguration.

---

### User Story 2 - Attendees reach the app over the public internet (Priority: P1)

An attendee who is not on the presenter's local network opens a shared public URL on their own
phone or laptop and reaches the same running web app the presenter sees locally.

**Why this priority**: Booth demos depend on attendees joining from personal devices over venue
wifi or cellular data, which is not on the presenter's local network. Without public reachability,
only the presenter can use the app.

**Independent Test**: From a network that is not the presenter's local network (e.g. a phone on
cellular data), open the shared public URL and confirm the same content and behavior as the local
view.

**Acceptance Scenarios**:

1. **Given** public hosting is enabled, **When** the presenter starts the app, **Then** a public
   URL becomes available and is easy for the presenter to find and share.
2. **Given** an attendee has the current public URL, **When** they open it from an external
   network, **Then** they see the same web app the presenter sees locally.
3. **Given** the app has been running for the length of a typical demo session, **When** an
   attendee opens the public URL, **Then** it still resolves to the running app without the
   presenter needing to intervene.

---

### User Story 3 - Local demo keeps working if public access fails (Priority: P2)

Public tunneling depends on a third-party service and the venue's internet connection, both of
which can fail during a live demo. When that happens, the presenter can keep demoing locally
without the whole app going down.

**Why this priority**: Demo resilience is explicitly a project goal; a public-access outage should
degrade the experience, not end it.

**Independent Test**: With the app running and public access enabled, simulate the public tunnel
being unavailable (e.g. no internet, or the tunnel service down) and confirm the web app is still
reachable locally.

**Acceptance Scenarios**:

1. **Given** the web app is running with public access enabled, **When** the public tunnel becomes
   unavailable, **Then** the web app remains reachable on the presenter's local machine.
2. **Given** the public tunnel is unavailable, **When** the presenter checks how to share the app,
   **Then** it is clear that public sharing is currently unavailable rather than failing silently.

---

### Edge Cases

- What happens when the public-tunnel credential/token is missing or invalid? The app should still
  come up and be usable locally, with a clear indication that public access is unavailable.
- What happens when the host machine's webserver port is already in use? Startup should fail with
  a clear, actionable error rather than silently binding to an unexpected port.
- What happens when the public URL changes between restarts (expected with a non-reserved public
  tunnel)? The presenter must be able to easily find the current URL each time it changes.
- What happens when the public tunnel provider throttles or rate-limits the connection? Local
  access must be unaffected.
- What happens when a second person starts the same setup on another machine at the same time
  (e.g. a rehearsal alongside the live booth machine)? Each instance gets its own independent
  local and public access without colliding with the other.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST serve the web app's content over HTTP from a single startup command,
  requiring no manual host-level software installation beyond the container runtime.
- **FR-002**: System MUST make the web app reachable locally on the presenter's machine immediately
  after startup.
- **FR-003**: System MUST be able to make the web app reachable from outside the presenter's local
  network via a public URL.
- **FR-004**: Public reachability MUST be an explicit, optional mode rather than always-on, so the
  presenter can choose to run local-only when public access is not needed or not available.
- **FR-005**: System MUST make the current public URL discoverable at a single, fixed local
  address the presenter can open on demand (e.g. a status/inspection page), without requiring the
  presenter to scrape logs or guess, so it can be shared with attendees.
- **FR-006**: System MUST keep the web app reachable locally even when public access is enabled but
  currently unavailable (e.g. missing credentials, provider outage, no internet).
- **FR-007**: System MUST allow the presenter to stop and restart hosting (including re-establishing
  public access) without any code change.
- **FR-008**: System MUST report a clear, actionable failure if the local webserver cannot start
  (e.g. a port conflict), rather than failing silently.

### Key Entities

- **Hosted Web App**: The running instance of the web app content being served; has a reachability
  state (local, and optionally public).
- **Public Access Endpoint**: The current shareable public URL, when public access is enabled and
  available; has a status (available / unavailable) and may change across restarts.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A presenter can go from a fresh clone to viewing the web app locally in under 2
  minutes using a single command.
- **SC-002**: An attendee on a network separate from the presenter's (e.g. cellular data) can load
  the web app from the shared public URL in under 5 seconds after opening it.
- **SC-003**: A shared public URL remains usable for at least a 2-hour demo session without any
  presenter intervention.
- **SC-004**: When public access becomes unavailable during a session, the presenter can continue
  the demo locally with zero recovery steps beyond using the local address.
- **SC-005**: 100% of hosting startups (local and public) complete without any manual, undocumented
  workaround.

## Assumptions

- "The web app" refers to the existing browser-based frontend in this repository; hosting a
  separate backend API is a related but distinct future feature and is out of scope here.
- A public tunneling approach (as already established for this project) is the mechanism for
  public reachability; a free-tier account/credential is sufficient, and a reserved/static public
  address is an optional enhancement, not a requirement, for this feature.
- Access control for the public URL (join windows, session gating, or similar) is out of scope for
  this feature — anyone who has the current URL can reach the app. Restricting who can join is a
  separate future feature.
- Local-network-only access (no tunnel, e.g. sharing a LAN address directly) and a no-tunnel
  fallback ("kiosk") mode described in the broader project vision are out of scope for this
  feature, which covers local hosting plus optional public hosting only.
- A single running instance is sufficient for a booth demo; multi-instance scaling or load
  balancing is not required.
