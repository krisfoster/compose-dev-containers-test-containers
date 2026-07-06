// Package gatetest provides an in-memory gate.WindowStore fake for tests.
//
// It is a regular (non-_test.go) package, deliberately: Go does not allow a _test.go
// file's symbols to be imported from another package's tests, and this fake needs to
// be usable both by gate's own middleware tests and by app/main_test.go. It must never
// be imported by production code.
package gatetest

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// FakeWindowStore is a simple in-memory gate.WindowStore. Zero value is ready to use,
// with no window active.
type FakeWindowStore struct {
	mu        sync.Mutex
	current   string
	expiresAt time.Time
	hasWindow bool
	nextID    int
}

// Current implements gate.WindowStore.
func (f *FakeWindowStore) Current(_ context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.hasWindow {
		return "", nil
	}
	if !f.expiresAt.IsZero() && time.Now().After(f.expiresAt) {
		return "", nil
	}
	return f.current, nil
}

// Activate implements gate.WindowStore. Fake IDs are simple incrementing tokens
// ("window-1", "window-2", ...) since uniqueness, not unguessability, is all tests need.
func (f *FakeWindowStore) Activate(_ context.Context, ttl time.Duration) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	f.current = "fake-window-" + strconv.Itoa(f.nextID)
	f.hasWindow = true
	if ttl > 0 {
		f.expiresAt = time.Now().Add(ttl)
	} else {
		f.expiresAt = time.Time{}
	}
	return f.current, nil
}

// Expire immediately simulates the current window's TTL having lapsed, without
// waiting on a real clock — useful for exercising expiry behavior in tests.
func (f *FakeWindowStore) Expire() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hasWindow = false
	f.current = ""
}
