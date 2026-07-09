// Package leaderboard implements score submission and standings retrieval for Crossy
// Whale: validating/authorizing a write, ranking reads, (handler.go) and durably
// recording/reading entries (store.go). See
// specs/004-leaderboard-page/contracts/leaderboard-openapi.yaml for the full HTTP
// contract (the canonical, current version — see that file's header for history).
package leaderboard

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// scoresStreamKey is the single Redis Stream all Leaderboard Entries are appended to.
//
// A Redis Stream is an append-only log: every XADD call adds a new entry with a
// unique auto-generated ID; entries are never modified or removed. This is the right
// data structure here because every game attempt must be recorded independently —
// two players with the same name should both appear, not overwrite each other.
// A Redis Sorted Set, by contrast, uses the member name as a key, so a second write
// for "Alice" would replace her first score rather than recording both.
const scoresStreamKey = "leaderboard:scores"

// Entry is one completed game attempt's recorded result.
type Entry struct {
	Name  string
	Score int
}

// ScoreStore is the seam between the score-write handler and Redis. Defining it as an
// interface (rather than depending directly on a Redis client) serves two purposes:
//
// 1. The handler and ranking logic are tested with an in-memory fake (see the
//    leaderboardtest package) — fast, no containers, covers validation and ordering.
//
// 2. The real Redis-backed implementation (RedisScoreStore below) is tested
//    separately using Testcontainers-go, which starts a real Redis Docker container
//    for each test run and tears it down on completion. See store_test.go for the
//    pattern. This ensures the stream append and read logic works against actual
//    Redis behaviour, not a simulation.
type ScoreStore interface {
	// Write appends entry as a new Leaderboard Entry. It never updates or removes an
	// existing entry — every call adds exactly one new record, even if an entry with
	// the same name already exists.
	Write(ctx context.Context, entry Entry) error
}

// RedisScoreStore is the production ScoreStore, backed by a Redis Stream.
type RedisScoreStore struct {
	client *redis.Client
}

// NewRedisScoreStore wraps an existing Redis client.
func NewRedisScoreStore(client *redis.Client) *RedisScoreStore {
	return &RedisScoreStore{client: client}
}

// Write implements ScoreStore.
func (s *RedisScoreStore) Write(ctx context.Context, entry Entry) error {
	return s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: scoresStreamKey,
		Values: map[string]any{
			"name":  entry.Name,
			"score": entry.Score,
		},
	}).Err()
}

// scoresNotifyChannel is the Redis pub/sub channel published to after each successful
// score write so the scores-service can push SSE updates to connected clients.
const scoresNotifyChannel = "leaderboard:score-updated"

// ScoreNotifier publishes a score-change notification. Kept as a separate interface
// from ScoreStore so handler tests can stub notification independently from writes.
type ScoreNotifier interface {
	Notify(ctx context.Context) error
}

// Notify implements ScoreNotifier on RedisScoreStore. A missed notification delays the
// SSE update on the scores-service but does not affect score recording.
func (s *RedisScoreStore) Notify(ctx context.Context) error {
	return s.client.Publish(ctx, scoresNotifyChannel, "").Err()
}
