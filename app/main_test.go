package main

import (
	"bytes"
	"context"
	"image"
	"image/png"
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

// Minimal template fixtures: enough content to satisfy the handler assertions used
// in existing tests. The real template files in templates/ are exercised by the
// compose stack validation (quickstart.md); these fixtures keep unit tests fast
// and focused on handler logic rather than page styling.

const testGettingStartedHTML = `<!DOCTYPE html><html><body>
<a href="/play">Play</a>
<a href="/leaderboard">Leaderboard</a>
</body></html>`

const testLeaderboardHTML = `<!DOCTYPE html><html><body>
<div id="scores-root"></div>
<script>ScoresComponent</script>
<script src="/leaderboard-assets/scores-component.js"></script>
<p id="scores-url">{{.ScoresServiceURL}}</p>
<p id="commits-url">{{.CommitsServiceURL}}</p>
</body></html>`

// newTestTemplatesDir creates a temp directory containing all three page template
// fixtures and returns its path. The caller's test cleanup removes it automatically.
func newTestTemplatesDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"index.html":           `<html><body>game</body></html>`,
		"getting-started.html": testGettingStartedHTML,
		"leaderboard.html":     testLeaderboardHTML,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write test template %s: %v", name, err)
		}
	}
	return dir
}

// minimalPNG returns a valid 1×1 PNG image for use as a stub response in tests
// that need a realistic image/png body without the real qr-service running.
func minimalPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode stub PNG: %v", err)
	}
	return buf.Bytes()
}

// newTestApp builds an App wired to an in-memory fake WindowStore, a fake ngrok
// inspection API, and a fake qr-service stub, so these tests never touch a real
// container or the network (constitution Principle III reserves real Redis for the
// WindowStore's and ScoreStore's own tests; everything above that boundary is fair
// game for fakes).
func newTestApp(t *testing.T) (*App, *gatetest.FakeWindowStore) {
	t.Helper()

	ngrokServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tunnels":[{"public_url":"https://demo.ngrok-free.app","proto":"https"}]}`))
	}))
	t.Cleanup(ngrokServer.Close)

	stubPNG := minimalPNG(t)
	qrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(stubPNG)
	}))
	t.Cleanup(qrServer.Close)

	store := &gatetest.FakeWindowStore{}
	signer := gate.NewSigner([]byte("test-secret"), time.Hour)

	app := &App{
		store:        store,
		gate:         gate.NewGate(store, signer),
		signer:       signer,
		templatesDir: newTestTemplatesDir(t),
		ngrokAPIURL:  ngrokServer.URL,
		qrWindowTTL:  time.Minute,
		httpClient:   &http.Client{Timeout: 3 * time.Second},
		qrServiceURL: qrServer.URL,
	}
	return app, store
}

func TestHandleAuthCheckWithValidCookie(t *testing.T) {
	app, _ := newTestApp(t)
	grant, err := app.signer.Sign(gate.NewGrant("test-window"))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/auth/check", nil)
	req.AddCookie(&http.Cookie{Name: gate.GrantCookieName, Value: grant})
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for valid cw_grant", rec.Code)
	}
}

func TestHandleAuthCheckWithNoCookie(t *testing.T) {
	app, _ := newTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/auth/check", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 when no cookie", rec.Code)
	}
}

func TestHandleAuthCheckWithExpiredCookie(t *testing.T) {
	app, _ := newTestApp(t)
	// Sign a grant with an IssuedAt 2 hours in the past — the app's signer has a 1-hour
	// lifetime, so this grant is expired from the signer's perspective even though the
	// HMAC is valid (same secret, different timestamp).
	expiredGrant := gate.Grant{
		GrantID:        "expired-grant",
		IssuedWindowID: "test-window",
		IssuedAt:       time.Now().Add(-2 * time.Hour),
	}
	cookieVal, err := app.signer.Sign(expiredGrant)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/auth/check", nil)
	req.AddCookie(&http.Cookie{Name: gate.GrantCookieName, Value: cookieVal})
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for expired grant", rec.Code)
	}
}

