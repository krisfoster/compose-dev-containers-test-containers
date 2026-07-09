# Contract: Compose Service Port Topology

**Feature**: 020-secure-service-port-exposure | **Date**: 2026-07-09

This document defines the authoritative port exposure contract for the Whale Runner compose stack after this feature is applied.

## Host-Published Ports (reachable from outside Docker)

| Service | Host Port | Container Port | Purpose |
|---------|-----------|----------------|---------|
| `nginx` | `${NGINX_PORT:-80}` | 80 | Single public ingress — all browser and ngrok traffic |
| `app` | `${WEB_PORT:-8080}` | 8080 | Direct developer access (bypasses nginx; not auth-gated for developer convenience) |
| `ngrok` | *(profile: public only)* | — | Tunnel agent; publishes to ngrok cloud, not host |

## Internal-Only Ports (Compose network, not host-reachable)

| Service | Container Port | Reachable By | Notes |
|---------|---------------|--------------|-------|
| `redis` | 6379 | `app`, `scores-service` | Already `expose:` before this feature |
| `commits-service` | 8082 | `nginx` (proxy_pass) | Changed from `ports:` to `expose:` by this feature |
| `scores-service` | 8083 | `nginx` (proxy_pass + auth_request gate) | Changed from `ports:` to `expose:` by this feature |
| `qr-service` | 8084 | `app` (internal HTTP call) | Changed from `ports:` to `expose:` by this feature |

## Auth Gate Topology

All external score submissions enter via nginx at port 80, hit the `auth_request` gate which calls `GET /auth/check` on `app:8080`, and only reach `scores-service:8083` if the gate returns 200. After this feature, there is no host-reachable port that bypasses this gate on the write path.

```
Host/ngrok → nginx:80 → auth_request → app:8080/auth/check
                      → (on 200) → scores-service:8083/scores [POST]
                      → (on 401) → 401 to caller
```

## Debug Access

If a developer needs direct access to a microservice port during debugging, a compose override file can temporarily restore it:

```yaml
# docker-compose.override.yml (gitignored or local only)
services:
  scores-service:
    ports:
      - "8083:8083"
  commits-service:
    ports:
      - "8082:8082"
  qr-service:
    ports:
      - "8084:8084"
```

This override is applied automatically by `docker compose up` when the file exists. Remove it when debugging is complete.
