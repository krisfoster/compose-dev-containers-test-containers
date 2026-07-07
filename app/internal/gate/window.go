// Package gate implements the QR-gated public access control for Crossy Whale:
// the current QR window (this file), signed visitor access grants (grant.go), and
// the HTTP gate decision that ties them together (middleware.go).
package gate

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// currentWindowKey is the single Redis key holding the currently active QR window ID.
// Its own TTL is what makes automatic expiry (FR-006) require no separate bookkeeping:
// when the key disappears, there is no valid window, full stop.
const currentWindowKey = "access:window:current"

// WindowStore is the seam between the gate decision and Redis. Defining it as an
// interface (rather than depending directly on a Redis client) serves two purposes:
//
// 1. Handlers and middleware are tested with an in-memory fake (see the gatetest
//    package) — fast, no containers, covers the logic above this boundary.
//
// 2. The real Redis-backed implementation (RedisWindowStore below) is tested
//    separately using Testcontainers-go, which starts a genuine Redis Docker
//    container for the test and tears it down on completion. This catches bugs
//    that mocks never would — for example, behaviour that changes between Redis
//    versions, or subtle differences in how TTL expiry works in a real server.
type WindowStore interface {
	// Current returns the active window ID, or "" if none is currently active
	// (never activated, or expired).
	Current(ctx context.Context) (string, error)

	// Activate generates a fresh window ID, makes it the current one (overwriting
	// any previous window immediately, per FR-007), and sets it to expire after ttl
	// on its own (per FR-006). It returns the new window ID.
	Activate(ctx context.Context, ttl time.Duration) (string, error)
}

// RedisWindowStore is the production WindowStore, backed by a single Redis string key.
type RedisWindowStore struct {
	client *redis.Client
}

// NewRedisWindowStore wraps an existing Redis client.
func NewRedisWindowStore(client *redis.Client) *RedisWindowStore {
	return &RedisWindowStore{client: client}
}

// Current implements WindowStore.
func (s *RedisWindowStore) Current(ctx context.Context) (string, error) {
	val, err := s.client.Get(ctx, currentWindowKey).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// Activate implements WindowStore.
func (s *RedisWindowStore) Activate(ctx context.Context, ttl time.Duration) (string, error) {
	id, err := generateWindowID()
	if err != nil {
		return "", err
	}
	if err := s.client.Set(ctx, currentWindowKey, id, ttl).Err(); err != nil {
		return "", err
	}
	return id, nil
}

// generateWindowID returns an opaque, unguessable, URL-safe random token.
func generateWindowID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
