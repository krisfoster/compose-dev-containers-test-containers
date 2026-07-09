package scores

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	testcontainers "github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// newTestRedisClient starts a real Redis container and returns a connected client.
// The container is stopped when the test ends.
func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()

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
	return client
}

func TestReadBestEmpty(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewStore(client, 10, "leaderboard:score-updated")

	standings, err := store.ReadBest(context.Background())
	if err != nil {
		t.Fatalf("ReadBest: %v", err)
	}
	if len(standings) != 0 {
		t.Fatalf("ReadBest on empty stream = %+v, want empty slice", standings)
	}
}

func TestReadBestSingleEntry(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewStore(client, 10, "leaderboard:score-updated")
	ctx := context.Background()

	client.XAdd(ctx, &redis.XAddArgs{
		Stream: scoresStreamKey,
		Values: map[string]any{"name": "alice", "score": "42"},
	})

	standings, err := store.ReadBest(ctx)
	if err != nil {
		t.Fatalf("ReadBest: %v", err)
	}
	if len(standings) != 1 {
		t.Fatalf("ReadBest = %+v, want 1 entry", standings)
	}
	if standings[0].Rank != 1 || standings[0].Name != "alice" || standings[0].Score != 42 {
		t.Fatalf("standings[0] = %+v, want {Rank:1 Name:alice Score:42}", standings[0])
	}
}

func TestReadBestKeepsBestScorePerPlayer(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewStore(client, 10, "leaderboard:score-updated")
	ctx := context.Background()

	for _, v := range []map[string]any{
		{"name": "alice", "score": "10"},
		{"name": "alice", "score": "50"},
		{"name": "alice", "score": "30"},
	} {
		client.XAdd(ctx, &redis.XAddArgs{Stream: scoresStreamKey, Values: v})
	}

	standings, err := store.ReadBest(ctx)
	if err != nil {
		t.Fatalf("ReadBest: %v", err)
	}
	if len(standings) != 1 {
		t.Fatalf("ReadBest = %+v, want exactly 1 entry for alice", standings)
	}
	if standings[0].Score != 50 {
		t.Fatalf("standings[0].Score = %d, want 50 (best score)", standings[0].Score)
	}
}

func TestReadBestRankedDescending(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewStore(client, 10, "leaderboard:score-updated")
	ctx := context.Background()

	for _, v := range []map[string]any{
		{"name": "charlie", "score": "5"},
		{"name": "alice", "score": "100"},
		{"name": "bob", "score": "50"},
	} {
		client.XAdd(ctx, &redis.XAddArgs{Stream: scoresStreamKey, Values: v})
	}

	standings, err := store.ReadBest(ctx)
	if err != nil {
		t.Fatalf("ReadBest: %v", err)
	}
	if len(standings) != 3 {
		t.Fatalf("ReadBest = %+v, want 3 entries", standings)
	}
	wantOrder := []string{"alice", "bob", "charlie"}
	for i, want := range wantOrder {
		if standings[i].Name != want {
			t.Fatalf("standings[%d].Name = %q, want %q (full: %+v)", i, standings[i].Name, want, standings)
		}
		if standings[i].Rank != i+1 {
			t.Fatalf("standings[%d].Rank = %d, want %d", i, standings[i].Rank, i+1)
		}
	}
}

func TestReadBestRespectsLimit(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewStore(client, 2, "leaderboard:score-updated")
	ctx := context.Background()

	for _, v := range []map[string]any{
		{"name": "a", "score": "30"},
		{"name": "b", "score": "20"},
		{"name": "c", "score": "10"},
	} {
		client.XAdd(ctx, &redis.XAddArgs{Stream: scoresStreamKey, Values: v})
	}

	standings, err := store.ReadBest(ctx)
	if err != nil {
		t.Fatalf("ReadBest: %v", err)
	}
	if len(standings) != 2 {
		t.Fatalf("ReadBest with limit 2 = %d entries, want 2", len(standings))
	}
	if standings[0].Name != "a" || standings[1].Name != "b" {
		t.Fatalf("standings = %+v, want [a, b] (top 2 by score)", standings)
	}
}

func TestStoreWriteThenReadBack(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewStore(client, 10, "leaderboard:score-updated")
	ctx := context.Background()

	if err := store.Write(ctx, Entry{Name: "kris", Score: 42}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	standings, err := store.ReadBest(ctx)
	if err != nil {
		t.Fatalf("ReadBest: %v", err)
	}
	if len(standings) != 1 || standings[0].Name != "kris" || standings[0].Score != 42 {
		t.Fatalf("ReadBest after Write = %+v, want [{kris 42}]", standings)
	}
}

func TestStoreWriteAppendsRatherThanOverwrites(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewStore(client, 10, "leaderboard:score-updated")
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

func TestStoreNotifyPublishesToChannel(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewStore(client, 10, "leaderboard:score-updated")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := client.Subscribe(ctx, "leaderboard:score-updated")
	defer sub.Close()
	msgCh := sub.Channel()

	time.Sleep(50 * time.Millisecond)

	if err := store.Notify(ctx); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	select {
	case msg := <-msgCh:
		if msg.Channel != "leaderboard:score-updated" {
			t.Fatalf("received on channel %q, want leaderboard:score-updated", msg.Channel)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Notify did not deliver message within 2s")
	}
}

func TestSubscribeFiresOnPublish(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewStore(client, 10, "leaderboard:score-updated")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan struct{}, 1)
	go store.Subscribe(ctx, ch)

	// Give the subscriber time to register before publishing.
	time.Sleep(50 * time.Millisecond)

	if err := client.Publish(ctx, "leaderboard:score-updated", "").Err(); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case <-ch:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("Subscribe did not fire within 2s after Publish")
	}
}
