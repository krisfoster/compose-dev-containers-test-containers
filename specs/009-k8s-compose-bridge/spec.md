# Feature Specification: K8s Manifest Generation via Compose Bridge

**Feature Branch**: `009-k8s-compose-bridge`

**Created**: 2026-07-07

**Status**: Draft

**Input**: User description: "Add support for using the Docker Compose Bridge to generate Kubernetes manifests from the compose file, so the app can be deployed to a k8s cluster. Add scripts to easily deploy the generated manifests to a local k8s cluster running on Docker desktop (Kind cluster). Use the industry standard tools to do this - no surprises, it should work as expected."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Generate K8s Manifests from Compose File (Priority: P1)

A developer wants to produce a set of Kubernetes manifests from the existing `docker-compose.yml` using Docker Compose Bridge, so the app can be deployed to any K8s cluster without manually authoring YAML from scratch.

**Why this priority**: Manifest generation is the foundational step. Without it, no other K8s workflow (deploy, teardown) is possible. Delivering this alone provides value because the generated manifests can be inspected, committed, and used externally.

**Independent Test**: Run the manifest generation script from a fresh clone and confirm that a directory of `.yaml` files is produced that corresponds to the services in `docker-compose.yml`.

**Acceptance Scenarios**:

1. **Given** the repo is cloned and Docker Desktop is running, **When** the developer runs the manifest generation script, **Then** a directory of Kubernetes manifests (Deployments, Services, ConfigMaps, Secrets where applicable) is produced with one manifest per service.
2. **Given** manifests have been previously generated, **When** the compose file is updated and the script is re-run, **Then** the manifests directory is refreshed to reflect the current compose state.
3. **Given** the developer runs the generation script, **When** Compose Bridge is not installed, **Then** a clear error message is shown explaining how to install it.

---

### User Story 2 - Deploy App to Local Kind Cluster (Priority: P2)

A developer wants to deploy the generated K8s manifests to a local Kind cluster running on Docker Desktop, using a single script, to verify the app works in a K8s context.

**Why this priority**: Deploying to a local cluster is the immediate proof that the generated manifests are valid and the app runs correctly under K8s orchestration.

**Independent Test**: With Kind installed and a cluster running, execute the deploy script and confirm all pods reach `Running` state and the app is accessible via the expected endpoint.

**Acceptance Scenarios**:

1. **Given** a Kind cluster exists and manifests have been generated, **When** the deploy script is run, **Then** all app services are deployed, all pods reach `Running` state, and the app responds to requests.
2. **Given** the deploy script is run, **When** the Kind cluster does not exist, **Then** the script creates the cluster before deploying, or provides a clear instruction for the user to create it first.
3. **Given** the app is deployed, **When** the developer re-runs the deploy script after a manifest change, **Then** the running deployment is updated to the new state.

---

### User Story 3 - Teardown Local K8s Deployment (Priority: P3)

A developer wants a simple way to remove the K8s deployment from the local Kind cluster and optionally delete the cluster, to free resources after testing.

**Why this priority**: Clean teardown is necessary to avoid resource leakage and enables repeated test cycles. It is lower priority because the deploy script (P2) delivers the core value.

**Independent Test**: After a successful deploy, run the teardown script and confirm all deployed resources are removed from the cluster.

**Acceptance Scenarios**:

1. **Given** the app is deployed to the Kind cluster, **When** the teardown script is run, **Then** all K8s resources created by the deploy script are deleted from the cluster.
2. **Given** a teardown with cluster deletion is requested, **When** the teardown script is run with a `--delete-cluster` flag, **Then** the Kind cluster is also deleted.

---

### Edge Cases

- What happens when `docker-compose.yml` references images not pushed to a registry (local builds only)? The manifests may reference images unreachable by the Kind cluster; the documentation must note that images need to be loaded into Kind or pushed to a registry.
- How does the system handle compose services with `build:` directives that have no `image:` tag? Compose Bridge may not handle these cleanly; the spec assumes services have a resolvable image reference.
- What happens if Kind is not installed? The deploy script must detect this and emit a clear error with installation instructions.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The repo MUST provide a script that runs Docker Compose Bridge against `docker-compose.yml` and writes the generated Kubernetes manifests to a versioned output directory (e.g. `k8s/`).
- **FR-002**: The manifest generation script MUST be idempotent: re-running it overwrites the output directory with freshly generated manifests.
- **FR-003**: The repo MUST provide a deploy script that applies the generated manifests to a local Kind cluster using `kubectl`.
- **FR-004**: The deploy script MUST verify that a Kind cluster is reachable before attempting to apply manifests, and emit a helpful error if not.
- **FR-005**: The repo MUST provide a teardown script that deletes all K8s resources created by the deploy script from the cluster.
- **FR-006**: All scripts MUST be documented in the project README (or a dedicated `k8s/README.md`) with step-by-step instructions covering prerequisites, cluster creation, manifest generation, deployment, and teardown.
- **FR-007**: The generated manifests MUST be committed to the repository so they are visible and auditable without requiring Compose Bridge to be installed.
- **FR-008**: Scripts MUST use industry-standard tooling: `compose-bridge` (or `docker compose convert`) for generation, `kind` for cluster management, and `kubectl` for resource management.
- **FR-009**: The documentation MUST note the image-loading requirement: locally-built images must be loaded into Kind via `kind load docker-image` before the deploy script is run.

### Key Entities

- **Kubernetes Manifests**: The set of YAML files produced by Compose Bridge from `docker-compose.yml`. Stored under `k8s/` in the repository. Represents the declarative desired state of the app in K8s.
- **Kind Cluster**: A local Kubernetes cluster managed by Kind, running inside Docker Desktop. The target deployment environment for local K8s testing.
- **Scripts**: Shell scripts that wrap the compose-bridge, kind, and kubectl commands. Stored under `k8s/` or `bin/` following the existing convention.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer with Docker Desktop, Kind, Compose Bridge, and kubectl installed can go from a fresh clone to a running K8s deployment of the full app stack in under 10 minutes following the documented steps.
- **SC-002**: All services defined in `docker-compose.yml` have corresponding manifests generated (one Deployment or equivalent resource per compose service).
- **SC-003**: All pods in the deployed app reach `Running` state within 2 minutes of the deploy script completing on a standard developer laptop.
- **SC-004**: The teardown script removes all deployed resources and leaves the Kind cluster clean (zero app-related pods or services remaining) within 60 seconds.
- **SC-005**: A developer unfamiliar with Compose Bridge can follow the documentation without consulting external sources for the core flow.

## Assumptions

- Docker Desktop with Kind support is the target local K8s environment; other local K8s distributions (Minikube, k3d) are out of scope for this feature.
- Compose Bridge (or the `docker compose convert` equivalent) is available as a CLI tool that the developer can install; the scripts do not bundle or vendor it.
- All compose services use a published or locally-buildable image with a tag suitable for loading into Kind; services using only `build:` with no `image:` tag are documented as unsupported.
- The ngrok service in `docker-compose.yml` is expected to be excluded or adapted in the K8s manifests, as it is not suitable for K8s deployment without modification. This adaptation is documented but not automated.
- Persistent storage (Redis) will use a simple `emptyDir` or `hostPath` volume in the Kind deployment; production-grade persistence is explicitly out of scope.
- No Ingress controller setup is required for this feature; service access via `kubectl port-forward` is the supported access pattern in the local Kind cluster.
- The generated manifests are committed to the repo and kept up to date manually when the compose file changes; there is no automated CI sync for this feature.
