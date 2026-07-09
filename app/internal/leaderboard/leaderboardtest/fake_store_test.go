package leaderboardtest

import (
	"context"
	"errors"
	"testing"

	"crossywhale/app/internal/leaderboard"
)

// This is test-support infrastructure, not production logic — but an incorrect fake
// would silently invalidate every test built on top of it, so it earns its own tests.

func TestFakeScoreStoreStartsEmpty(t *testing.T) {
	f := &FakeScoreStore{}
	if got := f.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0 before any Write", got)
	}
}

func TestFakeScoreStoreWriteRecordsEntry(t *testing.T) {
	f := &FakeScoreStore{}
	entry := leaderboard.Entry{Name: "kris", Score: 42}
	if err := f.Write(context.Background(), entry); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := f.Len(); got != 1 {
		t.Fatalf("Len() = %d, want 1", got)
	}
	if f.Entries[0] != entry {
		t.Fatalf("Entries[0] = %+v, want %+v", f.Entries[0], entry)
	}
}

func TestFakeScoreStoreWriteErrPreventsRecording(t *testing.T) {
	wantErr := errors.New("intentional test failure")
	f := &FakeScoreStore{WriteErr: wantErr}
	err := f.Write(context.Background(), leaderboard.Entry{Name: "kris", Score: 1})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Write error = %v, want %v", err, wantErr)
	}
	if got := f.Len(); got != 0 {
		t.Fatalf("Len() = %d after WriteErr, want 0", got)
	}
}
