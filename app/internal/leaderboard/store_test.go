package leaderboard

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
// returns a RedisScoreStore against it, plus the raw client for assertions the
// ScoreStore interface itself doesn't expose. The container is torn down when the
// test (and any subtests sharing t) completes.
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

func TestRedisScoreStoreTopOnEmptyStream(t *testing.T) {
	store, _ := newTestRedisStore(t)

	entries, err := store.Top(context.Background(), 10)
	if err != nil {
		t.Fatalf("Top: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Top on empty stream = %+v, want empty slice", entries)
	}
}

// FR-003 (specs/004-leaderboard-page/spec.md): standings are ordered by score
// descending.
func TestRedisScoreStoreTopOrdersByScoreDescending(t *testing.T) {
	store, _ := newTestRedisStore(t)
	ctx := context.Background()

	for _, e := range []Entry{{Name: "low", Score: 5}, {Name: "high", Score: 50}, {Name: "mid", Score: 20}} {
		if err := store.Write(ctx, e); err != nil {
			t.Fatalf("Write(%+v): %v", e, err)
		}
	}

	got, err := store.Top(ctx, 10)
	if err != nil {
		t.Fatalf("Top: %v", err)
	}
	want := []Entry{{Name: "high", Score: 50}, {Name: "mid", Score: 20}, {Name: "low", Score: 5}}
	if len(got) != len(want) {
		t.Fatalf("Top returned %d entries, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Top()[%d] = %+v, want %+v (full: %+v)", i, got[i], want[i], got)
		}
	}
}

// Edge case (spec.md): two entries with the exact same score are both shown, with the
// most recently written one ranked first.
func TestRedisScoreStoreTopBreaksTiesByMostRecentFirst(t *testing.T) {
	store, _ := newTestRedisStore(t)
	ctx := context.Background()

	if err := store.Write(ctx, Entry{Name: "first", Score: 10}); err != nil {
		t.Fatalf("Write (first): %v", err)
	}
	if err := store.Write(ctx, Entry{Name: "second", Score: 10}); err != nil {
		t.Fatalf("Write (second): %v", err)
	}

	got, err := store.Top(ctx, 10)
	if err != nil {
		t.Fatalf("Top: %v", err)
	}
	if len(got) != 2 || got[0].Name != "second" || got[1].Name != "first" {
		t.Fatalf("Top = %+v, want [second, first] (most recent tie wins)", got)
	}
}

// FR-004: the returned list is bounded to limit even when more entries exist.
func TestRedisScoreStoreTopRespectsLimit(t *testing.T) {
	store, _ := newTestRedisStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if err := store.Write(ctx, Entry{Name: "player", Score: i}); err != nil {
			t.Fatalf("Write (%d): %v", i, err)
		}
	}

	got, err := store.Top(ctx, 2)
	if err != nil {
		t.Fatalf("Top: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Top with limit 2 returned %d entries, want 2: %+v", len(got), got)
	}
	if got[0].Score != 4 || got[1].Score != 3 {
		t.Fatalf("Top(2) = %+v, want the two highest scores [4, 3]", got)
	}
}

func TestRedisScoreStoreTopPropagatesConnectionError(t *testing.T) {
	store := unreachableStore()
	if _, err := store.Top(context.Background(), 10); err == nil {
		t.Fatal("Top against an unreachable Redis returned no error")
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
