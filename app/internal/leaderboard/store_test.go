package leaderboard

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
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

	container, err := tcredis.Run(ctx, "redis:7-alpine")
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
