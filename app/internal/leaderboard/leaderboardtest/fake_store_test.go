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

func TestFakeScoreStoreTopRanksLikeRedisScoreStore(t *testing.T) {
	f := &FakeScoreStore{}
	ctx := context.Background()
	_ = f.Write(ctx, leaderboard.Entry{Name: "low", Score: 5})
	_ = f.Write(ctx, leaderboard.Entry{Name: "high", Score: 50})
	_ = f.Write(ctx, leaderboard.Entry{Name: "mid", Score: 20})

	got, err := f.Top(ctx, 2)
	if err != nil {
		t.Fatalf("Top: %v", err)
	}
	want := []leaderboard.Entry{{Name: "high", Score: 50}, {Name: "mid", Score: 20}}
	if len(got) != len(want) {
		t.Fatalf("Top(2) = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Top(2)[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestFakeScoreStoreTopErrPreventsRead(t *testing.T) {
	wantErr := errors.New("intentional test failure")
	f := &FakeScoreStore{TopErr: wantErr}
	_, err := f.Top(context.Background(), 10)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Top error = %v, want %v", err, wantErr)
	}
}
