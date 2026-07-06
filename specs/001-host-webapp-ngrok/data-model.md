# Data Model: Host Web App with Public Ngrok Access

This feature introduces no database or persisted data. The two "entities" named in the spec's Key
Entities section are runtime/configuration concepts, not stored records. They are documented here
as the shape any task or later feature should assume.

## Hosted Web App

The running instance of the static frontend content being served.

| Field | Type | Notes |
|-------|------|-------|
| `content_root` | path | Directory served by the webserver; `frontend/game/` for this feature. |
| `local_url` | string | Fixed, e.g. `http://localhost:8080`; always present once the webserver container is up. |
| `reachability` | enum: `local`, `local+public` | Whether the optional public tunnel is also active. |

No relationships to other entities beyond the Public Access Endpoint below. No state transitions
beyond "up" / "down" at the container level, which Docker Compose already models.

## Public Access Endpoint

The current shareable public URL, present only when the `public` compose profile is active and the
tunnel has successfully connected.

| Field | Type | Notes |
|-------|------|-------|
| `url` | string (HTTPS) | Assigned by ngrok on tunnel start; not reserved, so it MAY change on every restart (see spec Edge Cases). Sourced from ngrok's own agent API (`http://localhost:4040/api/tunnels`), not stored by this feature. |
| `status` | enum: `available`, `unavailable` | `unavailable` covers missing/invalid credential, provider outage, or no internet — see FR-006. |
| `discovered_via` | constant: `ngrok agent UI (localhost:4040)` | Documents where the presenter looks; not a value computed or stored by any new code. |

No relationships, no persistence, no validation rules beyond "is currently reachable" — this is
observed directly from the ngrok container's own state, never written to a database by this
feature.
