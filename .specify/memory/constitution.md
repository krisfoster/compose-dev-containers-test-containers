<!--
Sync Impact Report
Version change: 1.0.0 -> 1.1.0
Modified principles: (none renamed or removed)
Added sections:
  Technology Stack: new "Interim Static Hosting" carve-out permitting a non-Go static webserver
    (e.g. nginx) to serve purely static content before a Go backend exists for a given surface,
    with an explicit sunset condition (retire once a Go backend serves that content).
Removed sections: (none)
Rationale: Feature 001-host-webapp-ngrok needs to host the existing static frontend
  (frontend/game/) before any Go backend exists in this repo. The Technology Stack fixed
  "Backend: Go" with no carve-out for this pre-backend gap, which /speckit-analyze correctly
  flagged as an unamended stack change (Governance: "Adding a new runtime dependency ... requires
  a constitution amendment"). This amendment closes that gap explicitly rather than leaving it as
  a silently-justified Constitution Check PASS in a feature plan.
Templates requiring updates:
  OK  .specify/templates/plan-template.md: Constitution Check placeholder is design-compatible;
      gates populate at plan-generation time based on the principles here
  OK  .specify/templates/spec-template.md: no constitution reference; no update needed
  OK  .specify/templates/tasks-template.md: no constitution reference; no update needed
Follow-up TODOs:
  TODO(DEVCONTAINERS): DevContainer definitions arrive in a follow-up phase. Principle II
    already commits to consuming the same compose services, so the amendment when they land
    should be PATCH-level unless a boundary changes.
  TODO(INTERIM-HOSTING-SUNSET): When a Go backend is introduced that can serve frontend/game/
    directly, retire the nginx service introduced by 001-host-webapp-ngrok and remove the carve-out
    below (or narrow it if another surface still needs it). Track this at the time that backend
    feature is planned.
-->

# Whale Runner Constitution

## Core Principles

### I. Demo-First Delivery

Every feature is judged by whether it makes the live demo better. "Better" means one of: something
an attendee can see on the projected wall within two seconds of input, something that reduces the
presenter's cognitive load during a booth run, or something that increases the number of people
who can join without intervention. Code quality, test depth, and architectural elegance are
secondary to demo effect. If a change does not visibly improve the demo, it does not ship in this
milestone.

Rationale: The project exists to be shown at booths and conferences. Time spent on invisible
internals is time not spent on the moments that decide whether the demo lands.

### II. Compose-Orchestrated Reproducibility

Every runtime component (backend services, Redis, ngrok tunnel, optional LLM) MUST be defined as a
service in `docker-compose.yml`. A fresh clone plus `docker compose up` (or the equivalent
onboarding script) MUST reach a demoable local state without host-side global installs beyond
Docker Desktop and git. DevContainer definitions, when introduced in the follow-up phase, MUST
consume the same compose services rather than duplicating them.

Rationale: The team's laptops differ (uv config, Homebrew state, corporate policies). Compose is
the only reliable delivery mechanism for shared setup. Duplicating service definitions between
compose and devcontainer configs guarantees they drift.

### III. Testcontainers Over Mocks for Boundary Tests (NON-NEGOTIABLE)

Go unit tests that cross an external boundary (Redis, an HTTP client to a real service, filesystem
beyond a temp dir, any other network dependency) MUST use Testcontainers to spin up the real
dependency for the test. Mocks are permitted for pure-logic tests and for failure paths that a
real dependency cannot easily produce. Anything asserting a serialization format, a query shape,
or a pubsub interaction MUST run against a real Redis container.

Rationale: Mocked Redis passes tests that break in production the moment a command argument order
changes. Testcontainers cost a few seconds per test and buy the class of confidence a mock cannot.

### IV. Visible-in-the-Browser Definition of Done

A change is not complete until it has been observed working in the running app in a browser,
against the compose stack. Passing tests and successful builds MUST NOT be reported as "done" on
their own. The definition of done is a screenshot, screen recording, or observed behaviour in the
browser, plus a repeatable path from a fresh clone to that state (documented commands).

Rationale: Static checks miss the classes of bug that hurt on stage: dropped WebSockets,
misaligned meshes, colours that look fine on a laptop and awful on a projector, QR codes that scan
on a desk but not at three metres.

### V. Vendored-Code Hygiene (NON-NEGOTIABLE)

Every third-party asset (code, model, image, font) MUST have a current entry in the closest
`ATTRIBUTION.md` naming the author, source URL, licence, and any modifications made. Licence files
MUST be preserved alongside the vendored content. Where a licence requires visible attribution in
the running app (for example, CC BY 4.0), the credit MUST appear on a demo-visible surface before
ship. Silent copy-paste of code or assets is forbidden.

Rationale: The project already vendors an MIT-licensed game wrapper and a CC BY 4.0 whale model.
Attribution debt compounds fast once a demo is shown externally, and after-the-fact reconstruction
is unreliable.

## Technology Stack

The stack for this milestone is fixed:

- **Backend**: Go. All server-side code lives in Go modules under the repo. HTTP router,
  WebSocket hub, and Redis client are Go-native.
- **State and pubsub**: Redis. Streams, sorted sets, hashes with TTL, HyperLogLog, and pub/sub are
  all in scope. Persistence policy is decided per phase; the default is ephemeral.
- **Frontend**: three.js r160 loaded via ES-module importmap. Voxel and box-shaped geometry with
  flat shading. Docker Whale is a licensed asset (CC BY 4.0) preserved in its own style.
- **Orchestration**: `docker compose` in development, staging, and demo. DevContainers, when
  added, consume the same compose services.
- **Public URLs**: ngrok as the default tunnel. Cloudflare Tunnel is an acceptable alternative
  when a stable URL is needed for a specific event.
- **Testing**: Go standard testing plus Testcontainers-go for any test crossing a service
  boundary. Frontend behaviour is validated in a browser against the compose stack (principle IV).
- **Optional**: A local LLM served by Ollama is an optional enhancement, never a hard dependency
  of the core demo path.

Adding a new runtime dependency (a database, a message queue, a build system) is a stack change
and requires a constitution amendment following the Governance section. Swapping libraries within
the same category (for example, a different HTTP router) is not a stack change but SHOULD be noted
in the phase spec that introduces it.

**Interim Static Hosting carve-out**: A non-Go static webserver (for example, `nginx:alpine`) MAY
serve a surface's static assets in compose, but only while no Go backend yet exists to serve that
surface, and only for content requiring no server-side logic beyond file serving. This exists
because static frontends (for example, `frontend/game/`) need to be hosted before the Go backend
that will eventually own that responsibility is built. Once a Go backend is introduced for a given
surface, that surface's interim static webserver MUST be retired in the same phase that lands the
backend — this carve-out is a bridge, not a permanent parallel serving path.

## Development Workflow

Design docs (`crossy.md`, `project.md`) precede implementation. Structured phase work uses the
spec-kit skills installed via the `spec-kit` kit: `/speckit-specify` for the feature spec,
`/speckit-plan` for the implementation plan, `/speckit-tasks` to break it down,
`/speckit-implement` to build it, `/speckit-analyze` to cross-check artifacts before ship.

AI-assisted development runs inside the sbx sandbox launched by `bin/claude`. Kits under `kits/`
capture what a fresh sandbox needs. Host-side "install this globally first" steps are a smell; the
correct question is "which kit does this belong in?"

Visual assets MUST honour the voxel aesthetic: flat-shaded blocks, saturated colours over
textures, orthographic camera framing. Smooth or photorealistic content is only acceptable when
flat-shaded down to match. The Docker Whale is the deliberate exception, preserved for brand
recognition.

Commits are created only when the user asks. Pushes are never automatic. Attribution updates
(`ATTRIBUTION.md`) MUST be part of the same commit that introduces the vendored change, not a
follow-up.

## Governance

This constitution supersedes ad-hoc decisions and casual conventions. Where a code review, plan,
or spec conflicts with a principle here, the principle wins unless amended first.

Amendments to any principle or section require:

1. A Sync Impact Report at the top of this file describing what changed and why.
2. A version bump per semantic versioning: MAJOR for principle removals or contract breaks, MINOR
   for principle additions or material expansions, PATCH for clarifications and wording fixes.
3. An updated `Last Amended` date in ISO YYYY-MM-DD format.
4. Review of templates under `.specify/templates/` against the amendment; update any that carry
   constitution-derived language.

Compliance review happens at phase ship time: `/speckit-analyze` surfaces constitution deviations
across the phase's artifacts. Complexity that violates a principle MUST be justified in the phase
plan's Complexity Tracking section before it is accepted.

Runtime guidance for day-to-day work lives in `crossy.md` at the repo root.

**Version**: 1.1.0 | **Ratified**: 2026-07-06 | **Last Amended**: 2026-07-06
