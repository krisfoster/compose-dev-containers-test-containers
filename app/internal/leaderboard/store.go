// Package leaderboard implements score submission for Crossy Whale: validating and
// authorizing a player's completed-attempt name/score (handler.go) and durably
// recording it (store.go). See
// specs/003-leaderboard-score-submission/contracts/leaderboard-openapi.yaml for the
// full HTTP contract.
package leaderboard

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// scoresStreamKey is the single Redis Stream all Leaderboard Entries are appended to.
// A Stream (rather than a Sorted Set) is required because entries must never overwrite
// one another, even under an identical name (FR-007, FR-008) — see research.md §3.
const scoresStreamKey = "leaderboard:scores"

// Entry is one completed game attempt's recorded result.
type Entry struct {
	Name  string
	Score int
}

// ScoreStore is the seam between the score-write handler and Redis. Consumers depend
// on this interface rather than a concrete Redis client, so they can be tested against
// an in-memory fake (see the leaderboardtest package) while the Redis-backed
// implementation is tested against a real Redis via Testcontainers-go (constitution
// Principle III).
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
