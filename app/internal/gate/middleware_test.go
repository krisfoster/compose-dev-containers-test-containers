package gate_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crossywhale/app/internal/gate"
	"crossywhale/app/internal/gate/gatetest"
)

func newTestGate(t *testing.T) (*gate.Gate, *gatetest.FakeWindowStore) {
	t.Helper()
	store := &gatetest.FakeWindowStore{}
	signer := gate.NewSigner([]byte("test-secret"), time.Hour)
	return gate.NewGate(store, signer), store
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

// US2: neither a valid grant cookie nor a valid w token → rejected, including when no
// window has ever been activated at all (fail closed, FR-009).
func TestMiddlewareRejectsWithNoCookieAndNoToken(t *testing.T) {
	g, _ := newTestGate(t)
	handler := g.Middleware(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/play", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestMiddlewareRejectsWrongToken(t *testing.T) {
	g, store := newTestGate(t)
	if _, err := store.Activate(context.Background(), time.Minute); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	handler := g.Middleware(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/play?w=not-the-real-token", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// US1: a valid w token mints a grant cookie and redirects to the clean URL; a
// follow-up request with only that cookie succeeds with no token present.
func TestMiddlewareValidTokenGrantsAccessAndRedirects(t *testing.T) {
	g, store := newTestGate(t)
	windowID, err := store.Activate(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	handler := g.Middleware(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/play?w="+windowID, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d (redirect)", rec.Code, http.StatusFound)
	}
	location := rec.Header().Get("Location")
	if location != "/play" {
		t.Fatalf("Location = %q, want clean /play with token stripped", location)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != gate.GrantCookieName {
		t.Fatalf("cookies = %v, want exactly one %q cookie", cookies, gate.GrantCookieName)
	}

	// Follow-up request with only the cookie, no token, must succeed (FR-005).
	req2 := httptest.NewRequest(http.MethodGet, "/play", nil)
	req2.AddCookie(cookies[0])
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("follow-up status = %d, want %d", rec2.Code, http.StatusOK)
	}
}

// FR-010/FR-011: concurrent grant requests each get distinct, non-colliding IDs.
func TestMiddlewareConcurrentGrantsAreDistinct(t *testing.T) {
	g, store := newTestGate(t)
	windowID, err := store.Activate(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	handler := g.Middleware(okHandler())

	seen := map[string]bool{}
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/play?w="+windowID, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		cookies := rec.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("visitor %d: got %d cookies, want 1", i, len(cookies))
		}
		if seen[cookies[0].Value] {
			t.Fatalf("visitor %d: grant cookie value collided with a previous visitor", i)
		}
		seen[cookies[0].Value] = true
	}
}

// FR-008: a visitor's existing grant keeps working even after the window it
// originated from is rotated or has expired.
func TestMiddlewareGrantSurvivesWindowRotationAndExpiry(t *testing.T) {
	g, store := newTestGate(t)
	windowID, err := store.Activate(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	handler := g.Middleware(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/play?w="+windowID, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	grantCookie := cookies[0]

	// Rotate the window out from under the grant.
	if _, err := store.Activate(context.Background(), time.Minute); err != nil {
		t.Fatalf("Activate (rotate): %v", err)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/play", nil)
	req2.AddCookie(grantCookie)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("after rotation, status = %d, want %d", rec2.Code, http.StatusOK)
	}

	// Expire the window entirely.
	store.Expire()

	req3 := httptest.NewRequest(http.MethodGet, "/play", nil)
	req3.AddCookie(grantCookie)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("after expiry, status = %d, want %d", rec3.Code, http.StatusOK)
	}
}

func TestMiddlewareRejectsTamperedCookie(t *testing.T) {
	g, store := newTestGate(t)
	if _, err := store.Activate(context.Background(), time.Minute); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	handler := g.Middleware(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/play", nil)
	req.AddCookie(&http.Cookie{Name: gate.GrantCookieName, Value: "not-a-real-grant"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
