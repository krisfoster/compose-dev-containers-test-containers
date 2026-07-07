# Implementation Plan: Docker Hardened Images (DHI) Migration

**Branch**: `005-dhi-image-migration` | **Date**: 2026-07-07 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/005-dhi-image-migration/spec.md`

## Summary

Migrate every container image the project uses that has a hardened equivalent to Docker Hardened
Images (DHI), pulled free from `dhi.io` after `docker login dhi.io`. Concretely: the app's
multi-stage `app/Dockerfile` build stage moves from `golang:1.25-alpine` to the DHI Go **dev**
image, its runtime stage moves from `alpine:3.20` to the DHI `static` (Alpine) image (the app is a
`CGO_ENABLED=0` static binary), and Redis moves from `redis:7-alpine` to the DHI `redis:8-alpine`
image in both `docker-compose.yml` and the two Testcontainers-backed Go tests. There is no hardened
Redis 7.x (hardened Redis starts at 8.x on Alpine), so this is a documented 7→8 major bump —
anticipated by the spec's version-parity assumption. `ngrok/ngrok:3` has no DHI
equivalent and is exempted (demo-only, `public` profile). An image inventory records every image
with a migrated/exempt status, `project.md`'s image table is updated to match, and the README gains
a short note on the one new prerequisite (`docker login dhi.io` with a free Docker account) plus a
reinforced note that ngrok is a demo-only exception. Behaviour, ports, and the browser demo are
unchanged.

## Technical Context

**Language/Version**: Go 1.25.0 (module `crossywhale/app`); frontend is static three.js assets.

**Primary Dependencies**: `docker compose`; images — DHI `golang:1.25-alpine-dev` (build stage,
root + apk + shell), DHI `static:*-alpine` (runtime), DHI `redis:8-alpine` (state/pubsub);
`redis/go-redis/v9`; Testcontainers-go redis module v0.43.0.

**Storage**: Redis (ephemeral by default) — game state, gate windows, leaderboard.

**Testing**: Go standard testing + Testcontainers-go (boundary tests spin up a real Redis
container, constitution Principle III).

**Target Platform**: Linux containers via `docker compose`, `linux/amd64` + `linux/arm64` (both DHI
images support both platforms). App serves two HTTP listeners.

**Project Type**: Containerized web service (Go backend + static frontend) orchestrated by Compose.

**Performance Goals**: No regression — same gameplay/QR/leaderboard behaviour; image build and cold
`compose up` should remain comparable (DHI images are small: `static` ≈ 0.23 MB compressed).

**Constraints**:
- DHI non-dev runtime images run as **nonroot (uid 65532)**, ship **no shell / no package manager**,
  and DHI guidance recommends binding **ports ≥ 1024**. The app already listens on `:8080`/`:8081`
  and is a static binary, so it fits without code changes.
- Pulling DHI Community images requires a free Docker account and `docker login dhi.io`; images pull
  directly from `dhi.io` (no paid subscription, no org-namespace mirroring, no build-from-source).
- No new runtime service is introduced (image-source swap only) — not a constitution stack change.

**Scale/Scope**: 3 image references to migrate (golang, alpine→static, redis) across `app/Dockerfile`
(2 stages), `docker-compose.yml` (1 service), and 2 Go test files; 1 exemption (ngrok); docs:
inventory (new), `project.md` table, README.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Status |
|-----------|------|--------|
| I. Demo-First Delivery | Migration must not degrade the demo; value is a hardened, lower-CVE stack behind the same visible behaviour. | **PASS** — invisible-internals change justified by supply-chain security; validated in-browser (Principle IV) so it cannot silently break the demo. |
| II. Compose-Orchestrated Reproducibility | Fresh clone + `docker compose up` reaches a demoable state with no host installs beyond Docker Desktop + git. | **PASS (documented deviation)** — adds one new *authentication* step (`docker login dhi.io`, free account), not a global install. Documented in README + quickstart; the base-image swap itself stays inside compose. |
| III. Testcontainers Over Mocks (NON-NEGOTIABLE) | Boundary tests run against a real Redis container. | **PASS** — the Testcontainers Redis image string moves to the DHI redis image; no test is weakened or mocked. Risk (DHI redis wait-strategy/entrypoint compatibility with the redis module) is a Phase 0 research item. |
| IV. Visible-in-the-Browser Definition of Done | Done = observed working in a browser against the compose stack. | **PASS** — quickstart.md drives the game, QR/host, and leaderboard in a browser on the migrated stack. |
| V. Vendored-Code Hygiene (NON-NEGOTIABLE) | Third-party assets have ATTRIBUTION entries + licences. | **N/A** — base container images are dependencies, not vendored source/asset copies; no `ATTRIBUTION.md` entry required. No vendored code/assets are added. |

**Technology Stack note**: Swapping image sources within the same category (golang→DHI golang,
alpine→DHI static, redis→DHI redis) is explicitly "not a stack change" per the constitution; it is
noted here in the phase plan as required. No amendment needed.

**Gate result**: PASS. No unjustified violations → Complexity Tracking not required.

## Project Structure

### Documentation (this feature)

```text
specs/005-dhi-image-migration/
├── plan.md              # This file (/speckit-plan output)
├── spec.md              # Feature spec (+ clarifications)
├── research.md          # Phase 0 output — exact DHI tags + compatibility decisions
├── data-model.md        # Phase 1 output — Image Inventory Entry schema
├── quickstart.md        # Phase 1 output — in-browser + test validation guide
├── contracts/
│   └── image-inventory.md   # The migrated/exempt image inventory (the FR-008 deliverable)
└── checklists/
    └── requirements.md  # Spec quality checklist (from /speckit-specify)
```

### Source Code (repository root)

```text
app/
├── Dockerfile           # EDIT: build stage → DHI golang dev; runtime stage → DHI static
├── go.mod               # unchanged
├── main.go              # unchanged (listens :8080/:8081, static binary)
└── internal/
    ├── leaderboard/store_test.go   # EDIT: Testcontainers image "redis:7-alpine" → DHI redis
    └── gate/window_test.go         # EDIT: Testcontainers image "redis:7-alpine" → DHI redis

docker-compose.yml       # EDIT: redis service image → DHI redis; ngrok unchanged (exempt)
project.md               # EDIT: image table reflects DHI sources
README.md                # EDIT: add `docker login dhi.io` prerequisite; reinforce ngrok exception
```

**Structure Decision**: Existing single-service-plus-frontend layout is unchanged. This feature only
edits image references (Dockerfile stages, one compose service, two test files) and documentation;
no new source directories or runtime components are introduced.

## Complexity Tracking

> No constitution violations requiring justification — this section is intentionally empty.
