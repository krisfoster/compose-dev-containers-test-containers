package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"crossywhale/app/internal/gate"
	"crossywhale/app/internal/gate/gatetest"
)

// newTestApp builds an App wired to an in-memory fake WindowStore and a fake ngrok
// inspection API, so these tests never touch a real container or the network
// (constitution Principle III reserves real Redis for the WindowStore's own tests;
// everything above that boundary is fair game for fakes).
func newTestApp(t *testing.T) (*App, *gatetest.FakeWindowStore) {
	t.Helper()

	ngrokServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tunnels":[{"public_url":"https://demo.ngrok-free.app","proto":"https"}]}`))
	}))
	t.Cleanup(ngrokServer.Close)

	frontendDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(frontendDir, "index.html"), []byte("<html>game</html>"), 0o644); err != nil {
		t.Fatalf("write fake index.html: %v", err)
	}

	store := &gatetest.FakeWindowStore{}
	signer := gate.NewSigner([]byte("test-secret"), time.Hour)

	app := &App{
		store:       store,
		gate:        gate.NewGate(store, signer),
		frontendDir: frontendDir,
		ngrokAPIURL: ngrokServer.URL,
		qrWindowTTL: time.Minute,
		httpClient:  ngrokServer.Client(),
	}
	return app, store
}

func TestHandleHostAutoActivatesOnFirstVisit(t *testing.T) {
	app, store := newTestApp(t)

	before, err := store.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if before != "" {
		t.Fatalf("test setup: expected no window active yet, got %q", before)
	}

	req := httptest.NewRequest(http.MethodGet, "/host", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	after, err := store.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if after == "" {
		t.Fatal("expected /host to auto-activate a window on first visit")
	}
}

func TestHandleHostDoesNotReactivateWhenAlreadyActive(t *testing.T) {
	app, store := newTestApp(t)
	existing, err := store.Activate(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/host", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	current, err := store.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if current != existing {
		t.Fatalf("Current() = %q, want unchanged %q", current, existing)
	}
}

// The rotate button must submit via fetch/JS so the QR image updates without a full
// page reload, per the presenter-facing UX request; this checks that wiring is
// actually present in the served markup rather than asserting on behavior a Go test
// can't execute (there's no JS runtime here to click the button).
func TestHandleHostPageWiresUpInPlaceRotate(t *testing.T) {
	app, _ := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/host", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `id="qr"`) {
		t.Fatal("host page missing img#qr for the rotate script to target")
	}
	if !strings.Contains(body, `fetch('/host/rotate'`) {
		t.Fatal("host page missing the in-place rotate fetch() call")
	}
}

