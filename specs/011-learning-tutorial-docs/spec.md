# Feature Specification: Learning Tutorial and Documentation

**Feature Branch**: `011-learning-tutorial-docs`

**Created**: 2026-07-07

**Status**: Draft

**Input**: User description: "look at issues 8 and 9 in issues.md and create a specification"

## Clarifications

### Session 2026-07-07

- Q: Should Kubernetes deployment be covered in the README tutorial? → A: Yes — deploy to a local Kubernetes cluster (Docker Desktop with Kubernetes enabled). Instructions are in `k8s/README.md`. Docker Compose Bridge and Helm chart generation from a compose file must NOT be mentioned.
- Q: Should `k8s/README.md` be enriched with educational context, or left as bare commands? → A: Enrich it with a brief intro explaining Kubernetes and Helm at a high level — no deep dives.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Developer explores code to learn Docker (Priority: P1)

A developer new to Docker clones the repository and navigates through the codebase. They find inline comments in the Go source files and Docker configuration files that explain what each piece does and why, using accessible language. They leave the repo with a concrete understanding of Docker Compose, Docker images, Testcontainers, Docker Hardened Images, and Kubernetes deployment.

**Why this priority**: Inline code explanations are the first thing a reader encounters when exploring files. Without them, the README tutorial has no payoff — the reader clicks into source and finds unexplained code. This is the foundational layer.

**Independent Test**: Can be tested by opening any Go source file and any Docker-related config file and verifying that a reader unfamiliar with Docker can understand what the file does and why key choices were made, without needing external reference material.

**Acceptance Scenarios**:

1. **Given** a developer opens `docker-compose.yml`, **When** they read the file top to bottom, **Then** they find a comment block at the start of each service explaining what it does and why it exists in the stack.
2. **Given** a developer opens a Go source file that starts or manages a Testcontainer, **When** they read the surrounding code, **Then** they find a comment explaining the Testcontainers pattern and why a real container is used instead of a mock.
3. **Given** a developer opens the Dockerfile or image reference for a DHI image, **When** they read the surrounding config, **Then** they find a comment explaining what Docker Hardened Images are and why the project uses them.
4. **Given** a developer opens any Go source file, **When** they encounter an idiomatic Go pattern (interface, goroutine, channel, struct embedding), **Then** they find a short comment explaining the pattern's purpose — not restating the code but explaining the choice.

---

### User Story 2 - Reader works through README as a structured tutorial (Priority: P1)

A reader opens the README and finds a learning-outcomes-first structure. A "What you will learn" section up front tells them exactly which Docker technologies they will encounter. The body of the README is a tutorial that progresses logically from running the app to understanding how each Docker technology fits together, ending with deploying the app to a local Kubernetes cluster. They finish with a summary that recaps what they have learned.

**Why this priority**: Equal priority to inline docs because the README is the entry point for most readers. Both P1 stories must be complete for the feature to deliver its core learning value.

**Independent Test**: Can be tested by having someone unfamiliar with the project read the README end to end and answer a short quiz on the Docker technologies covered. The README is also testable independently of inline code comments.

**Acceptance Scenarios**:

1. **Given** a reader opens the README, **When** they read the first substantive section, **Then** they find a "What you will learn" section listing the Docker technologies and concepts the tutorial covers.
2. **Given** a reader follows the tutorial, **When** they reach each tutorial section, **Then** the section instructs them to run a command or look at a specific piece of code — never to write code.
3. **Given** a reader follows the tutorial in order, **When** they reach the end, **Then** the progression feels logical: running the app → understanding how it is composed → understanding how it is built → understanding how it is tested → deploying to Kubernetes.
4. **Given** a reader reaches the end of the README, **When** they read the closing section, **Then** they find a summary that explicitly names each Docker technology or concept they have encountered and recaps what they now know about it.
5. **Given** a reader is on a mobile device or skimming, **When** they scan the README headings, **Then** the headings alone communicate the learning journey.
6. **Given** a reader reaches the Kubernetes tutorial step, **When** they follow the instructions, **Then** they are directed to `k8s/README.md` for the full deployment commands — the main README does not duplicate those commands.

---

### User Story 3 - Existing operational steps remain intact (Priority: P2)

A contributor who already knows the project opens the README and can still find the start-up commands, development workflow, and architecture overview they relied on before. The tutorial structure does not bury or remove operational information.

**Why this priority**: The README restructure must not regress existing users. Operational content should be present — it may be reorganised or condensed into a "Getting Started" section, but it must remain discoverable.

**Independent Test**: Can be tested by a current contributor locating `docker compose up`, the dev container setup, and the leaderboard API endpoint description in under 60 seconds.

**Acceptance Scenarios**:

1. **Given** a contributor opens the README, **When** they search for the command to start the app, **Then** they find it within the first two minutes of reading.
2. **Given** a contributor opens the README, **When** they look for the dev container and VS Code setup instructions, **Then** they find them without visiting a separate document.

