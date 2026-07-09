// Package leaderboardtest provides an in-memory leaderboard.ScoreStore fake for tests.
//
// It is a regular (non-_test.go) package, deliberately: Go does not allow a _test.go
// file's symbols to be imported from another package's tests, and this fake needs to
// be usable both by leaderboard's own handler tests and by app/main_test.go. It must
// never be imported by production code.
package leaderboardtest

import (
	"context"
	"sync"

	"crossywhale/app/internal/leaderboard"
)

// FakeScoreStore is a simple in-memory leaderboard.ScoreStore. Zero value is ready to
// use, with no entries recorded.
type FakeScoreStore struct {
	mu      sync.Mutex
	Entries []leaderboard.Entry

	// WriteErr, if set, is returned by Write instead of recording an entry — useful
	// for exercising a handler's failure path.
	WriteErr error
}

// Write implements leaderboard.ScoreStore.
func (f *FakeScoreStore) Write(_ context.Context, entry leaderboard.Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.WriteErr != nil {
		return f.WriteErr
	}
	f.Entries = append(f.Entries, entry)
	return nil
}

// Len reports how many entries have been recorded so far.
func (f *FakeScoreStore) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.Entries)
}

// FakeScoreNotifier is a leaderboard.ScoreNotifier stub for tests.
type FakeScoreNotifier struct {
	NotifyCalls int
	NotifyErr   error
}

// Notify implements leaderboard.ScoreNotifier.
func (f *FakeScoreNotifier) Notify(_ context.Context) error {
	f.NotifyCalls++
	return f.NotifyErr
}