func TestHandleQRPNGBeforeAnyWindowActive(t *testing.T) {
	app, _ := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/qr.png", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d before any window is active", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleQRPNGWithActiveWindow(t *testing.T) {
	app, store := newTestApp(t)
	if _, err := store.Activate(context.Background(), time.Minute); err != nil {
		t.Fatalf("Activate: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/qr.png", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected non-empty PNG body")
	}
}

func TestHandleHostRotateGeneratesFreshWindow(t *testing.T) {
	app, store := newTestApp(t)
	oldID, err := store.Activate(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/host/rotate", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/host" {
		t.Fatalf("Location = %q, want /host", loc)
	}

	newID, err := store.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if newID == "" || newID == oldID {
		t.Fatalf("Current() = %q, want a fresh window distinct from %q", newID, oldID)
	}
}

func TestHandleHostRotateRejectsGet(t *testing.T) {
	app, _ := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/host/rotate", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// FR-004: the ungated listener serves /play with no cookie and no token required.
func TestUngatedPlayRequiresNoGate(t *testing.T) {
	app, _ := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/play", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "<html>game</html>" {
		t.Fatalf("body = %q, want the game's index.html content", rec.Body.String())
	}
}

// US2: the gated listener rejects a request with neither a valid grant nor token,
// including when no window has ever been activated at all (fail closed, FR-009).
func TestGatedPlayRejectsWithNoGrantOrToken(t *testing.T) {
	app, _ := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/play", nil)
	rec := httptest.NewRecorder()
	app.gatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestGatedPlayAllowsValidToken(t *testing.T) {
	app, store := newTestApp(t)
	windowID, err := store.Activate(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/play?w="+windowID, nil)
	rec := httptest.NewRecorder()
	app.gatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d (redirect after granting access)", rec.Code, http.StatusFound)
	}
}

// /qr.png, /host, and /host/rotate must not exist on the gated listener at all.
func TestGatedListenerDoesNotExposeHostRoutes(t *testing.T) {
	app, _ := newTestApp(t)

	for _, path := range []string{"/qr.png", "/host", "/host/rotate"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		app.gatedMux().ServeHTTP(rec, req)

		// Falls through to gatedMux's "/" gate, since these paths aren't
		// separately registered there — so an unauthorized request is
		// rejected (403), never served as if it were a real host route.
		if rec.Code == http.StatusOK {
			t.Fatalf("path %q unexpectedly succeeded on the gated listener", path)
		}
	}
}

func TestEnvOr(t *testing.T) {
	t.Setenv("APP_TEST_ENV_OR", "custom")
	if got := envOr("APP_TEST_ENV_OR", "fallback"); got != "custom" {
		t.Fatalf("envOr = %q, want %q", got, "custom")
	}
	if got := envOr("APP_TEST_ENV_OR_UNSET", "fallback"); got != "fallback" {
		t.Fatalf("envOr = %q, want %q", got, "fallback")
	}
}

func TestEnvDurationOr(t *testing.T) {
	t.Setenv("APP_TEST_DURATION", "30s")
	if got := envDurationOr("APP_TEST_DURATION", time.Minute); got != 30*time.Second {
		t.Fatalf("envDurationOr = %v, want 30s", got)
	}
	if got := envDurationOr("APP_TEST_DURATION_UNSET", time.Minute); got != time.Minute {
		t.Fatalf("envDurationOr = %v, want fallback 1m", got)
	}
	t.Setenv("APP_TEST_DURATION_BAD", "not-a-duration")
	if got := envDurationOr("APP_TEST_DURATION_BAD", time.Minute); got != time.Minute {
		t.Fatalf("envDurationOr(invalid) = %v, want fallback 1m", got)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	for _, key := range []string{
		"APP_WEB_PORT", "APP_GATED_PORT", "REDIS_ADDR", "GRANT_COOKIE_SECRET",
		"QR_WINDOW_TTL", "GRANT_LIFETIME", "NGROK_API_URL", "FRONTEND_DIR",
	} {
		t.Setenv(key, "") // envOr/envDurationOr treat "" the same as unset
	}

	cfg := loadConfig()
	if cfg.WebPort != "8080" || cfg.GatedPort != "8081" || cfg.RedisAddr != "redis:6379" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.QRWindowTTL != 15*time.Minute || cfg.GrantLifetime != 4*time.Hour {
		t.Fatalf("unexpected duration defaults: %+v", cfg)
	}
}

func TestLoadConfigReadsOverrides(t *testing.T) {
	t.Setenv("APP_WEB_PORT", "9090")
	t.Setenv("QR_WINDOW_TTL", "5m")

	cfg := loadConfig()
	if cfg.WebPort != "9090" {
		t.Fatalf("WebPort = %q, want 9090", cfg.WebPort)
	}
	if cfg.QRWindowTTL != 5*time.Minute {
		t.Fatalf("QRWindowTTL = %v, want 5m", cfg.QRWindowTTL)
	}
}

func TestNoPublicTunnelErrorMessage(t *testing.T) {
	if errNoPublicTunnel.Error() == "" {
		t.Fatal("expected a non-empty error message")
	}
}

// erroringStore is a gate.WindowStore test double that always fails, for exercising
// the error-handling branches real Redis outages would trigger.
type erroringStore struct{}

func (erroringStore) Current(context.Context) (string, error) {
	return "", errIntentionalTestFailure
}

func (erroringStore) Activate(context.Context, time.Duration) (string, error) {
	return "", errIntentionalTestFailure
}

var errIntentionalTestFailure = &testStoreError{}

type testStoreError struct{}

func (*testStoreError) Error() string { return "intentional test failure" }

func appWithErroringStore(t *testing.T) *App {
	t.Helper()
	signer := gate.NewSigner([]byte("test-secret"), time.Hour)
	store := erroringStore{}
	frontendDir := t.TempDir()
	return &App{
		store:       store,
		gate:        gate.NewGate(store, signer),
		frontendDir: frontendDir,
		ngrokAPIURL: "http://127.0.0.1:1/unreachable",
		qrWindowTTL: time.Minute,
		httpClient:  &http.Client{Timeout: 200 * time.Millisecond},
	}
}

func TestHandleQRPNGWhenStoreErrors(t *testing.T) {
	app := appWithErroringStore(t)
	req := httptest.NewRequest(http.MethodGet, "/qr.png", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleHostWhenStoreErrors(t *testing.T) {
	app := appWithErroringStore(t)
	req := httptest.NewRequest(http.MethodGet, "/host", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

// activateFailsStore succeeds at reporting "no window active" but fails specifically
// on Activate, to exercise handleHost's second, distinct error branch (separate from
// its Current-fails branch, covered above by erroringStore).
type activateFailsStore struct{}

func (activateFailsStore) Current(context.Context) (string, error) { return "", nil }

func (activateFailsStore) Activate(context.Context, time.Duration) (string, error) {
	return "", errIntentionalTestFailure
}

func TestHandleHostWhenActivateFails(t *testing.T) {
	store := activateFailsStore{}
	app := &App{
		store:       store,
		gate:        gate.NewGate(store, gate.NewSigner([]byte("test-secret"), time.Hour)),
		frontendDir: t.TempDir(),
	}
	req := httptest.NewRequest(http.MethodGet, "/host", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

// TestHandleQRPNGWhenNgrokUnreachable covers handleQRPNG's discoverPublicHost-fails
// branch specifically (an active window exists, but the public URL can't be found) —
// distinct from TestHandleQRPNGBeforeAnyWindowActive, which fails earlier for a
// different reason.
func TestHandleQRPNGWhenNgrokUnreachable(t *testing.T) {
	app, store := newTestApp(t)
	if _, err := store.Activate(context.Background(), time.Minute); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	app.ngrokAPIURL = "http://127.0.0.1:1/unreachable"

	req := httptest.NewRequest(http.MethodGet, "/qr.png", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleHostRotateWhenStoreErrors(t *testing.T) {
	app := appWithErroringStore(t)
	req := httptest.NewRequest(http.MethodPost, "/host/rotate", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestDiscoverPublicHostUnreachable(t *testing.T) {
	app, _ := newTestApp(t)
	app.ngrokAPIURL = "http://127.0.0.1:1/unreachable"
	if _, err := app.discoverPublicHost(context.Background()); err == nil {
		t.Fatal("expected an error when the ngrok API is unreachable")
	}
}

func TestDiscoverPublicHostMalformedJSON(t *testing.T) {
	app, _ := newTestApp(t)
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer badServer.Close()
	app.ngrokAPIURL = badServer.URL

	if _, err := app.discoverPublicHost(context.Background()); err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}

func TestDiscoverPublicHostNoHTTPSTunnel(t *testing.T) {
	app, _ := newTestApp(t)
	noTunnelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tunnels":[]}`))
	}))
	defer noTunnelServer.Close()
	app.ngrokAPIURL = noTunnelServer.URL

	if _, err := app.discoverPublicHost(context.Background()); err == nil {
		t.Fatal("expected an error when no https tunnel is reported")
	}
}
