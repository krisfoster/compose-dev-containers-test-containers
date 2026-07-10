// Command app serves Crossy Whale. It handles dynamic routes (QR gate, game page,
// host management, leaderboard page, auth check) and renders four Go templates from
// the TEMPLATES_DIR directory. Static game assets (JS, CSS, models) are served
// exclusively by nginx; the app has no file server of its own. See
// specs/002-qr-gated-access/contracts/gate-http-contract.md for the route table.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"crossywhale/app/internal/gate"
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
		store:             store,
		gate:              g,
		signer:            signer,
		templatesDir:      cfg.TemplatesDir,
		ngrokAPIURL:       cfg.NgrokAPIURL,
		qrWindowTTL:       cfg.QRWindowTTL,
		httpClient:        &http.Client{Timeout: 3 * time.Second},
		commitsServiceURL: cfg.CommitsServiceURL,
		scoresServiceURL:  cfg.ScoresServiceURL,
		qrServiceURL:      cfg.QRServiceURL,
		showCommits:       cfg.ShowCommits,
	}

	// Fail fast if any page template is missing — better to surface this at
	// startup than to silently serve 500s on the first request to each page.
	for _, name := range []string{"getting-started.html", "leaderboard.html"} {
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
	WebPort           string
	RedisAddr         string
	GrantCookieSecret string
	QRWindowTTL       time.Duration
	GrantLifetime     time.Duration
	NgrokAPIURL       string
	TemplatesDir      string
	CommitsServiceURL string
	ScoresServiceURL  string
	QRServiceURL      string
	ShowCommits       bool
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
		NgrokAPIURL: envOr("NGROK_API_URL", "http://ngrok:4040/api/tunnels"),
		// TEMPLATES_DIR is the container-side path to the page template .html files,
		// including index.html (the game page). Bind-mounted from ./templates in
		// docker-compose.yml so edits on the host reach the running container
		// immediately for live-reload (015-extract-html-templates).
		TemplatesDir: envOr("TEMPLATES_DIR", "/templates"),
		// CommitsServiceURL is the base URL of the commits microservice, reachable
		// from the browser. Default is localhost:8082 for local dev; override via
		// COMMITS_SERVICE_URL in docker-compose or .env for demo environments.
		CommitsServiceURL: envOr("COMMITS_SERVICE_URL", "http://localhost:8082"),
		// ScoresServiceURL is the base URL of the scores microservice, reachable
		// from the browser. Default is localhost:8083 for local dev; override via
		// SCORES_SERVICE_URL in docker-compose or .env for demo environments.
		ScoresServiceURL: envOr("SCORES_SERVICE_URL", "http://localhost:8083"),
		// QRServiceURL is the internal base URL of the qr microservice, reachable
		// only from within the Compose network. Default is localhost:8084 for local dev;
		// set to http://qr-service:8084 in docker-compose (018-qr-code-microservice).
		QRServiceURL: envOr("QR_SERVICE_URL", "http://localhost:8084"),
		// ShowCommits controls whether the recent commits column is rendered on the
		// leaderboard page. Set COMMITS_SHOW=true to enable; hidden and collapsed by default.
		ShowCommits: os.Getenv("COMMITS_SHOW") == "true",
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

// App holds the dependencies shared by the request handlers.
type App struct {
	store             gate.WindowStore
	gate              *gate.Gate
	signer            *gate.Signer
	templatesDir      string
	ngrokAPIURL       string
	qrWindowTTL       time.Duration
	httpClient        *http.Client
	commitsServiceURL string
	scoresServiceURL  string
	qrServiceURL      string
	showCommits       bool
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
		filepath.Join(a.templatesDir, "index.html"),
		filepath.Join(a.templatesDir, "getting-started.html"),
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

// ungatedMux serves the game with no access check at all (FR-004). The bare "/"
// serves a getting-started landing page rather than the raw game index.html —
// routing players through the explicit "/play" link ensures the gate.Middleware
// cookie check runs before the game loads.
func (a *App) ungatedMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleRootOrAsset)
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
	mux.HandleFunc("/repo-qr.png", a.handleRepoQRPNG)
	mux.HandleFunc("/api/ping", a.handlePing)
	mux.HandleFunc("/leaderboard", a.handleLeaderboardPage)
	mux.HandleFunc("/auth/check", a.handleAuthCheck)
	mux.HandleFunc("/host/rotate", a.handleHostRotate)
	return mux
}

// handlePlayIndex serves the game's index page. The template lives in templatesDir
// alongside the other page templates; static assets (script.js, models, etc.) are
// served by nginx from its own document root.
func (a *App) handlePlayIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(filepath.Join(a.templatesDir, "index.html"))
	if err != nil {
		http.Error(w, "failed to load game", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, nil); err != nil {
		log.Printf("render index.html: %v", err)
	}
}

// handleQRPNG returns the current QR code as a PNG. If no window is currently
// active it auto-activates one, so the first load always produces a valid code.
// PNG rendering is delegated to qr-service (018-qr-code-microservice).
func (a *App) handleQRPNG(w http.ResponseWriter, r *http.Request) {
	windowID, err := a.store.Current(r.Context())
	if err != nil {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	if windowID == "" {
		windowID, err = a.store.Activate(r.Context(), a.qrWindowTTL)
		if err != nil {
			http.Error(w, "failed to activate QR window", http.StatusInternalServerError)
			return
		}
	}

	publicHost, err := a.discoverPublicHost(r.Context())
	if err != nil {
		http.Error(w, "public URL not available yet", http.StatusServiceUnavailable)
		return
	}

	playU := &url.URL{Scheme: "https", Host: publicHost, Path: "/play"}
	q := playU.Query()
	q.Set("w", windowID)
	playU.RawQuery = q.Encode()
	png, err := a.renderQR(r.Context(), playU.String(), 320)
	if err != nil {
		http.Error(w, "QR render service unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

// handleRootOrAsset serves the getting-started page for an exact "/" request and
// 404 for all other paths. Static game assets are served by nginx; the app has no
// file server of its own.
func (a *App) handleRootOrAsset(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
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
		ShowCommits       bool
	}{
		CommitsServiceURL: a.commitsServiceURL,
		ScoresServiceURL:  a.scoresServiceURL,
		ShowCommits:       a.showCommits,
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

// handleHostRotate activates a new QR join window, invalidating the previous one.
// It is called by the leaderboard page's "Refresh QR" button and 60-second auto-rotate
// timer. Returns 204 on success; 500 if the store is unavailable.
func (a *App) handleHostRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if _, err := a.store.Activate(r.Context(), a.qrWindowTTL); err != nil {
		http.Error(w, "failed to rotate window", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleRepoQRPNG serves a static QR code that encodes the project's GitHub URL.
// Unlike /qr.png it requires no active window and no ngrok tunnel — the target
// URL never changes, so the PNG can be generated fresh on each request cheaply.
// PNG rendering is delegated to qr-service (018-qr-code-microservice).
func (a *App) handleRepoQRPNG(w http.ResponseWriter, r *http.Request) {
	png, err := a.renderQR(r.Context(), repoURL, 320)
	if err != nil {
		http.Error(w, "QR render service unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(png)
}

// renderQR calls qr-service to render content as a PNG of roughly size×size pixels.
// It returns the raw PNG bytes or an error if the service is unreachable or returns
// a non-200 status.
func (a *App) renderQR(ctx context.Context, content string, size int) ([]byte, error) {
	reqURL := a.qrServiceURL + "/qr.png?content=" + url.QueryEscape(content) + "&size=" + strconv.Itoa(size)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qr-service returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
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
