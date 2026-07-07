# Container Image Inventory

**Purpose**: The FR-008 deliverable — a single record of every container image this project
references, its migration status, and the rationale for any exemption. Cross-checked against a
repo-wide search for image references (SC-002).

**Access model**: DHI Community images are pulled from `dhi.io` after `docker login dhi.io` with a
free Docker account (Apache-2.0, no paid subscription). See
[`../research.md`](../research.md) Decision 1.

**Last verified**: 2026-07-07 (all migrated tags reported 0 critical/high/medium/low CVEs).

| Role | Previous ref | Status | New ref (DHI) | Version line | Used in | Core demo path? | Rationale |
|------|--------------|--------|---------------|--------------|---------|-----------------|-----------|
| app-build | `golang:1.25-alpine` | ✅ migrated | `dhi.io/golang:1.25-alpine-dev` | 1.25 | `app/Dockerfile` (build stage) | build-time | Hardened Go toolchain; **dev** variant chosen for build (root + `apk` + shell); non-dev golang runs nonroot with `go help` entrypoint and no package manager, so it can't build. |
| app-runtime | `alpine:3.20` | ✅ migrated | `dhi.io/static:20260611-alpine3.24` | static (date tag) | `app/Dockerfile` (final stage) | yes | App is a `CGO_ENABLED=0` static binary; `static` is the minimal hardened runtime for Go binaries (nonroot uid 65532, ~0.23 MB, ships ca-certs+tzdata). Date-stamped tag — refresh on rebuild. |
| redis | `redis:7-alpine` | ✅ migrated | `dhi.io/redis:8-alpine` | 8.x (**was 7.x**) | `docker-compose.yml`; `app/internal/leaderboard/store_test.go`; `app/internal/gate/window_test.go` | yes | **Documented 7→8 major bump**: no hardened Redis 7.x exists (7.x is Debian-only, un-hardened); hardened Redis starts at 8.x on Alpine. Redis 8.x is backward-compatible for the sorted-set/hash/TTL/pub-sub/stream/HLL usage here; `go-redis/v9` supports it. |
| tunnel (ngrok) | `ngrok/ngrok:3` | ⛔ exempt | — | 3 | `docker-compose.yml` (`public` profile) | **no** | No DHI equivalent for ngrok. Runs only under the optional `public` tunnel profile, off the core local demo path, so exempting it does not affect the default demo. README documents obtaining a free ngrok account/authtoken. |

## Reconciliation (repo-wide image references → inventory)

Every image reference in the repo maps to exactly one row above:

- `app/Dockerfile` line 1 `FROM golang:1.25-alpine` → **app-build**
- `app/Dockerfile` line 8 `FROM alpine:3.20` → **app-runtime**
- `docker-compose.yml` `redis.image: redis:7-alpine` → **redis**
- `app/internal/leaderboard/store_test.go` `tcredis.Run(ctx, "redis:7-alpine")` → **redis**
- `app/internal/gate/window_test.go` `tcredis.Run(ctx, "redis:7-alpine")` → **redis**
- `docker-compose.yml` `ngrok.image: ngrok/ngrok:3` → **tunnel (ngrok)**
- `project.md` image table (docs only) → reflects the rows above (kept in sync per FR-011)

**Result**: 3 images with a DHI equivalent are migrated (SC-001 = 100%); 1 image (ngrok) is exempt
with rationale (SC-006 = exactly one). No image is left unaccounted for (SC-002 = 100%).

## Known configuration difference: DHI Redis enables protected-mode

⚠️ **This is the one behavioural gotcha of the migration — read this before debugging Redis errors.**

**Symptom.** After swapping `redis:7-alpine` → `dhi.io/redis:8-alpine` with no other change, every
Redis-backed feature fails even though the container is "healthy" and `redis-cli ping` works:

- App: `GET /host` → `503`, `GET /api/leaderboard/scores` → `500 {"error":"failed to load standings"}`,
  score submission → `500`.
- Tests: the Testcontainers boundary tests fail to connect.
- Redis-side error on the denied connection:
  `DENIED Redis is running in protected mode because protected mode is enabled and no password is set
  for the default user. In this mode connections are only accepted from the loopback interface.`

**Root cause.** DHI Redis ships a hardened `/etc/redis/redis.conf` with **`protected-mode` enabled**.
Protected mode refuses every **non-loopback** connection unless a password is set — so the app
container (a different container on the compose network) and the Testcontainers client (connecting
via the mapped host port) are both rejected. The public `redis:7-alpine` image did not enable this,
which is why the app worked before the swap. `redis-cli ping` succeeds only because it runs from
*inside* the container, i.e. over loopback.

**Correct workaround — disable `protected-mode` (redis is not network-exposed here).** Redis has
**no published port** in this project (compose publishes only the app's `8080`), so it is reachable
only on the internal compose network. The exposure protected-mode guards against (an
accidentally-internet-facing Redis) does not exist, so disabling it is the proportionate fix and
restores the prior `redis:7-alpine` behaviour without any app-code change. Keep the hardened conf and
append the override:

- **`docker-compose.yml`** (the `redis` service):

  ```yaml
  command: ["redis-server", "/etc/redis/redis.conf", "--protected-mode", "no"]
  ```

- **Testcontainers** (`store_test.go`, `window_test.go`) — pass the **full** command via `WithCmd`:

  ```go
  tcredis.Run(ctx, "dhi.io/redis:8-alpine",
      testcontainers.WithCmd("redis-server", "/etc/redis/redis.conf", "--protected-mode", "no"))
  ```

  ⚠️ Use `WithCmd` (replace the whole command), **not** `WithCmdArgs` (append). DHI Redis's
  entrypoint is `/usr/bin/tini --`, not a `docker-entrypoint.sh` that prepends `redis-server` — so a
  bare `--protected-mode no` would be executed as `tini -- --protected-mode no` and fail to start.

**Alternative (not used): set a password.** The more "hardened" option is to keep protected-mode on
and set `--requirepass <secret>`, then supply that secret to the Redis client. Rejected here as
disproportionate: it means threading a password through the go-redis client, `.env`, and both test
helpers, for a Redis that is already network-isolated behind the compose network. If this app ever
publishes the Redis port or runs Redis on a shared/untrusted network, switch to `requirepass`
instead of disabling protected-mode.

**Verification.** With the override applied: `go test ./...` passes (Testcontainers boundary tests
included), and the live API round-trip (score write → standings read) and `/host` QR window succeed
against `dhi.io/redis:8-alpine`.
