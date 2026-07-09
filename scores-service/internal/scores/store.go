// Package scores implements the store and HTTP handlers for the scores microservice.
// It reads from the leaderboard:scores Redis Stream and subscribes to a Redis
// pub/sub channel to push live standings updates via SSE.
package scores

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const scoresStreamKey = "leaderboard:scores"

// Standing is one entry in the ranked standings list.
type Standing struct {
	Rank  int    `json:"rank"`
	Name  string `json:"name"`
	Score int    `json:"score"`
}

// Store reads score data from Redis and notifies on score changes.
type Store struct {
	client  *redis.Client
	limit   int
	channel string
}

// NewStore returns a Store that reads from client, caps results at limit entries,
// and subscribes to channel for change notifications.
func NewStore(client *redis.Client, limit int, channel string) *Store {
	return &Store{client: client, limit: limit, channel: channel}
}

// ReadBest reads the full leaderboard:scores stream, aggregates best score per
// player, sorts descending, and returns the top s.limit entries with 1-based ranks.
func (s *Store) ReadBest(ctx context.Context) ([]Standing, error) {
	msgs, err := s.client.XRange(ctx, scoresStreamKey, "-", "+").Result()
	if err != nil {
		return nil, fmt.Errorf("scores: xrange: %w", err)
	}

	// Aggregate best score per player name.
	best := make(map[string]int, len(msgs))
	for _, msg := range msgs {
		name, _ := msg.Values["name"].(string)
		scoreStr, _ := msg.Values["score"].(string)
		if name == "" {
			continue
		}
		score, err := strconv.Atoi(scoreStr)
		if err != nil {
			continue // skip malformed entries gracefully
		}
		if existing, ok := best[name]; !ok || score > existing {
			best[name] = score
		}
	}

	// Convert map to slice and sort by score descending.
	standings := make([]Standing, 0, len(best))
	for name, score := range best {
		standings = append(standings, Standing{Name: name, Score: score})
	}
	sort.Slice(standings, func(i, j int) bool {
		if standings[i].Score != standings[j].Score {
			return standings[i].Score > standings[j].Score
		}
		return standings[i].Name < standings[j].Name // stable tie-break by name
	})

	// Truncate to limit.
	if s.limit > 0 && len(standings) > s.limit {
		standings = standings[:s.limit]
	}

	// Assign 1-based ranks.
	for i := range standings {
		standings[i].Rank = i + 1
	}

	return standings, nil
}

// Entry is one completed game attempt's recorded result.
type Entry struct {
	Name  string
	Score int
}

// Write appends entry to the leaderboard:scores Redis Stream.
func (s *Store) Write(ctx context.Context, entry Entry) error {
	return s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: scoresStreamKey,
		Values: map[string]any{
			"name":  entry.Name,
			"score": entry.Score,
		},
	}).Err()
}

// Notify publishes a score-change notification to s.channel so SSE subscribers
// receive a live push after each successful write.
func (s *Store) Notify(ctx context.Context) error {
	return s.client.Publish(ctx, s.channel, "").Err()
}

// Subscribe subscribes to the pub/sub channel and sends struct{}{} on ch each
// time a message arrives. It runs until ctx is cancelled.
func (s *Store) Subscribe(ctx context.Context, ch chan<- struct{}) {
	sub := s.client.Subscribe(ctx, s.channel)
	defer sub.Close()

	msgCh := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-msgCh:
			if !ok {
				return
			}
			select {
			case ch <- struct{}{}:
			default:
				// drop if receiver is not ready — next notification will refresh anyway
			}
		}
	}
}
