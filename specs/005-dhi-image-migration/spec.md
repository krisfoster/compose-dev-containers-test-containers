# Feature Specification: Docker Hardened Images (DHI) Migration

**Feature Branch**: `005-dhi-image-migration`

**Created**: 2026-07-07

**Status**: Draft

**Input**: User description: "specify issue 4 from issues.md — Docker Hardened Images (DHI). Migrate all container images used by the app to DHI."

## Clarifications

### Session 2026-07-07

- Q: Access path for pulling hardened images? → A: Option A — require a free Docker account and `docker login dhi.io`, then pull DHI Community images directly from `dhi.io`. No paid subscription, no org-namespace mirroring, and no build-from-source fallback in scope.
- Q: Should onboarding docs also cover ngrok access? → A: Yes — add short README instructions for obtaining an ngrok account/authtoken, noting ngrok is a demo-only exception to the migration.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Application image runs on a hardened base (Priority: P1)

The whale-runner application container is built and run from Docker Hardened Images: the Go
toolchain that compiles the binary and the minimal runtime base that ships it both come from the
hardened catalog instead of the public `golang` and `alpine` images. After the migration, `docker
compose up` builds the app and the game plays in the browser exactly as before.

**Why this priority**: The application image is the project's own deliverable and its largest,
most-changed attack surface. Hardening it delivers the bulk of the security value and must not
break the demo. If only this story ships, the project already runs its own code on a hardened,
lower-CVE base — a viable, valuable slice on its own.

