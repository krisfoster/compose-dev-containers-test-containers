package gate

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	testcontainers "github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// newTestRedisStore spins up a real Redis via Testcontainers-go (constitution
// Principle III — no mocked Redis client for anything touching this boundary) and
// returns a RedisWindowStore against it. The container is torn down when the test
// (and any subtests sharing t) completes.
func newTestRedisStore(t *testing.T) *RedisWindowStore {
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

	return NewRedisWindowStore(client)
}

func TestRedisWindowStoreCurrentWhenNeverActivated(t *testing.T) {
	store := newTestRedisStore(t)

	got, err := store.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got != "" {
		t.Fatalf("Current() = %q before any Activate, want empty (fail closed, FR-009)", got)
	}
}

func TestRedisWindowStoreActivateThenCurrent(t *testing.T) {
	store := newTestRedisStore(t)
	ctx := context.Background()

	id, err := store.Activate(ctx, time.Minute)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if id == "" {
		t.Fatal("Activate returned empty window ID")
	}

	got, err := store.Current(ctx)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got != id {
		t.Fatalf("Current() = %q, want %q", got, id)
	}
}

func TestRedisWindowStoreRotateInvalidatesPreviousToken(t *testing.T) {
	store := newTestRedisStore(t)
	ctx := context.Background()

	oldID, err := store.Activate(ctx, time.Minute)
	if err != nil {
		t.Fatalf("Activate (initial): %v", err)
	}

	newID, err := store.Activate(ctx, time.Minute)
	if err != nil {
		t.Fatalf("Activate (rotate): %v", err)
	}
	if newID == oldID {
		t.Fatalf("rotation produced the same window ID twice: %q", newID)
	}

	got, err := store.Current(ctx)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got != newID {
		t.Fatalf("Current() = %q after rotation, want the new ID %q (old ID must not still match)", got, newID)
	}
	if got == oldID {
		t.Fatalf("Current() still equals the pre-rotation ID %q", oldID)
	}
}

// unreachableStore points at a port nothing listens on, so calls fail with a real
// connection error rather than redis.Nil — covering the generic error path in
// Current/Activate that a healthy Redis in the other tests never exercises.
func unreachableStore() *RedisWindowStore {
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 200 * time.Millisecond,
	})
	return NewRedisWindowStore(client)
}

func TestRedisWindowStoreCurrentPropagatesConnectionError(t *testing.T) {
	store := unreachableStore()
	if _, err := store.Current(context.Background()); err == nil {
		t.Fatal("Current against an unreachable Redis returned no error")
	}
}

func TestRedisWindowStoreActivatePropagatesConnectionError(t *testing.T) {
	store := unreachableStore()
	if _, err := store.Activate(context.Background(), time.Minute); err == nil {
		t.Fatal("Activate against an unreachable Redis returned no error")
	}
}

func TestRedisWindowStoreExpiresOnItsOwn(t *testing.T) {
	store := newTestRedisStore(t)
	ctx := context.Background()

	if _, err := store.Activate(ctx, 500*time.Millisecond); err != nil {
		t.Fatalf("Activate: %v", err)
	}

	time.Sleep(1200 * time.Millisecond) // past the short TTL, with margin

	got, err := store.Current(ctx)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got != "" {
		t.Fatalf("Current() = %q after TTL lapsed with no manual action, want empty (FR-006)", got)
	}
}
