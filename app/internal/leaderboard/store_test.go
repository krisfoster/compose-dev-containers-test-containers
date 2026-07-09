package leaderboard

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	testcontainers "github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// newTestRedisStore starts a real Redis container using Testcontainers-go and
// returns a RedisScoreStore connected to it, plus the raw client for assertions
// that the ScoreStore interface itself doesn't expose.
//
// Testcontainers-go starts an actual Docker container as part of the test.
// Each call to tcredis.Run() pulls the Redis image, maps a random host port to
// Redis's 6379, and returns a handle. The container is stopped and removed when
// the test ends (t.Cleanup). This means tests run against real Redis — the same
// stream semantics, the same append behaviour, the same XRANGE output format —
// rather than against a mock that could silently diverge from the real thing.
//
// DHI note: the container uses dhi.io/redis:8-alpine (matching production). Its
// hardened redis.conf enables protected-mode, blocking connections via the mapped
// host port. The command override below disables it for the test — the same fix
// applied in docker-compose.yml.
func newTestRedisStore(t *testing.T) (*RedisScoreStore, *redis.Client) {
	t.Helper()
	ctx := context.Background()

	// DHI redis enables protected-mode in its bundled redis.conf, which refuses the
	// test's connection (arriving via the mapped port, i.e. non-loopback). Override the
	// full command — the DHI entrypoint is `tini --`, not a shim that prepends
	// redis-server — to keep the hardened conf but disable protected-mode.
	container, err := tcredis.Run(ctx, "dhi.io/redis:8-alpine",
		testcontainers.WithCmd("redis-server", "/etc/redis/redis.conf", "--protected-mode", "no"))
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate redis container: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("get redis connection string: %v", err)
	}
	opts, err := redis.ParseURL(connStr)
	if err != nil {
		t.Fatalf("parse redis connection string: %v", err)
	}
	client := redis.NewClient(opts)
	t.Cleanup(func() { _ = client.Close() })

	return NewRedisScoreStore(client), client
}

func TestRedisScoreStoreWriteThenReadBack(t *testing.T) {
	store, client := newTestRedisStore(t)
	ctx := context.Background()

	if err := store.Write(ctx, Entry{Name: "kris", Score: 42}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	msgs, err := client.XRange(ctx, scoresStreamKey, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("XRange returned %d entries, want 1", len(msgs))
	}
	if got := msgs[0].Values["name"]; got != "kris" {
		t.Fatalf("name = %v, want %q", got, "kris")
	}
	if got := msgs[0].Values["score"]; got != "42" {
		t.Fatalf("score = %v, want %q", got, "42")
	}
}

// FR-007/FR-008: repeat attempts, including under an identical name, must each
// produce their own entry rather than overwriting a prior one.
func TestRedisScoreStoreWriteAppendsRatherThanOverwrites(t *testing.T) {
	store, client := newTestRedisStore(t)
	ctx := context.Background()

	if err := store.Write(ctx, Entry{Name: "kris", Score: 10}); err != nil {
		t.Fatalf("Write (1): %v", err)
	}
	if err := store.Write(ctx, Entry{Name: "kris", Score: 20}); err != nil {
		t.Fatalf("Write (2): %v", err)
	}

	length, err := client.XLen(ctx, scoresStreamKey).Result()
	if err != nil {
		t.Fatalf("XLen: %v", err)
	}
	if length != 2 {
		t.Fatalf("XLen = %d, want 2 (repeat name must not overwrite)", length)
	}
}

func unreachableStore() *RedisScoreStore {
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 200 * time.Millisecond,
	})
	return NewRedisScoreStore(client)
}

func TestRedisScoreStoreWritePropagatesConnectionError(t *testing.T) {
	store := unreachableStore()
	if err := store.Write(context.Background(), Entry{Name: "kris", Score: 1}); err == nil {
		t.Fatal("Write against an unreachable Redis returned no error")
	}
}

func TestRedisScoreStoreNotifyPublishesToChannel(t *testing.T) {
	store, client := newTestRedisStore(t)
	ctx := context.Background()

	sub := client.Subscribe(ctx, "leaderboard:score-updated")
	defer sub.Close()
	msgCh := sub.Channel()

	// Give the subscriber time to register before publishing.
	time.Sleep(50 * time.Millisecond)

	if err := store.Notify(ctx); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	select {
	case msg := <-msgCh:
		if msg.Channel != "leaderboard:score-updated" {
			t.Fatalf("received on channel %q, want %q", msg.Channel, "leaderboard:score-updated")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Notify did not deliver message to subscriber within 2s")
	}
}