---

### Edge Cases

- What happens when a code section is too complex to annotate briefly? Comments should explain the *why* (design intent, constraint, tradeoff) not exhaustively document every line; a pointer to the relevant spec or external doc is acceptable.
- What if the README becomes very long as a tutorial? Sections may link to sub-pages or the relevant spec documents in `specs/`, but the core learning path must be completeable from the README alone without navigating away.
- What about the SBX/sandbox development workflow? Per issue 9, SBX/sandbox development is explicitly out of scope for this tutorial iteration. It should not be removed from the repo but may be placed in a separate doc or clearly marked as an optional advanced topic.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every service in `docker-compose.yml` MUST have a comment block explaining its role and why it is part of the stack, written for a Docker beginner.
- **FR-002**: Every Go source file that interacts with a Testcontainer MUST have a comment explaining the Testcontainers pattern and the reason a real container is preferred over a mock for that test.
- **FR-003**: Every Dockerfile reference and DHI image reference MUST have an accompanying comment explaining Docker Hardened Images and why the project uses them.
- **FR-004**: Go source files MUST have comments on non-obvious idiomatic patterns explaining the intent (not restating the code).
- **FR-005**: The README MUST open with a "What you will learn" section that lists all Docker technologies and concepts a reader will encounter, including Kubernetes deployment.
- **FR-006**: The README MUST be structured as a tutorial whose sections progress logically from running the app to understanding its components to deploying the app to Kubernetes.
- **FR-007**: Every tutorial step in the README MUST instruct the reader to either run a command or look at a specific file or code section — never to write code.
- **FR-008**: The README MUST end with a summary section that recaps each Docker technology or concept covered and what the reader now knows about it.
- **FR-009**: The existing operational content (start-up commands, dev container setup, architecture overview) MUST be preserved and remain discoverable in the restructured README.
- **FR-010**: The SBX/sandbox development workflow MUST NOT be included in the main tutorial path; it may be noted as an optional advanced topic or in a separate document.
- **FR-011**: The README MUST include a tutorial step on deploying the app to a local Kubernetes cluster (Docker Desktop with Kubernetes enabled) by directing the reader to `k8s/README.md` for the full deployment commands.
- **FR-012**: The README MUST NOT mention `docker compose bridge`, Helm chart generation from a compose file, or any tooling that converts compose definitions into Kubernetes manifests.
- **FR-013**: `k8s/README.md` MUST be updated to include a brief, high-level intro (2–4 sentences) explaining what Kubernetes is and what Helm does, placed before the existing commands — without going into implementation depth.

### Key Entities

- **Inline Comment**: A comment in a Go source file or Docker configuration file that explains a design choice, pattern, or technology concept to a beginner; not a restatement of the code.
- **Tutorial Section**: A named section of the README that introduces one Docker technology or concept and contains one or more "run this command" or "look at this file" steps.
- **Learning Outcome**: A statement of what a reader can explain or do after completing the tutorial; listed in the "What you will learn" section and revisited in the summary.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer unfamiliar with Docker can read the README tutorial from start to finish and correctly identify at least six distinct Docker technologies used in the project (Compose, Dockerfile/images, DHI, Testcontainers, Dev Containers, Kubernetes), without consulting external documentation.
- **SC-002**: Every service in `docker-compose.yml` and every Go file containing Testcontainer or DHI usage has at least one explanatory comment visible without scrolling past the declaration.
- **SC-003**: A current contributor can locate the `docker compose up` command and the dev container setup steps in the README in under 60 seconds.
- **SC-004**: The README tutorial contains no steps that require the reader to write or edit code.
- **SC-005**: The README summary section explicitly names and recaps every Docker technology introduced in the tutorial body, including Kubernetes.
- **SC-006**: The README contains no mention of `docker compose bridge` or Helm chart generation from a compose file.

## Assumptions

- The target reader is a developer with basic programming knowledge who is new to Docker technologies; they are not expected to know Go.
- "Docker technologies" to cover: Docker Compose, Docker images and Dockerfiles, Docker Hardened Images (DHI), Testcontainers, Dev Containers, and Kubernetes (local cluster via Docker Desktop). Compose Bridge and manifest generation from compose are explicitly excluded.
- The Kubernetes tutorial step introduces the concept and directs the reader to `k8s/README.md` for commands; it does not embed all deployment commands in the main README.
- `k8s/README.md` will receive a brief (2–4 sentence) intro explaining Kubernetes and Helm at a high level — no in-depth explanation of Kubernetes internals or Helm templating.
- The existing README content (startup commands, architecture notes, GIF, leaderboard description) is kept; only structure and annotation are changed, not substance removed.
- The SBX/sandbox development workflow is documented elsewhere in the project and will not be included in this tutorial iteration.
- Inline comments are added to source files but do not alter runtime behaviour; no refactoring of logic is required.
- The tutorial structure should work for readers viewing the README on GitHub (standard Markdown rendering).
