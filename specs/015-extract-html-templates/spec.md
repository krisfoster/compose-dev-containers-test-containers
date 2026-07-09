# Feature Specification: Go HTML Template Extraction with Live Reload

**Feature Branch**: `015-extract-html-templates`

**Created**: 2026-07-09

**Status**: Draft

**Input**: User description: "Create a specification for extracting the HTML from the go apps. They can be extracted out to some form of standard templating technology that is common to the go eco system. If possible it would be prefferred if the templates can be live updated, which means that if they are edited the changes ae picked up and the end app is pinged to refresh and load the changes in the browser (note we are already doing this for some things in this repo)."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Developer Edits a Template and Sees Changes Without Restarting (Priority: P1)

A developer working on the presenter or game HTML wants to tweak copy, layout, or styling. Currently the HTML lives as an inline Go string constant, meaning any edit requires rebuilding and restarting the Go service. After this feature, the developer edits a template file on disk, and the running browser reflects the change promptly — no service rebuild or restart needed.

**Why this priority**: This is the core value of the feature. Removing the edit→rebuild→restart loop is the visible demo improvement that justifies the work (Principle I: Demo-First Delivery). All other stories depend on templates existing as external files.

**Independent Test**: Open `/host` in a browser. Edit the host template file to change a heading. The browser reloads and displays the updated heading without any `docker compose restart`.

**Acceptance Scenarios**:

1. **Given** the compose stack is running and a developer has `/host` open in a browser, **When** the developer edits the host page template file and saves it, **Then** the browser automatically reloads and displays the updated HTML within a few seconds.
2. **Given** the compose stack is running and a developer has `/` open in a browser, **When** the developer edits the getting-started page template file and saves it, **Then** the browser automatically reloads and shows the change without any manual intervention.
3. **Given** a template file is modified while no browser is connected, **When** a browser subsequently opens the page, **Then** it displays the latest version of the template.

---

### User Story 2 - All Inline HTML Constants Replaced by External Template Files (Priority: P1)

Currently, three pages have their HTML embedded as Go string constants in `main.go`: the getting-started landing page (`/`), the host presenter page (`/host`), and the leaderboard page (`/leaderboard`). A developer who wants to read or edit page structure must navigate Go source code rather than a dedicated template file. After this feature, each page's HTML lives in a named template file, and finding or editing a page means opening that file directly.

**Why this priority**: This is the structural prerequisite for User Story 1 and for any future UI work. Template files are independently navigable, version-controllable, and editable without touching Go source.

**Independent Test**: Can be tested by confirming that `main.go` contains no multi-line HTML string constants and that each previously-inline page renders correctly from its external file.

**Acceptance Scenarios**:

1. **Given** the updated app is running, **When** a developer inspects the Go source, **Then** `main.go` contains no multi-line inline HTML strings; the HTML for each page is found only in its dedicated template file.
2. **Given** the app is running with all templates externalised, **When** a browser visits `/`, `/host`, and `/leaderboard`, **Then** each page renders identically to how it rendered before the change (no visual or functional regression).
3. **Given** the app is running, **When** a browser visits `/play`, **Then** the game loads correctly and the leaderboard token is still injected into the page (the one page already using `template.ParseFiles` continues to work).

---

### User Story 3 - Template Syntax Errors Are Reported Clearly (Priority: P2)

A developer editing a template file accidentally introduces a syntax error (for example, an unclosed `{{` block). The browser should not hang silently or show a blank page — it should display a clear error message, and the terminal log should explain what went wrong so the developer can fix it quickly.

**Why this priority**: A developer workflow without actionable error feedback is frustrating and slows iteration. This does not block the core loop (User Stories 1 and 2) but is important for day-to-day usability.

**Independent Test**: Introduce a deliberate `{{` syntax error in one template file. Open the corresponding page. Verify the browser shows an error message and the terminal shows which file and what error.

**Acceptance Scenarios**:

1. **Given** a template file contains a syntax error, **When** a browser requests the corresponding page, **Then** the browser displays an error page (not a blank page or silent hang) and the terminal log names the file and error.
2. **Given** a template file had a syntax error that the developer then fixed and saved, **When** a browser requests the page, **Then** the corrected page renders normally.