func TestHandleAuthCheckWithInvalidCookie(t *testing.T) {
	app, _ := newTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/auth/check", nil)
	req.AddCookie(&http.Cookie{Name: gate.GrantCookieName, Value: "not.valid"})
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for invalid cookie value", rec.Code)
	}
}

func TestHandleAuthCheckRejectsNonGet(t *testing.T) {
	app, _ := newTestApp(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/check", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405 for non-GET", rec.Code)
	}
}

func TestSingleMuxGatesPlayWithMiddleware(t *testing.T) {
	app, _ := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/play", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 — single mux must gate /play via gate.Middleware", rec.Code)
	}
}

func TestSingleMuxIssuesGrantOnValidToken(t *testing.T) {
	app, store := newTestApp(t)
	windowID, err := store.Activate(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/play?w="+windowID, nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (redirect after grant issuance)", rec.Code)
	}
	if cookies := rec.Header().Get("Set-Cookie"); cookies == "" {
		t.Fatal("expected Set-Cookie header with cw_grant after valid window token")
	}
}

func TestHandleQRPNGAutoActivatesWhenNoWindow(t *testing.T) {
	app, store := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/qr.png", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d — /qr.png should auto-activate and return PNG", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", ct)
	}
	after, err := store.Current(context.Background())
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if after == "" {
		t.Fatal("/qr.png should have auto-activated a window")
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

func TestHandleQRPNGWhenQRServiceDown(t *testing.T) {
	app, store := newTestApp(t)
	if _, err := store.Activate(context.Background(), time.Minute); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	app.qrServiceURL = "http://127.0.0.1:1/unreachable"

	req := httptest.NewRequest(http.MethodGet, "/qr.png", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d when qr-service is down", rec.Code, http.StatusServiceUnavailable)
	}
}

// The bare "/" must serve the getting-started landing page, not the raw
// (credential-broken) game file — this is what closes the silent score-submission
// failure a player hits by opening "/" instead of "/play".
func TestHandleRootServesGettingStartedPage(t *testing.T) {
	app, _ := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{`href="/play"`, `href="/leaderboard"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("landing page missing link %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "__LEADERBOARD_TOKEN__") {
		t.Fatal("landing page must not be the raw (credential-broken) game file")
	}
}

// TestHandleRootMissingTemplate confirms handleRootOrAsset returns 500 when the
// getting-started.html template file cannot be found.
func TestHandleRootMissingTemplate(t *testing.T) {
	app, _ := newTestApp(t)
	app.templatesDir = t.TempDir() // empty — no templates here

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 when getting-started.html is missing", rec.Code)
	}
}

// The old /api/commits endpoint was removed when the commits microservice was
// introduced (012-git-commits-microservice). The ungated mux must return 404.
// The gated mux's catch-all "/" is gated, so an unauthenticated request returns
// 403 from the gate middleware — this also confirms no dedicated commits route exists.
func TestOldCommitsEndpointRemoved(t *testing.T) {
	app, _ := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/commits", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("ungated mux /api/commits: got %d, want 404", rec.Code)
	}

}

// /play on the single mux is gated by gate.Middleware, so a request with no cookie
// is rejected with 403 (covered by TestSingleMuxGatesPlayWithMiddleware). A request
// with a valid window token is redirected after grant issuance
// (TestSingleMuxIssuesGrantOnValidToken). The template file missing case is below.

func TestHandlePlayIndexWhenTemplateFileMissing(t *testing.T) {
	app, _ := newTestApp(t)
	app.templatesDir = t.TempDir() // no index.html written here

	// gate.Middleware gates /play; attach a valid grant cookie so the middleware
	// passes through to handlePlayIndex, which then fails on the missing template.
	token, err := app.signer.Sign(gate.NewGrant("test-window"))
	if err != nil {
		t.Fatalf("signer.Sign: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/play", nil)
	req.AddCookie(&http.Cookie{Name: gate.GrantCookieName, Value: token})
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}


// The page's own script (not exercised by this Go test) is what actually fetches and
// polls standings; this only asserts the served markup wires up the expected pieces.
func TestHandleLeaderboardPageMountsScoresComponent(t *testing.T) {
	app, _ := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/leaderboard", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `id="scores-root"`) {
		t.Fatal("leaderboard page missing scores-root mount point")
	}
	if !strings.Contains(body, `ScoresComponent`) {
		t.Fatal("leaderboard page missing ScoresComponent mount")
	}
	if !strings.Contains(body, `scores-component.js`) {
		t.Fatal("leaderboard page missing scores-component.js script")
	}
}

// TestHandleLeaderboardPageInjectsServiceURLs confirms the template data fields
// CommitsServiceURL and ScoresServiceURL are rendered into the leaderboard page.
func TestHandleLeaderboardPageInjectsServiceURLs(t *testing.T) {
	app, _ := newTestApp(t)
	app.commitsServiceURL = "http://commits.test"
	app.scoresServiceURL = "http://scores.test"

	req := httptest.NewRequest(http.MethodGet, "/leaderboard", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "http://commits.test") {
		t.Fatal("leaderboard page missing injected commitsServiceURL")
	}
	if !strings.Contains(body, "http://scores.test") {
		t.Fatal("leaderboard page missing injected scoresServiceURL")
	}
}

// TestHandleLeaderboardPageMissingTemplate confirms handleLeaderboardPage returns 500
// when the leaderboard.html template file cannot be found.
func TestHandleLeaderboardPageMissingTemplate(t *testing.T) {
	app, _ := newTestApp(t)
	app.templatesDir = t.TempDir() // empty — no templates here

	req := httptest.NewRequest(http.MethodGet, "/leaderboard", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 when leaderboard.html is missing", rec.Code)
	}
}

// TestHandlePingCompositeID confirms handlePing returns a composite id of the form
// "<startupID>.<templateVersion>" and that the version part increments when
// templateVersion is advanced.
func TestHandlePingCompositeID(t *testing.T) {
	app, _ := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	rec := httptest.NewRecorder()
	app.handlePing(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	// The id must contain a dot separating startupID from templateVersion.
	if !strings.Contains(body, ".") {
		t.Fatalf("ping response id missing dot separator: %s", body)
	}
	// Advance templateVersion and confirm the id changes.
	app.templateVersion.Add(1)
	rec2 := httptest.NewRecorder()
	app.handlePing(rec2, httptest.NewRequest(http.MethodGet, "/api/ping", nil))
	body2 := rec2.Body.String()
	if body == body2 {
		t.Fatalf("ping id did not change after templateVersion increment: %s", body2)
	}
}

// TestWatchTemplatesDetectsChange confirms that watchTemplates increments
// templateVersion when a monitored file's mtime advances.
func TestWatchTemplatesDetectsChange(t *testing.T) {
	app, _ := newTestApp(t)
	// Point the watcher at the test templates dir (already contains the three files).
	// Override the poll interval by starting watchTemplates directly in a goroutine
	// — the default 1-second poll is fast enough for this test.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go app.watchTemplates(ctx)

	// Give the watcher time to record its initial baselines.
	time.Sleep(50 * time.Millisecond)

	initialVersion := app.templateVersion.Load()

	// Rewrite one of the template files to advance its mtime.
	p := filepath.Join(app.templatesDir, "leaderboard.html")
	if err := os.WriteFile(p, []byte(testLeaderboardHTML+" <!-- touched -->"), 0o644); err != nil {
		t.Fatalf("rewrite leaderboard.html: %v", err)
	}

	// Wait up to 3 seconds for the watcher to detect the change.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if app.templateVersion.Load() > initialVersion {
			return // success
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("templateVersion did not increment within 3s after modifying leaderboard.html (initial=%d, current=%d)",
		initialVersion, app.templateVersion.Load())
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
		"APP_WEB_PORT", "REDIS_ADDR", "GRANT_COOKIE_SECRET",
		"QR_WINDOW_TTL", "GRANT_LIFETIME", "NGROK_API_URL", "TEMPLATES_DIR",
	} {
		t.Setenv(key, "") // envOr/envDurationOr treat "" the same as unset
	}

	cfg := loadConfig()
	if cfg.WebPort != "8080" || cfg.RedisAddr != "redis:6379" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.QRWindowTTL != 15*time.Minute || cfg.GrantLifetime != 4*time.Hour {
		t.Fatalf("unexpected duration defaults: %+v", cfg)
	}
	if cfg.TemplatesDir != "/templates" {
		t.Fatalf("TemplatesDir = %q, want /templates", cfg.TemplatesDir)
	}
}

func TestLoadConfigReadsOverrides(t *testing.T) {
	t.Setenv("APP_WEB_PORT", "9090")
	t.Setenv("QR_WINDOW_TTL", "5m")
	t.Setenv("TEMPLATES_DIR", "/custom-templates")

	cfg := loadConfig()
	if cfg.WebPort != "9090" {
		t.Fatalf("WebPort = %q, want 9090", cfg.WebPort)
	}
	if cfg.QRWindowTTL != 5*time.Minute {
		t.Fatalf("QRWindowTTL = %v, want 5m", cfg.QRWindowTTL)
	}
	if cfg.TemplatesDir != "/custom-templates" {
		t.Fatalf("TemplatesDir = %q, want /custom-templates", cfg.TemplatesDir)
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
	return &App{
		store:        store,
		gate:         gate.NewGate(store, signer),
		templatesDir: newTestTemplatesDir(t),
		ngrokAPIURL:  "http://127.0.0.1:1/unreachable",
		qrWindowTTL:  time.Minute,
		httpClient:   &http.Client{Timeout: 200 * time.Millisecond},
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

// activateFailsStore succeeds at reporting "no window active" but fails specifically
// on Activate, to exercise handleQRPNG's auto-activation error branch (separate from
// its Current-fails branch, covered above by erroringStore).
type activateFailsStore struct{}

func (activateFailsStore) Current(context.Context) (string, error) { return "", nil }

func (activateFailsStore) Activate(context.Context, time.Duration) (string, error) {
	return "", errIntentionalTestFailure
}

func TestHandleQRPNGWhenAutoActivateFails(t *testing.T) {
	store := activateFailsStore{}
	signer := gate.NewSigner([]byte("test-secret"), time.Hour)
	app := &App{
		store:        store,
		gate:         gate.NewGate(store, signer),
		signer:       signer,
		templatesDir: newTestTemplatesDir(t),
		ngrokAPIURL:  "http://127.0.0.1:1/unreachable",
		qrWindowTTL:  time.Minute,
		httpClient:   &http.Client{Timeout: 200 * time.Millisecond},
	}
	req := httptest.NewRequest(http.MethodGet, "/qr.png", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 when auto-activation fails", rec.Code)
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

func TestHandleRepoQRPNGReturnsValidPNG(t *testing.T) {
	app, _ := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/repo-qr.png", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected non-empty PNG body")
	}
}

func TestHandleRepoQRPNGCacheControl(t *testing.T) {
	app, _ := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/repo-qr.png", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=86400" {
		t.Fatalf("Cache-Control = %q, want public, max-age=86400", cc)
	}
}

func TestHandleRepoQRPNGWhenQRServiceDown(t *testing.T) {
	app, _ := newTestApp(t)
	app.qrServiceURL = "http://127.0.0.1:1/unreachable"

	req := httptest.NewRequest(http.MethodGet, "/repo-qr.png", nil)
	rec := httptest.NewRecorder()
	app.ungatedMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d when qr-service is down", rec.Code, http.StatusServiceUnavailable)
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
		t.Fatal("expected no https tunnel is reported")
	}
}
