# Implementation Plan: K8s Manifest Generation via Compose Bridge

**Branch**: `009-k8s-compose-bridge` | **Date**: 2026-07-07 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/009-k8s-compose-bridge/spec.md`

## Summary

Add K8s deployment capability for the Whale Runner app by using Docker Compose Bridge to generate Kubernetes manifests from `docker-compose.yml`, combined with Kind for local cluster management. Introduces a K8s-specific Docker image build that embeds the frontend assets (removing the bind-mount dependency), shell scripts for generate/deploy/teardown, and documentation. The compose-based development workflow is unchanged.

## Technical Context

**Language/Version**: Shell (bash scripts); YAML (Kubernetes manifests)

**Primary Dependencies**: `compose-bridge` (Docker CLI plugin), `kind`, `kubectl`

**Storage**: Redis (`emptyDir` volume in Kind — ephemeral; sufficient for local demo)

**Testing**: Manual validation via `kubectl` and browser; no new Go tests required

**Target Platform**: Kind cluster on Docker Desktop (local Kubernetes)

**Project Type**: Tooling / scripts + deployment artifacts

**Performance Goals**: All pods reach `Running` state within 2 minutes of deploy script completion

**Constraints**: No Ingress controller; browser access via `kubectl port-forward`; no persistent storage

**Scale/Scope**: Single-developer local demo environment

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Demo-First) | ✅ PASS | K8s deployment is a demo capability — shows compose→k8s portability at conferences |
| Principle II (Compose-Orchestrated Reproducibility) | ✅ PASS | Compose remains the primary development and demo path; K8s is an ADDITIONAL deployment target, not a replacement |
| Principle III (Testcontainers for boundary tests) | ✅ PASS | No new Go tests; no new mocks introduced |
| Principle IV (Visible-in-Browser DoD) | ✅ PASS | Quickstart requires browser verification after `kubectl port-forward` |
| Principle V (Vendored-Code Hygiene) | ✅ PASS | No new vendored assets |
| Technology Stack (Orchestration) | ⚠️ AMENDMENT REQUIRED | Constitution currently lists only "docker compose in development, staging, and demo". Adding K8s/Kind as an optional deployment target requires a MINOR amendment to the Technology Stack section before this feature ships. The amendment should add a carve-out: "Kind is an optional local Kubernetes target for demonstrating compose→k8s portability; it does not replace compose for development or primary demo use." |

**Post-Phase 1 re-check**: Amendment required before ship. All other principles hold. The `k8s/Dockerfile` duplication is justified in Complexity Tracking below.

## Project Structure

### Documentation (this feature)

```text
specs/009-k8s-compose-bridge/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── scripts-interface.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
k8s/
├── manifests/              # Generated manifests — committed to repo
│   ├── redis-deployment.yaml
│   ├── redis-service.yaml
│   ├── app-deployment.yaml
│   ├── app-service.yaml
│   └── app-secrets.yaml.example   # template only — shows required keys, not real values
├── Dockerfile              # K8s-specific: extends app build to embed frontend/game
├── generate.sh             # Runs compose-bridge; writes to k8s/manifests/
├── deploy.sh               # Builds image, loads into Kind, applies manifests
├── teardown.sh             # Removes K8s resources; --delete-cluster to remove Kind cluster
└── README.md               # Step-by-step prerequisites, usage, and troubleshooting
```

No changes to `app/`, `bin/`, or `docker-compose.yml`.

**Structure Decision**: All K8s artifacts live under `k8s/` to keep them clearly separated from the primary compose-based workflow. Scripts in `k8s/` rather than `bin/` because they are K8s-specific and not part of the general developer onboarding path.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| `k8s/Dockerfile` duplicates build logic from `app/Dockerfile` | Kind cannot use host bind mounts; the frontend (`frontend/game/`) must be embedded in the image for K8s deployment | Modifying `app/Dockerfile` to conditionally include the frontend would couple K8s concerns into the primary development Dockerfile, risking breakage of the compose volume-mount flow and adding build-arg complexity to every developer's daily workflow |