---

### Edge Cases

- What happens when a template file is deleted while the app is running? → The app should serve a clear 500-level error with a log message identifying the missing file, not crash.
- What happens when two browser tabs are open to the same page and a template changes? → Both tabs should receive the reload signal and refresh independently.
- What happens when a template change occurs during an in-flight request? → The in-flight request may return either the old or new content; no crash or partial response is acceptable.
- What happens when the templates directory is missing at startup? → The app should fail to start with a clear error message, not start and silently serve 500s on first request.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: All HTML currently embedded as inline Go string constants in the Go app source (the getting-started landing page, the host presenter page, and the leaderboard page) MUST be moved to external template files on disk using Go's standard `html/template` package (already used for `/play` and `/leaderboard` partial rendering).
- **FR-002**: The app MUST re-read template files from disk on each request (or detect changes and invalidate a cache) so that an edit to a template file is reflected in the very next browser request to that page — without rebuilding or restarting the Go service.
- **FR-003**: When a template file is modified on disk, browsers that currently have the affected page open MUST be automatically prompted to reload within a few seconds, using the existing auto-reload polling mechanism already present in this repo (the `/api/ping` endpoint with a startup ID, already used by the leaderboard page) extended or adapted to cover template-level changes.
- **FR-004**: The template files MUST be co-located in a dedicated directory that is mounted into the running container so that edits on the host are visible inside the container without an image rebuild — consistent with how the app already mounts game assets and the `.git` volume for other live-updated content.
- **FR-005**: Existing functionality for all pages MUST be preserved: dynamic values injected at render time (such as the leaderboard API token on `/play`, the service URLs on `/leaderboard`) MUST continue to be injected correctly via template data, not hardcoded.
- **FR-006**: If a template file cannot be parsed (syntax error) or cannot be found (missing file), the app MUST return an appropriate HTTP error response with a user-visible error message and MUST log the file path and error detail; it MUST NOT crash.
- **FR-007**: The `docker-compose.yml` MUST be updated to mount the templates directory from the host into the app container so that template edits on the host reach the running container immediately.

### Key Entities

- **Template file**: A named `.html` file on disk containing Go `html/template` syntax; one per page (getting-started, host, leaderboard) plus the existing play/index template.
- **Templates directory**: A single top-level directory in the repository holding all page template files, mounted into the app container at a configurable path.
- **Template change signal**: The mechanism by which the app communicates to open browsers that a template has changed and they should reload — extended from or compatible with the existing `/api/ping` live-reload mechanism.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can edit any page template and see the change reflected in an open browser tab within 5 seconds, without running any `docker compose` command.
- **SC-002**: All three previously-inline pages (`/`, `/host`, `/leaderboard`) and the existing `/play` page render correctly after the change, with no visual or functional regression observable in the browser.
- **SC-003**: A template syntax error produces a visible error message in the browser and a file-path-and-error entry in the container logs within 2 seconds of the bad file being saved.
- **SC-004**: The on-disk `main.go` contains zero multi-line HTML string constants after the change (verifiable by inspection).
- **SC-005**: A fresh `docker compose up` from a clean checkout reaches a fully working demoable state with no extra steps, consistent with Principle II (Compose-Orchestrated Reproducibility).

## Assumptions

- The Go `html/template` package is the templating technology; it is already used in this codebase for `/play` and `/leaderboard` rendering, so no new runtime dependency is introduced.
- Template files are placed under a new top-level `templates/` directory in the repository (name is an implementation choice but assumed during spec writing).
- The existing `/api/ping` live-reload mechanism (already polling in the leaderboard page at 2-second intervals) is the foundation for the browser-reload signal on template change; extending it rather than introducing a separate WebSocket or SSE channel is the assumed approach, consistent with the "no new runtime dependency" constraint.
- The app container already has a configurable `FRONTEND_DIR` pattern for externally-mounted content; a similar `TEMPLATES_DIR` environment variable is the assumed extension point.
- Live reload during development (in the compose stack with a volume mount) is the primary use case; production image builds bake the template files into the image, so a volume mount is not required in production.
- The `commits-service` and `scores-service` microservices have no inline HTML and are out of scope for this feature.