**Independent Test**: Build the app image and run the full stack via compose; confirm the game
loads and is playable in a browser (per the project's Visible-in-the-Browser definition of done),
and that a vulnerability scan of the app image reports fewer known CVEs than the pre-migration
image.

**Acceptance Scenarios**:

1. **Given** a fresh clone with DHI access configured, **When** a contributor runs the documented
   startup command, **Then** the app image builds from hardened base images and the game is
   playable in the browser with no behavioural change.
2. **Given** the migrated app image, **When** its runtime container starts, **Then** the app
   process runs as a non-root user, binds its unprivileged web/gated ports, and serves the frontend
   without requiring a shell or package manager in the runtime image.
3. **Given** the migrated app image, **When** it is scanned for known vulnerabilities, **Then** the
   reported CVE count is lower than the previous public-base image.

---

### User Story 2 - Redis runs on a hardened image everywhere it is used (Priority: P2)

The Redis dependency is sourced from the hardened catalog in every place it appears: the `redis`
service in compose and the Redis image spun up by the Testcontainers-backed Go tests. Runtime and
test environments use the same hardened image so behaviour stays consistent.

**Why this priority**: Redis holds all game state, pub/sub, gate windows, and the leaderboard, so
it is on the core demo path. Test/runtime image parity is required by the project's testing
principle (boundary tests run against a real Redis), so the test image must move with the runtime
image rather than lagging behind.

**Independent Test**: Start the stack with the hardened Redis image and confirm the app connects
and gameplay/leaderboard work in the browser; run the Go test suite and confirm the
Testcontainers-based tests pass against the hardened Redis image.

**Acceptance Scenarios**:

1. **Given** the compose stack, **When** it is brought up, **Then** the Redis service runs from the
   hardened image and the app reads and writes state against it with no behavioural change.
2. **Given** the Go test suite, **When** boundary tests execute, **Then** Testcontainers launches
   the hardened Redis image and the tests pass with equivalent behaviour to the previous image.
3. **Given** the migrated Redis image, **When** it is scanned for known vulnerabilities, **Then**
   the reported CVE count is lower than the previous public-base image.

---

### User Story 3 - Every image is accounted for, with exemptions documented (Priority: P3)

A single inventory records every container image the project uses, its migration status (migrated
to hardened, or exempt), and the reason for any exemption. Images that have no hardened equivalent
(for example, the ngrok tunnel agent) are explicitly listed as exempt with a rationale, so "migrate
all images" has a verifiable, honest answer rather than a silent gap.

**Why this priority**: Completeness and auditability. Without an inventory, an unmigrated or
newly-added image can slip through unnoticed and the claim "all images are hardened" cannot be
checked. This closes the loop but depends on the actual migrations (P1, P2) being done first.

**Independent Test**: Open the inventory and confirm every image referenced anywhere in the repo
(Dockerfile stages, compose services, test code, and docs) appears exactly once with a status of
migrated or exempt-with-reason, and cross-check it against a repo-wide search for image references.

**Acceptance Scenarios**:

1. **Given** the repository after migration, **When** the inventory is compared against a
   repo-wide search for image references, **Then** every referenced image appears in the inventory
   with a migrated-or-exempt status.
2. **Given** an image with no hardened equivalent (ngrok), **When** the inventory is reviewed,
   **Then** it is listed as exempt with a stated reason and, where relevant, a note that it is off
   the core local demo path.
3. **Given** the project documentation that lists images (the image table in `project.md`), **When**
   it is reviewed after migration, **Then** it reflects the hardened sources and matches the
   inventory.

---

### Edge Cases

- **Image with no hardened equivalent**: `ngrok/ngrok` is not in the hardened catalog. It is
  exempted, documented, and — because it only runs under the optional `public` tunnel profile — does
  not block the core local demo.
- **No shell / no package manager in runtime image**: Hardened runtime images are minimal and
  typically ship without a shell or package manager. Any entrypoint, healthcheck, or debugging step
  that assumes a shell or installs packages at runtime must be reworked or removed.
- **Non-root by default**: Hardened images run as a non-root user. Anything that assumed root
  (writing outside designated writable paths, binding privileged ports below 1024) must be
  compatible; the app's unprivileged ports (web and gated) must continue to work.
- **Contributor or CI not logged in to `dhi.io`**: Pulling hardened Community images requires a
  (free) Docker account and `docker login dhi.io`. A contributor or CI runner that has not
  authenticated must get a clear failure and documented guidance (create a free account, then
  `docker login dhi.io`), rather than a confusing pull error.
- **Tag/version differences in the hardened catalog**: The hardened catalog may not expose an
  identically-named tag (for example, exact Go or Redis tag strings differ). The migration must
  select an equivalent version on the same major line and document any version movement.
- **Test/runtime drift**: If the compose Redis image is migrated but the Testcontainers image is
  not (or vice versa), tests and runtime diverge. Both must move together.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every container image the project uses that has an available hardened-catalog
  equivalent MUST be sourced from the hardened catalog rather than from public registries.
- **FR-002**: The application's build stage MUST use the hardened Go toolchain image on the same Go
  major/minor line currently in use.
- **FR-003**: The application's runtime stage MUST use a hardened minimal/base image suitable for
  running the compiled static binary, and the resulting container MUST run the app as a non-root
  user.
- **FR-004**: The Redis service defined in the compose stack MUST use the hardened Redis image on an
  equivalent version line to the one currently in use.
- **FR-005**: Every test that starts a Redis container via Testcontainers MUST use the same hardened
  Redis image as the runtime, preserving runtime/test parity.
- **FR-006**: The migrated application MUST run correctly under hardened-image constraints —
  non-root execution, no reliance on a runtime shell or package manager, and correct binding of the
  existing unprivileged web and gated ports.
- **FR-007**: A fresh clone MUST still reach the demoable browser state via the documented startup
  path. The only new prerequisite is a free Docker account plus `docker login dhi.io` to pull DHI
  Community images directly from `dhi.io`; this MUST be documented. No paid subscription,
  org-namespace mirroring, or build-from-source step is required.
- **FR-008**: The project MUST maintain an inventory listing every container image it references,
  each marked as migrated-to-hardened or exempt, with a stated reason for every exemption.
- **FR-009**: Any image with no hardened-catalog equivalent (for example, the ngrok tunnel agent)
  MUST be explicitly exempted in the inventory with its rationale, and MUST NOT be silently left
  unaddressed.
- **FR-010**: Migrated images MUST preserve equivalent functionality and version parity (same major
  line) with the images they replace; any intentional version change MUST be documented.
- **FR-011**: Project documentation that lists container images (including the image table in
  `project.md`) MUST be updated to reflect the hardened sources and MUST agree with the inventory.
- **FR-012**: The README MUST include short setup instructions covering both new access
  prerequisites: (a) create a free Docker account and run `docker login dhi.io` to pull the
  hardened images, and (b) obtain an ngrok account and authtoken for the optional public-tunnel
  demo — explicitly noting that ngrok is a demo-only exception to the hardened-image migration.

### Key Entities *(include if feature involves data)*

- **Image Inventory Entry**: One record per container image the project uses. Attributes: logical
  name/role (e.g. app build, app runtime, redis, ngrok), current source reference, hardened status
  (migrated | exempt), version/line, and — for exemptions — the reason and whether it sits on the
  core demo path.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of container images that have an available hardened-catalog equivalent are
  sourced from the hardened catalog after the migration.
- **SC-002**: 100% of container images referenced anywhere in the repository (Dockerfile stages,
  compose services, test code, docs) appear in the inventory with a migrated-or-exempt status and a
  reason for each exemption.
- **SC-003**: A fresh clone reaches the same playable-in-the-browser demo state via the documented
  startup path after migration, with no behavioural regression in gameplay, gating, or leaderboard.
- **SC-004**: The existing automated test suite passes with equivalent behaviour against the
  hardened Redis image, with no test disabled or weakened to accommodate the change.
- **SC-005**: The count of known vulnerabilities (CVEs) reported by a scan of the migrated app and
  Redis images is lower than the count for the public images they replace.
- **SC-006**: Exactly one image (the ngrok tunnel agent) is recorded as exempt for lack of a
  hardened equivalent, and it is confirmed to be off the core local demo path.
- **SC-007**: A new contributor, following only the README, can reach the playable demo state
  without external help — the README covers creating a free Docker account, `docker login dhi.io`,
  and (for the optional public tunnel) obtaining an ngrok account/authtoken.

## Assumptions

- Hardened-catalog equivalents exist and are usable for the Go toolchain, a minimal Go-binary
  runtime base, and Redis (confirmed present in the catalog); `ngrok/ngrok` has no hardened
  equivalent and is therefore exempted.
- DHI Community images are free (Apache 2.0) and pulled directly from `dhi.io`; the only new
  host-side prerequisite beyond Docker and git is a free Docker account and `docker login dhi.io`.
  No paid subscription, org-namespace mirroring, or build-from-source path is in scope (those are
  Select/Enterprise-tier options and are explicitly excluded).
- The application is built as a static binary (CGO disabled), so a distroless/static or minimal
  hardened runtime base is viable without a full OS userland.
- The app listens only on unprivileged ports (its web and gated ports), so the hardened non-root
  default does not require privileged-port workarounds.
- Version parity is maintained on the current major lines (Go 1.25.x, Redis 7.x) unless a specific
  upgrade is chosen and documented during planning.
- ngrok runs only under the optional `public` tunnel profile and is not part of the core local demo
  path, so exempting it does not compromise the core demo; its account/authtoken setup is a
  demo-only concern, documented in the README as an exception.
- No new runtime service or dependency is introduced by this migration (it swaps image sources, not
  the stack), so no constitution stack amendment is required; the base-image swaps SHOULD be noted
  in the plan.
