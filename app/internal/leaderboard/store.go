// Package leaderboard implements score submission and standings retrieval for Crossy
// Whale: validating/authorizing a write, ranking reads, (handler.go) and durably
// recording/reading entries (store.go). See
// specs/004-leaderboard-page/contracts/leaderboard-openapi.yaml for the full HTTP
// contract (the canonical, current version — see that file's header for history).
package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"

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

	// Top returns up to limit Leaderboard Entries, ranked by score descending with
	// ties broken by most-recently-written first (specs/004-leaderboard-page/spec.md
	// FR-002 through FR-004). Returns an empty, non-nil slice if no entries exist.
	Top(ctx context.Context, limit int) ([]Entry, error)
}

// RankTop sorts entries — assumed to be in oldest-to-newest write order — by score
// descending, breaking ties by most-recently-written first, and truncates to at most
// limit results. Shared by RedisScoreStore.Top and the in-memory fake in
// leaderboardtest so both rank identically (research.md §1).
func RankTop(entries []Entry, limit int) []Entry {
	ranked := make([]Entry, len(entries))
	copy(ranked, entries)

	// Reverse to newest-first so a stable sort-by-score preserves "most recent wins
	// ties" without needing to track each entry's original stream position.
	for i, j := 0, len(ranked)-1; i < j; i, j = i+1, j-1 {
		ranked[i], ranked[j] = ranked[j], ranked[i]
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Score > ranked[j].Score })

	if limit < 0 {
		limit = 0
	}
	if limit < len(ranked) {
		ranked = ranked[:limit]
	}
	return ranked
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

// Top implements ScoreStore. It reads the entire stream (booth-scale — at most a few
// hundred entries per event, plan.md Scale/Scope) rather than maintaining a separate
// ranked structure, per research.md §1.
func (s *RedisScoreStore) Top(ctx context.Context, limit int) ([]Entry, error) {
	msgs, err := s.client.XRange(ctx, scoresStreamKey, "-", "+").Result()
	if err != nil {
		return nil, err
	}

	// msgs is oldest-to-newest (XRange ascending), matching RankTop's expected input.
	entries := make([]Entry, 0, len(msgs))
	for _, msg := range msgs {
		name, _ := msg.Values["name"].(string)
		scoreStr, _ := msg.Values["score"].(string)
		score, err := strconv.Atoi(scoreStr)
		if err != nil {
			return nil, fmt.Errorf("leaderboard: malformed score in stream entry %s: %w", msg.ID, err)
		}
		entries = append(entries, Entry{Name: name, Score: score})
	}

	return RankTop(entries, limit), nil
}
