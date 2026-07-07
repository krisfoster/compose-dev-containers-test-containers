// Command app serves Crossy Whale on two listeners: an ungated one for the
// presenter's local access (docker-compose.yml publishes this port to the host), and
// a gated one that only the ngrok service points at. See
// specs/002-qr-gated-access/contracts/gate-http-contract.md for the full route table.
// The leaderboard score-write API added by 003-leaderboard-score-submission is
// documented separately in
// specs/003-leaderboard-score-submission/contracts/leaderboard-openapi.yaml.
package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/redis/go-redis/v9"

	"crossywhale/app/internal/gate"
	"crossywhale/app/internal/leaderboard"
	"crossywhale/app/internal/qrcode"
)

func main() {
	cfg := loadConfig()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()

	store := gate.NewRedisWindowStore(redisClient)
	signer := gate.NewSigner([]byte(cfg.GrantCookieSecret), cfg.GrantLifetime)
	g := gate.NewGate(store, signer)

	scoreStore := leaderboard.NewRedisScoreStore(redisClient)
	leaderboardHandler := leaderboard.NewHandler(scoreStore, cfg.LeaderboardAPISecret)

	app := &App{
		store:              store,
		gate:               g,
		frontendDir:        cfg.FrontendDir,
		ngrokAPIURL:        cfg.NgrokAPIURL,
		qrWindowTTL:        cfg.QRWindowTTL,
		httpClient:         &http.Client{Timeout: 3 * time.Second},
		leaderboardHandler: leaderboardHandler,
		leaderboardSecret:  cfg.LeaderboardAPISecret,
	}

	errc := make(chan error, 2)
	go func() {
		log.Printf("ungated listener starting on :%s", cfg.WebPort)
		errc <- http.ListenAndServe(":"+cfg.WebPort, app.ungatedMux())
	}()
	go func() {
		log.Printf("gated listener starting on :%s", cfg.GatedPort)
		errc <- http.ListenAndServe(":"+cfg.GatedPort, app.gatedMux())
	}()
	log.Fatal(<-errc)
}

// Config holds the environment-driven settings for the app service.
type Config struct {
	WebPort              string
	GatedPort            string
	RedisAddr            string
	GrantCookieSecret    string
	QRWindowTTL          time.Duration
	GrantLifetime        time.Duration
	NgrokAPIURL          string
	FrontendDir          string
	LeaderboardAPISecret string
}

func loadConfig() Config {
	return Config{
		WebPort:              envOr("APP_WEB_PORT", "8080"),
		GatedPort:            envOr("APP_GATED_PORT", "8081"),
		RedisAddr:            envOr("REDIS_ADDR", "redis:6379"),
		GrantCookieSecret:    envOr("GRANT_COOKIE_SECRET", "dev-only-change-me"),
		QRWindowTTL:          envDurationOr("QR_WINDOW_TTL", 15*time.Minute),
		GrantLifetime:        envDurationOr("GRANT_LIFETIME", 4*time.Hour),
		NgrokAPIURL:          envOr("NGROK_API_URL", "http://ngrok:4040/api/tunnels"),
		FrontendDir:          envOr("FRONTEND_DIR", "/frontend"),
		LeaderboardAPISecret: envOr("LEADERBOARD_API_SECRET", "dev-only-change-me"),
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
	store              gate.WindowStore
	gate               *gate.Gate
	frontendDir        string
	ngrokAPIURL        string
	qrWindowTTL        time.Duration
	httpClient         *http.Client
	leaderboardHandler http.Handler
	leaderboardSecret  string
}

// ungatedMux serves the game with no access check at all (FR-004), plus the
// presenter-only routes for displaying and rotating the QR code (FR-002, FR-007).
func (a *App) ungatedMux() http.Handler {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir(a.frontendDir))
	mux.Handle("/", fileServer)
	mux.HandleFunc("/play", a.handlePlayIndex)
	mux.HandleFunc("/qr.png", a.handleQRPNG)
	mux.HandleFunc("/host", a.handleHost)
	mux.HandleFunc("/host/rotate", a.handleHostRotate)
	mux.Handle("/api/leaderboard/scores", a.leaderboardHandler)
	return mux
}

// gatedMux serves only the game itself, behind the gate decision (FR-003, FR-009).
// /qr.png, /host, and /host/rotate are intentionally absent here — a request for
// them gets the same 404 any undefined route would, per the gate HTTP contract.
// /api/leaderboard/scores is deliberately NOT wrapped in the gate middleware — its
// own credential check (see internal/leaderboard) is this feature's independent
// protection, not tied to QR visitor access (specs/003-leaderboard-score-submission/
// research.md §2).
func (a *App) gatedMux() http.Handler {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir(a.frontendDir))
	mux.Handle("/", a.gate.Middleware(fileServer))
	mux.Handle("/play", a.gate.Middleware(http.HandlerFunc(a.handlePlayIndex)))
	mux.Handle("/api/leaderboard/scores", a.leaderboardHandler)
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
	data := struct{ LeaderboardToken string }{LeaderboardToken: a.leaderboardSecret}
	if err := tmpl.Execute(w, data); err != nil {
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

const hostPageHTML = `<!DOCTYPE html>
<html>
<head><title>Crossy Whale - Host</title></head>
<body>
<h1>Crossy Whale</h1>
<img id="qr" src="/qr.png" alt="QR code to join" width="320" height="320">
<form id="rotate-form" action="/host/rotate" method="post">
<button type="submit">Rotate QR</button>
</form>
<script>
// Progressive enhancement: with JS, rotating swaps the QR image in place with no
// page reload. Without JS, the form still works via its normal POST + redirect.
document.getElementById('rotate-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const resp = await fetch('/host/rotate', { method: 'POST' });
  if (resp.ok) {
    document.getElementById('qr').src = '/qr.png?t=' + Date.now();
  }
});
</script>
</body>
</html>
`

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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(hostPageHTML))
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
