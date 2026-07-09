// Command app serves Crossy Whale on two listeners: an ungated one for the
// presenter's local access (docker-compose.yml publishes this port to the host), and
// a gated one that only the ngrok service points at. See
// specs/002-qr-gated-access/contracts/gate-http-contract.md for the full route table.
// The leaderboard API (score submission, added by 003-leaderboard-score-submission, and
// standings retrieval, added by 004-leaderboard-page) is documented separately in
// specs/004-leaderboard-page/contracts/leaderboard-openapi.yaml. The leaderboard
// display page itself (004-leaderboard-page) is served at /leaderboard on both
// listeners, ungated (FR-011/FR-013 of that feature's spec.md).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"crossywhale/app/internal/gate"
	"crossywhale/app/internal/qrcode"
)

// startupID is set once when the process starts. The browser polls /api/ping
// and reloads whenever this value changes, giving instant live-reload on redeploy.
var startupID = fmt.Sprintf("%d", time.Now().UnixNano())

func main() {
	cfg := loadConfig()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()

	store := gate.NewRedisWindowStore(redisClient)
	signer := gate.NewSigner([]byte(cfg.GrantCookieSecret), cfg.GrantLifetime)
	g := gate.NewGate(store, signer)

	app := &App{
		store:                store,
		gate:                 g,
		signer:               signer,
		frontendDir:          cfg.FrontendDir,
		templatesDir:         cfg.TemplatesDir,
		ngrokAPIURL:          cfg.NgrokAPIURL,
		qrWindowTTL:          cfg.QRWindowTTL,
		httpClient:           &http.Client{Timeout: 3 * time.Second},
		leaderboardAssetsDir: cfg.LeaderboardAssetsDir,
		commitsServiceURL:    cfg.CommitsServiceURL,
		scoresServiceURL:     cfg.ScoresServiceURL,
	}

	// Fail fast if any page template is missing — better to surface this at
	// startup than to silently serve 500s on the first request to each page.
	for _, name := range []string{"getting-started.html", "host.html", "leaderboard.html"} {
		p := filepath.Join(cfg.TemplatesDir, name)
		if _, err := os.Stat(p); err != nil {
			log.Fatalf("template file missing: %s: %v", p, err)
		}
	}

	go app.watchTemplates(context.Background())

	errc := make(chan error, 1)
	go func() {
		log.Printf("ungated listener starting on :%s", cfg.WebPort)
		errc <- http.ListenAndServe(":"+cfg.WebPort, app.ungatedMux())
	}()
	log.Fatal(<-errc)
}

// Config holds the environment-driven settings for the app service.
type Config struct {
	WebPort              string
	RedisAddr            string
	GrantCookieSecret    string
	QRWindowTTL          time.Duration
	GrantLifetime        time.Duration
	NgrokAPIURL          string
	FrontendDir          string
	TemplatesDir         string
	LeaderboardAssetsDir string
	CommitsServiceURL    string
	ScoresServiceURL     string
}

