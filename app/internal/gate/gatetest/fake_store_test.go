package gatetest

import (
	"context"
	"testing"
	"time"
)

// This is test-support infrastructure, not production logic — but an incorrect fake
// would silently invalidate every test built on top of it, so it earns its own tests.

func TestFakeWindowStoreStartsEmpty(t *testing.T) {
	f := &FakeWindowStore{}
	got, err := f.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got != "" {
		t.Fatalf("Current() = %q, want empty before any Activate", got)
	}
}

func TestFakeWindowStoreActivateThenCurrent(t *testing.T) {
	f := &FakeWindowStore{}
	id, err := f.Activate(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	got, err := f.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got != id {
		t.Fatalf("Current() = %q, want %q", got, id)
	}
}

func TestFakeWindowStoreActivateProducesDistinctIDs(t *testing.T) {
	f := &FakeWindowStore{}
	a, err := f.Activate(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	b, err := f.Activate(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if a == b {
		t.Fatalf("Activate produced the same ID twice: %q", a)
	}
}

func TestFakeWindowStoreExpire(t *testing.T) {
	f := &FakeWindowStore{}
	if _, err := f.Activate(context.Background(), time.Minute); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	f.Expire()
	got, err := f.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got != "" {
		t.Fatalf("Current() = %q after Expire, want empty", got)
	}
}

func TestFakeWindowStoreTTLZeroNeverExpires(t *testing.T) {
	f := &FakeWindowStore{}
	id, err := f.Activate(context.Background(), 0)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	got, err := f.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got != id {
		t.Fatalf("Current() = %q, want %q (ttl=0 should not expire)", got, id)
	}
}

func TestFakeWindowStoreTTLExpiresOnItsOwn(t *testing.T) {
	f := &FakeWindowStore{}
	if _, err := f.Activate(context.Background(), 20*time.Millisecond); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	time.Sleep(60 * time.Millisecond)
	got, err := f.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got != "" {
		t.Fatalf("Current() = %q after TTL lapsed, want empty", got)
	}
}
