# Research: K8s Manifest Generation via Compose Bridge

**Feature**: 009-k8s-compose-bridge | **Date**: 2026-07-07

Key decisions made during planning, with rationale and alternatives evaluated.

---

## Decision 1: Manifest Generation Tool — Docker Compose Bridge

**Decision**: Use `compose-bridge convert` to generate Kubernetes manifests from `docker-compose.yml`.

**Rationale**: The issue explicitly requests Docker Compose Bridge by name. It is Docker's native Compose→K8s conversion tool, designed to produce idiomatic K8s manifests from compose service definitions.

**Alternatives considered**:
- `kompose`: The most widely adopted open-source Compose→K8s tool. Well-maintained, battle-tested, and produces clean manifests. Rejected because the issue specifically names Compose Bridge; using kompose would be a deviation from the stated requirement.
- Manual authoring: Error-prone, does not demonstrate the compose-bridge workflow, and diverges from the compose file when it changes.

**Installation**: Built into Docker Desktop — no separate installation required. Available as `docker compose bridge` (a Docker CLI plugin shipped with Docker Desktop).

**CLI invocation**: `docker compose bridge convert -o k8s/manifests/` (reads `docker-compose.yml` from the current directory automatically). No `-f` flag needed when run from the repo root.

---

## Decision 2: Frontend Asset Embedding in K8s Image

**Decision**: Create `k8s/Dockerfile` that builds from the repo root context and copies `frontend/game/` into the image at `/frontend`, replacing the runtime bind-mount.

**Rationale**: Kubernetes (and Kind specifically) does not support host bind mounts the way Docker Compose does. The compose config mounts `./frontend/game:/frontend:ro` at runtime, but in K8s the container filesystem must be self-contained. The frontend assets must be present inside the image.

**Image tag**: `whale-runner:k8s-local` — built locally and loaded into Kind via `kind load docker-image whale-runner:k8s-local --name whale-runner`. Using `imagePullPolicy: Never` in the Deployment ensures Kind uses the locally loaded image.

**Alternatives considered**:
- **ConfigMap**: Impractical — game assets include JS files that may exceed ConfigMap size limits, and mounting many files from a ConfigMap adds YAML volume complexity.
- **emptyDir + initContainer**: An initContainer could copy assets from a separate image. Adds deployment complexity (two containers per pod, sequencing logic) for no benefit in a local demo context.
- **Modify `app/Dockerfile`**: Could add a build arg to optionally include the frontend. Rejected because it couples K8s concerns into the primary development Dockerfile, adds cognitive overhead for every developer running compose, and risks subtle breakage if the build arg path diverges from the compose volume mount path.

---

## Decision 3: ngrok Service Exclusion

**Decision**: Exclude the `ngrok` service from the generated K8s manifests entirely.

**Rationale**: The ngrok service uses `profiles: public` in compose, meaning it is already treated as optional. It requires a live `NGROK_AUTHTOKEN`, is not part of the core app flow (game + leaderboard), and has no equivalent K8s deployment model without additional Ingress/tunnel configuration well beyond the scope of this feature.

**Implementation**: The `generate.sh` script passes compose-bridge options to suppress the ngrok service, or post-processes the output directory to remove any ngrok-related manifests. The README notes that external access in K8s is handled via `kubectl port-forward`, not ngrok.

**App env var**: `NGROK_API_URL` is set to an empty string in the K8s app Deployment. The app's ngrok integration is guarded so an empty URL disables the feature gracefully without crashing.

---

## Decision 4: Redis in Kind

**Decision**: Deploy Redis using the DHI image (`dhi.io/redis:8-alpine`) with an `emptyDir` volume for storage.

**Rationale**: Using the same DHI Redis image as compose maintains consistency. `emptyDir` is sufficient for a local demo — data is ephemeral in demo context regardless. Introducing a PersistentVolumeClaim for local demo would add Kind storage class complexity with no user value.

**Redis command override**: The compose config overrides the Redis command to add `--protected-mode no`. This override must be preserved in the K8s Deployment `command` field: `["redis-server", "/etc/redis/redis.conf", "--protected-mode", "no"]`. Without this, Redis refuses connections from the app pod (same issue as in compose before the override was added).

**Image availability in Kind**: Kind pulls images at pod scheduling time. `dhi.io/redis:8-alpine` is a public DHI image and should pull without credentials. As a reliability measure, `deploy.sh` pre-pulls the image locally (`docker pull dhi.io/redis:8-alpine`) and loads it into Kind (`kind load docker-image dhi.io/redis:8-alpine`), eliminating pull latency and any intermittent registry connectivity issues during demos.

---

## Decision 5: Secrets Handling

**Decision**: Store `GRANT_COOKIE_SECRET` and `LEADERBOARD_API_SECRET` in a K8s Secret named `whale-runner-secrets`. The deploy script creates it from environment variables.

**Rationale**: Secrets must not be committed to the repo. The deploy script reads them from the environment (or prompts if unset) and creates the Secret via `kubectl create secret generic whale-runner-secrets --from-literal=GRANT_COOKIE_SECRET=... --from-literal=LEADERBOARD_API_SECRET=... --dry-run=client -o yaml | kubectl apply -f -` (idempotent — safe to re-run).

**app-secrets.yaml.example**: Committed to `k8s/manifests/` as a reference only — shows the Secret structure with placeholder values. Never applied directly; it is documentation.

---

## Decision 6: Access Pattern

**Decision**: Browser access via `kubectl port-forward svc/app 8080:8080`. No Ingress controller.

**Rationale**: An Ingress controller (nginx-ingress, Traefik, etc.) requires additional Kind cluster configuration and is overkill for a single local developer. Port-forward is universally available with any kubectl installation, requires no cluster-level setup, and is the standard approach for local K8s service access. The deploy script prints the exact command after applying manifests, so the developer does not need to remember it.

**Leaderboard**: Accessible at `http://localhost:8080/leaderboard` via the same port-forward — same host and port as the game.
