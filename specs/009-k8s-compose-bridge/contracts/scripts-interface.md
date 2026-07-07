# Scripts Interface Contract

**Feature**: 009-k8s-compose-bridge | **Date**: 2026-07-07

Defines the external interface for the three K8s workflow scripts in `k8s/`. All scripts are executed from the repository root.

---

## generate.sh

**Purpose**: Run Docker Compose Bridge against `docker-compose.yml` and write Kubernetes manifests to `k8s/manifests/`.

**Usage**:
```
./k8s/generate.sh
```

**Prerequisites**:
- `compose-bridge` command available on PATH (Docker CLI plugin or standalone binary)
- Executed from repository root

**Inputs**:
- `docker-compose.yml` (repo root) — read-only

**Outputs**:
- `k8s/manifests/*.yaml` — created or overwritten on each run
- ngrok-related manifests excluded from output

**Exit codes**:
| Code | Meaning |
|------|---------|
| 0 | Manifests generated successfully |
| 1 | `compose-bridge` not found on PATH; error message includes install instructions |
| 2 | `compose-bridge` invocation failed; stderr from compose-bridge forwarded |

**Idempotent**: Yes — re-running overwrites existing manifests.

**Notes**: The ngrok service is excluded by passing the appropriate flags to compose-bridge or by post-processing the output directory to remove any ngrok manifests.

---

## deploy.sh

**Purpose**: Build the K8s app image, load all images into Kind, apply manifests, and wait for pods to be ready.

**Usage**:
```
./k8s/deploy.sh [--cluster-name NAME]
```

**Options**:
| Option | Default | Description |
|--------|---------|-------------|
| `--cluster-name NAME` | `whale-runner` | Name of the Kind cluster to create or use |

**Prerequisites**:
- `kind` installed and reachable on PATH
- `kubectl` installed and reachable on PATH
- Docker daemon running
- `k8s/manifests/` directory exists and contains manifests (run `generate.sh` first)
- `GRANT_COOKIE_SECRET` environment variable set (non-empty)
- `LEADERBOARD_API_SECRET` environment variable set (non-empty)

**Environment variables** (read at runtime):
| Variable | Required | Description |
|----------|----------|-------------|
| `GRANT_COOKIE_SECRET` | Yes | Deployed to K8s Secret; script exits with error if unset |
| `LEADERBOARD_API_SECRET` | Yes | Deployed to K8s Secret; script exits with error if unset |

**Actions** (in order):
1. Validate prerequisites (`kind`, `kubectl`, Docker daemon, manifests directory)
2. Create Kind cluster `--cluster-name` if it does not already exist
3. Build `whale-runner:k8s-local` from `app/Dockerfile` with repo root as build context
4. `kind load docker-image whale-runner:k8s-local --name <cluster-name>`
5. `docker pull dhi.io/redis:8-alpine` and `kind load docker-image dhi.io/redis:8-alpine --name <cluster-name>`
6. Create/update K8s Secret `whale-runner-secrets` from environment variables (idempotent via `kubectl apply`)
7. `kubectl apply -f k8s/manifests/`
8. `kubectl rollout status deployment/redis` and `kubectl rollout status deployment/app` (waits up to 2 minutes each)
9. Print the `kubectl port-forward` command for browser access

**Exit codes**:
| Code | Meaning |
|------|---------|
| 0 | All pods running; port-forward command printed |
| 1 | Prerequisite missing (kind/kubectl not found, docker not running, manifests missing, required env var unset) |
| 2 | Deploy failed (image build error, kind load error, kubectl apply error, rollout timeout) |

**Idempotent**: Yes — re-running updates the cluster to the current manifest state.

---

## teardown.sh

**Purpose**: Remove all Kubernetes resources deployed by `deploy.sh` from the Kind cluster.

**Usage**:
```
./k8s/teardown.sh [--delete-cluster] [--cluster-name NAME]
```

**Options**:
| Option | Default | Description |
|--------|---------|-------------|
| `--delete-cluster` | off | Also delete the Kind cluster after removing resources |
| `--cluster-name NAME` | `whale-runner` | Name of the Kind cluster to target |

**Prerequisites**:
- `kind` installed and reachable on PATH
- `kubectl` installed and reachable on PATH
- Executed from repository root

**Actions** (in order):
1. `kubectl delete -f k8s/manifests/` (ignores not-found errors — safe if partially deployed)
2. `kubectl delete secret whale-runner-secrets --ignore-not-found`
3. If `--delete-cluster`: `kind delete cluster --name <cluster-name>`

**Exit codes**:
| Code | Meaning |
|------|---------|
| 0 | Resources removed (or were already absent) |
| 1 | Kind cluster not found and `--delete-cluster` was requested with no matching cluster |
| 2 | kubectl/kind command failed unexpectedly |

**Idempotent**: Yes — safe to run when nothing is deployed.