func loadConfig() Config {
	return Config{
		WebPort: envOr("APP_WEB_PORT", "8080"),
		// REDIS_ADDR default is "redis:6379" — "redis" is the service name defined in
		// docker-compose.yml. Docker Compose automatically provides DNS for every
		// service, so containers in the same Compose network resolve service names to
		// the correct container IP without any hard-coded addresses.
		RedisAddr:            envOr("REDIS_ADDR", "redis:6379"),
		GrantCookieSecret:    envOr("GRANT_COOKIE_SECRET", "dev-only-change-me"),
		QRWindowTTL:          envDurationOr("QR_WINDOW_TTL", 15*time.Minute),
		GrantLifetime:        envDurationOr("GRANT_LIFETIME", 4*time.Hour),
		NgrokAPIURL:          envOr("NGROK_API_URL", "http://ngrok:4040/api/tunnels"),
		FrontendDir:          envOr("FRONTEND_DIR", "/frontend"),
		// TEMPLATES_DIR is the container-side path to the page template .html files.
		// In the compose stack it is bind-mounted from ./templates so edits on the
		// host reach the running container immediately for live-reload (015-extract-html-templates).
		TemplatesDir:         envOr("TEMPLATES_DIR", "/templates"),
		LeaderboardAssetsDir: envOr("LEADERBOARD_ASSETS_DIR", "/leaderboard-assets"),
		// CommitsServiceURL is the base URL of the commits microservice, reachable
		// from the browser. Default is localhost:8082 for local dev; override via
		// COMMITS_SERVICE_URL in docker-compose or .env for demo environments.
		CommitsServiceURL: envOr("COMMITS_SERVICE_URL", "http://localhost:8082"),
		// ScoresServiceURL is the base URL of the scores microservice, reachable
		// from the browser. Default is localhost:8083 for local dev; override via
		// SCORES_SERVICE_URL in docker-compose or .env for demo environments.
		ScoresServiceURL: envOr("SCORES_SERVICE_URL", "http://localhost:8083"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDurationOr(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

// App holds the dependencies shared by both listeners' handlers.
type App struct {
	store                gate.WindowStore
	gate                 *gate.Gate
	signer               *gate.Signer
	frontendDir          string
	templatesDir         string
	ngrokAPIURL          string
	qrWindowTTL          time.Duration
	httpClient           *http.Client
	leaderboardAssetsDir string
	commitsServiceURL    string
	scoresServiceURL     string
	// templateVersion is atomically incremented by watchTemplates whenever any
	// page template file on disk changes. handlePing incorporates it into the
	// response id so the browser auto-reloads within its next poll cycle.
	templateVersion atomic.Int64
}

// watchTemplates polls the mtime of each page template file every second. When
// any file's modification time advances, templateVersion is incremented so the
// next /api/ping response carries a new id and all open browsers reload.
func (a *App) watchTemplates(ctx context.Context) {
	files := []string{
		filepath.Join(a.templatesDir, "getting-started.html"),
		filepath.Join(a.templatesDir, "host.html"),
		filepath.Join(a.templatesDir, "leaderboard.html"),
	}
	baselines := make(map[string]time.Time, len(files))
	for _, f := range files {
		if info, err := os.Stat(f); err == nil {
			baselines[f] = info.ModTime()
		}
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, f := range files {
				info, err := os.Stat(f)
				if err != nil {
					log.Printf("watchTemplates: stat %s: %v", f, err)
					continue
				}
				if info.ModTime().After(baselines[f]) {
					baselines[f] = info.ModTime()
					ver := a.templateVersion.Add(1)
					log.Printf("watchTemplates: %s changed (template version now %d)", f, ver)
				}
			}
		}
	}
}

// ungatedMux serves the game with no access check at all (FR-004), plus the
// presenter-only routes for displaying and rotating the QR code (FR-002, FR-007).
// The bare "/" serves a getting-started landing page rather than the raw game
// index.html — routing players through the explicit "/play" link ensures the
// gate.Middleware cookie check runs before the game loads.
func (a *App) ungatedMux() http.Handler {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir(a.frontendDir))
	mux.Handle("/", a.handleRootOrAsset(fileServer))
	mux.Handle("/play", a.gate.Middleware(http.HandlerFunc(a.handlePlayIndex)))
	mux.HandleFunc("/play-local", func(w http.ResponseWriter, r *http.Request) {
		token, err := a.signer.Sign(gate.NewGrant("local"))
		if err != nil {
			http.Error(w, "failed to generate local grant", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     gate.GrantCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		a.handlePlayIndex(w, r)
	})
	mux.HandleFunc("/qr.png", a.handleQRPNG)
	mux.HandleFunc("/repo-qr.png", handleRepoQRPNG)
	mux.HandleFunc("/api/ping", a.handlePing)
	mux.HandleFunc("/host", a.handleHost)
	mux.HandleFunc("/host/rotate", a.handleHostRotate)
	mux.HandleFunc("/leaderboard", a.handleLeaderboardPage)
	mux.Handle("/leaderboard-assets/", http.StripPrefix("/leaderboard-assets/", http.FileServer(http.Dir(a.leaderboardAssetsDir))))
	mux.HandleFunc("/auth/check", a.handleAuthCheck)
	return mux
}

// handlePlayIndex serves the game's index page under the /play path the QR code
// encodes, rendered as an html/template rather than a raw file so the configured
// leaderboard write credential can be injected into an inline script tag for the
// game client to read (specs/003-leaderboard-score-submission/research.md §4). The
// frontend's own asset references are root-relative (./script.js resolves to
// /script.js from a request to /play), which the same listener's root file server
// (also gated, on the gated mux) already serves.
func (a *App) handlePlayIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(filepath.Join(a.frontendDir, "index.html"))
	if err != nil {
		http.Error(w, "failed to load game", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, nil); err != nil {
		log.Printf("render index.html: %v", err)
	}
}

// handleQRPNG returns the current QR code as a PNG. It does not itself activate a
// window — that only happens via /host — so a request here before the presenter has
// ever opened /host correctly reports "not ready" rather than fabricating a code.
func (a *App) handleQRPNG(w http.ResponseWriter, r *http.Request) {
	windowID, err := a.store.Current(r.Context())
	if err != nil {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	if windowID == "" {
		http.Error(w, "no active QR code yet - open /host to generate one", http.StatusServiceUnavailable)
		return
	}

	publicHost, err := a.discoverPublicHost(r.Context())
	if err != nil {
		http.Error(w, "public URL not available yet", http.StatusServiceUnavailable)
		return
	}

	png, err := qrcode.RenderPNG(qrcode.BuildPlayURL(publicHost, windowID), 320)
	if err != nil {
		http.Error(w, "failed to render QR code", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

// handleRootOrAsset serves the getting-started page for an exact "/" request and
// delegates everything else (script.js, style.css, model assets, ...) to assets — the
// same catch-all fileServer /play's rendered page depends on for its own
// root-relative asset references.
func (a *App) handleRootOrAsset(assets http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			assets.ServeHTTP(w, r)
			return
		}
		p := filepath.Join(a.templatesDir, "getting-started.html")
		tmpl, err := template.ParseFiles(p)
		if err != nil {
			log.Printf("template parse error %s: %v", p, err)
			http.Error(w, "failed to load page", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, nil); err != nil {
			log.Printf("template execute error %s: %v", p, err)
		}
	}
}

// handleHost renders the presenter-facing page embedding the QR code, auto-activating
// the first window on first visit if none is active yet (FR-002).
func (a *App) handleHost(w http.ResponseWriter, r *http.Request) {
	current, err := a.store.Current(r.Context())
	if err != nil {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	if current == "" {
		if _, err := a.store.Activate(r.Context(), a.qrWindowTTL); err != nil {
			http.Error(w, "failed to activate a QR code", http.StatusInternalServerError)
			return
		}
	}
	p := filepath.Join(a.templatesDir, "host.html")
	tmpl, err := template.ParseFiles(p)
	if err != nil {
		log.Printf("template parse error %s: %v", p, err)
		http.Error(w, "failed to load page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, nil); err != nil {
		log.Printf("template execute error %s: %v", p, err)
	}
}

// handleHostRotate invalidates the current QR code and issues a fresh one (FR-007).
func (a *App) handleHostRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, err := a.store.Activate(r.Context(), a.qrWindowTTL); err != nil {
		http.Error(w, "failed to rotate the QR code", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/host", http.StatusSeeOther)
}

// handleLeaderboardPage serves the leaderboard display page, rendering the
// leaderboard template with the configured service URLs so the React
// components connect to the correct microservice endpoints.
func (a *App) handleLeaderboardPage(w http.ResponseWriter, r *http.Request) {
	p := filepath.Join(a.templatesDir, "leaderboard.html")
	tmpl, err := template.ParseFiles(p)
	if err != nil {
		log.Printf("template parse error %s: %v", p, err)
		http.Error(w, "failed to render leaderboard", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		CommitsServiceURL string
		ScoresServiceURL  string
	}{
		CommitsServiceURL: a.commitsServiceURL,
		ScoresServiceURL:  a.scoresServiceURL,
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template execute error %s: %v", p, err)
	}
}

// handlePing returns the process startup ID combined with the current template
// version as JSON. Browsers poll this to detect a redeploy (startupID changes)
// or a template file edit (templateVersion increments) — either causes a reload.
func (a *App) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	id := startupID + "." + strconv.FormatInt(a.templateVersion.Load(), 10)
	fmt.Fprintf(w, `{"id":%q}`, id)
}

// repoURL is the public GitHub repository for this project, encoded into the
// static repo QR code served at /repo-qr.png.
const repoURL = "https://github.com/krisfoster/compose-dev-containers-test-containers"

// handleAuthCheck is an internal-only endpoint called by nginx auth_request to validate
// the cw_grant cookie before forwarding requests to protected upstream routes. It returns
// 200 if the cookie is present and cryptographically valid, 401 otherwise. The nginx
// config marks the corresponding location as "internal" so external clients cannot call
// this endpoint directly.
func (a *App) handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	cookie, err := r.Cookie(gate.GrantCookieName)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if _, err := a.signer.Verify(cookie.Value); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleRepoQRPNG serves a static QR code that encodes the project's GitHub URL.
// Unlike /qr.png it requires no active window and no ngrok tunnel — the target
// URL never changes, so the PNG can be generated fresh on each request cheaply.
func handleRepoQRPNG(w http.ResponseWriter, r *http.Request) {
	png, err := qrcode.RenderPNG(repoURL, 320)
	if err != nil {
		http.Error(w, "failed to render QR code", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(png)
}

// ngrokTunnelsResponse is the shape of ngrok's local inspection API
// (http://ngrok:4040/api/tunnels), trimmed to the fields this app needs.
type ngrokTunnelsResponse struct {
	Tunnels []struct {
		PublicURL string `json:"public_url"`
		Proto     string `json:"proto"`
	} `json:"tunnels"`
}

// discoverPublicHost finds the current public hostname by querying ngrok's own local
// API, per research.md §5 and the precedent set by 001-host-webapp-ngrok.
func (a *App) discoverPublicHost(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.ngrokAPIURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var parsed ngrokTunnelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	for _, t := range parsed.Tunnels {
		if t.Proto == "https" && t.PublicURL != "" {
			return hostOnly(t.PublicURL), nil
		}
	}
	return "", errNoPublicTunnel
}

var errNoPublicTunnel = &noPublicTunnelError{}

type noPublicTunnelError struct{}

func (*noPublicTunnelError) Error() string { return "no https tunnel currently reported by ngrok" }

func hostOnly(rawURL string) string {
	const httpsPrefix = "https://"
	s := rawURL
	if len(s) >= len(httpsPrefix) && s[:len(httpsPrefix)] == httpsPrefix {
		s = s[len(httpsPrefix):]
	}
	return s
}
