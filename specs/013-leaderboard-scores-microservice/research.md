# Research: Leaderboard Scores Microservice

All items below were open questions in the Technical Context; none remain as NEEDS CLARIFICATION.

## 1. Push trigger mechanism: Redis pub/sub vs polling

**Decision**: Redis pub/sub as the primary change-detection mechanism. The `app` service publishes to a `leaderboard:score-updated` channel immediately after each successful score write. The `scores-service` subscribes to this channel and pushes an SSE event to all connected clients on receipt.

**Rationale**:
- Pub/sub delivers notifications with sub-millisecond latency, satisfying SC-001's 5-second freshness target with significant headroom.
- No polling loop needed in the microservice — the service is reactive rather than periodic.
- Redis pub/sub is already in scope per the project constitution ("pub/sub are all in scope") and requires no new dependency in either `app` or `scores-service`.
- The only code change to `app` is a single `PUBLISH` call in the existing `serveSubmit` handler after a successful `store.Write()`.

**Alternatives considered**:
- **Fixed-interval polling of Redis (in scores-service)**: Simpler to implement (no pub/sub subscription goroutine), but introduces artificial latency proportional to the polling interval. A 3-second poll means standings updates could be delayed up to 3 seconds even after a near-instant pub/sub notification would have been available.
- **Redis keyspace notifications** (automatic notifications from Redis internals when a key changes): Avoids any change to `app`, but requires `notify-keyspace-events` Redis config to be set (an additional configuration dependency); also couples the service to Redis internal key names rather than an explicit application-level event.
- **Polling from scores-service + scores-service initiated SSE push**: Equivalent to polling but from a different vantage point; same latency penalty as above.

## 2. Best-per-player aggregation: in-process vs Redis Sorted Set

**Decision**: In-process aggregation from the full Redis Stream. On each SSE trigger, the scores-service reads the full `leaderboard:scores` stream with `XRANGE leaderboard:scores - +`, groups entries by player name in a Go map, retains only the maximum score per player, sorts descending, and caps at the configured limit.

**Rationale**:
- The existing Redis data structure is a Stream (`leaderboard:scores`) and the spec requires no changes to the write path. An in-process aggregation approach requires zero additional Redis writes.
- Booth scale is bounded: the stream will contain at most a few hundred entries over a demo session. A full read + in-process map aggregation is negligible overhead at this cardinality.
- Consistent with the existing `RankTop` pattern in `app/internal/leaderboard/store.go` — the same "read everything, rank in Go" approach is already proven correct for this data size.

**Alternatives considered**:
- **Redis Sorted Set with `ZADD NX GT`** (add only if score is greater than existing member): Would give O(log n) reads with `ZRANGE`; eliminates the need to read the full stream. However, it requires `app` to maintain an additional Redis data structure (`leaderboard:best-scores`) in parallel with the stream, adding write-path complexity to a service that is supposed to be read-only for scores. The `GT` flag also requires Redis 6.2+, adding a version constraint.
- **Sorted Set maintained solely by scores-service**: The scores-service would maintain its own Sorted Set from stream reads. Adds stateful mutation to a read-only service; introduces a dual-source-of-truth risk.

## 3. New Go module structure

**Decision**: `scores-service/` at the repo root as a standalone Go module (`module crossywhale/scores-service`), mirroring the `commits-service/` structure. It has its own `go.mod`, `go.sum`, and Dockerfile.

**Rationale**:
- Matches the established pattern for microservices in this repo (see `commits-service/`).
- Independent module means independent dependency graph: `scores-service` carries only `github.com/redis/go-redis/v9` and Testcontainers, without inheriting `app`'s full dependency tree.
- The Dockerfile build context is scoped to `scores-service/`, keeping image layers small and reproducible.

**Alternatives considered**:
- **Single module with a new `cmd/scores-service` entry point inside `crossywhale/app`**: Would share `go.mod` with `app`, making it impossible to drop the `app`-specific dependencies (QR, UUID, etc.) from the scores-service binary. Also complicates the Dockerfile build context.

## 4. SSE on-connect and stream behaviour

**Decision**: On connect, the scores-service immediately emits one `standings` event with the current standings from Redis. It then holds the connection open, emitting a new `standings` event each time a pub/sub notification arrives on `leaderboard:score-updated`. If the pub/sub subscription is lost, the service attempts reconnection in a retry loop rather than closing client connections. The `scores-component.js` React component falls back to polling `GET /scores` at a 5-second interval if `EventSource` is unavailable or the SSE connection fails permanently.

**Rationale**:
- Immediate on-connect event matches the pattern of the commits-service and ensures the leaderboard is populated from the first page load.
- Event-driven push (not periodic push) means the SSE connection stays quiet when no scores are submitted, avoiding unnecessary serialization and network traffic.
- A polling fallback in the React component provides resilience for environments where SSE connections are dropped by intermediaries (reverse proxies, ngrok tunnels), consistent with the commits-component fallback pattern.

## 5. Port assignment

**Decision**: `scores-service` listens on port `8083`, controlled by the `SCORES_LISTEN_ADDR` environment variable (default `:8083`). Port `8083` is published to the host in `docker-compose.yml` so the browser can reach it directly for SSE connections.

**Rationale**: Ports 8080 (app web), 8081 (app gated), and 8082 (commits-service) are already allocated. Port 8083 is the natural next assignment and does not conflict with any existing service.

## 6. Redis pub/sub channel name

**Decision**: `leaderboard:score-updated` (colon-namespaced, consistent with the existing `leaderboard:scores` stream key convention).

**Rationale**: The colon-namespace mirrors the existing Redis key naming convention in this repo. The channel name is explicit and unlikely to collide with any future keys.

## 7. App changes: adding the PUBLISH call

**Decision**: A `ScoreNotifier` interface (with a single `Notify(ctx context.Context) error` method) is injected into the leaderboard `Handler` alongside the existing `ScoreStore`. The `RedisScoreStore` implements both by publishing to `leaderboard:score-updated`. The handler calls `notifier.Notify()` after a successful `store.Write()` and logs (but does not return an error to the caller) on notification failure — a missed pub/sub event results in a slightly delayed standings update, not a failed score submission.

**Rationale**:
- Keeping `Notify` as a separate interface preserves testability: the handler tests can stub notification without needing a Redis pub/sub connection.
- Failing silently on a missed notification is appropriate: the score was recorded successfully; the SSE clients will receive the update on the next pub/sub event (or the next reconnect fallback poll). The caller should not receive a 500 error because a notification channel was momentarily unavailable.
