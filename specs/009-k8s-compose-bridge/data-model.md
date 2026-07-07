# Data Model: K8s Manifest Entities

**Feature**: 009-k8s-compose-bridge | **Date**: 2026-07-07

Describes the Kubernetes resource entities produced by Compose Bridge and the deploy scripts. All resources live in the `default` namespace for simplicity.

---

## Deployment: redis

| Field | Value |
|-------|-------|
| Name | `redis` |
| Image | `dhi.io/redis:8-alpine` |
| Replicas | 1 |
| Command | `["redis-server", "/etc/redis/redis.conf", "--protected-mode", "no"]` |
| Volume | `emptyDir` mounted at `/data` (Redis default data dir) |
| imagePullPolicy | `IfNotPresent` (pre-loaded into Kind by deploy.sh) |

**Why emptyDir**: Demo context only; Redis data is ephemeral. No PersistentVolumeClaim needed.

**Why command override**: The DHI Redis image ships with `protected-mode on` in its hardened config, which rejects connections from non-loopback clients. The override matches the compose config behaviour.

---

## Service: redis (ClusterIP)

| Field | Value |
|-------|-------|
| Name | `redis` |
| Type | `ClusterIP` |
| Port | 6379 → 6379 (TCP) |
| Selector | `app: redis` |

The app pod connects to Redis via `redis:6379` (same hostname as in compose), so `REDIS_ADDR=redis:6379` works without change.

---

## Deployment: app

| Field | Value |
|-------|-------|
| Name | `app` |
| Image | `whale-runner:k8s-local` |
| Replicas | 1 |
| imagePullPolicy | `Never` (locally built and loaded into Kind) |

**Environment variables**:

| Variable | Source | Value |
|----------|--------|-------|
| `REDIS_ADDR` | literal | `redis:6379` |
| `GRANT_COOKIE_SECRET` | Secret `whale-runner-secrets` | key `GRANT_COOKIE_SECRET` |
| `LEADERBOARD_API_SECRET` | Secret `whale-runner-secrets` | key `LEADERBOARD_API_SECRET` |
| `FRONTEND_DIR` | literal | `/frontend` |
| `APP_WEB_PORT` | literal | `8080` |
| `APP_GATED_PORT` | literal | `8081` |
| `NGROK_API_URL` | literal | `""` (empty — ngrok not deployed in K8s) |
| `QR_WINDOW_TTL` | literal | `15m` |

**Volumes**: None. The frontend is embedded in `whale-runner:k8s-local` at `/frontend` at image build time.

---

## Service: app (NodePort)

| Field | Value |
|-------|-------|
| Name | `app` |
| Type | `NodePort` (or `ClusterIP` — port-forward works with either) |
| Port | 8080 → 8080 (TCP) |
| Selector | `app: app` |

NodePort type is preferred over ClusterIP so that Kind can expose the service on the node IP if needed, but `kubectl port-forward svc/app 8080:8080` is the documented access method.

---

## Secret: whale-runner-secrets

| Field | Value |
|-------|-------|
| Name | `whale-runner-secrets` |
| Type | `Opaque` |
| Keys | `GRANT_COOKIE_SECRET`, `LEADERBOARD_API_SECRET` |

**Created by**: `deploy.sh` reads values from environment variables and creates/updates the Secret via `kubectl apply`. The file `k8s/manifests/app-secrets.yaml.example` documents the structure with placeholder values — it is never applied directly.

---

## Entity Relationships

```
Secret: whale-runner-secrets
    └── envFrom → Deployment: app
                      └── connects to → Service: redis → Deployment: redis
Service: app
    └── port-forwarded → developer browser (localhost:8080)
```
