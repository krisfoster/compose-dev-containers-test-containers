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
// The bare "/" serves a getting-started landing page rather than the raw game
// index.html — playing straight from "/" used to silently break score submission
// (it bypassed handlePlayIndex's credential injection, so
// window.__LEADERBOARD_TOKEN__ was left as the unrendered template placeholder and
// every submission was rejected with no visible error). Routing players through the
// landing page's explicit "/play" link avoids that trap entirely.
func (a *App) ungatedMux() http.Handler {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir(a.frontendDir))
	mux.Handle("/", handleRootOrAsset(fileServer))
	mux.HandleFunc("/play", a.handlePlayIndex)
	mux.HandleFunc("/qr.png", a.handleQRPNG)
	mux.HandleFunc("/host", a.handleHost)
	mux.HandleFunc("/host/rotate", a.handleHostRotate)
	mux.Handle("/api/leaderboard/scores", a.leaderboardHandler)
	mux.HandleFunc("/leaderboard", handleLeaderboardPage)
	return mux
}

// gatedMux serves only the game itself, behind the gate decision (FR-003, FR-009).
// /qr.png, /host, and /host/rotate are intentionally absent here — a request for
// them gets the same 404 any undefined route would, per the gate HTTP contract.
// /api/leaderboard/scores and /leaderboard are deliberately NOT wrapped in the gate
// middleware — the former has its own independent credential check on writes and no
// check at all on reads (see internal/leaderboard); the latter has no gating
// requirement of its own (specs/004-leaderboard-page/spec.md FR-011, FR-013). Neither
// is tied to QR visitor access (specs/003-leaderboard-score-submission/research.md §2).
func (a *App) gatedMux() http.Handler {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir(a.frontendDir))
	mux.Handle("/", a.gate.Middleware(fileServer))
	mux.Handle("/play", a.gate.Middleware(http.HandlerFunc(a.handlePlayIndex)))
	mux.Handle("/api/leaderboard/scores", a.leaderboardHandler)
	mux.HandleFunc("/leaderboard", handleLeaderboardPage)
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

// gettingStartedPageHTML is served at the bare "/" on the ungated listener — a small
// landing page linking to the three local presenter destinations, so nobody lands on
// (or accidentally plays through) the raw, un-rendered game file directly.
const gettingStartedPageHTML = `<!DOCTYPE html>
<html>
<head>
<title>Crossy Whale</title>
<style>
  body { font-family: sans-serif; margin: 0; padding: 2rem 1rem; }
  h1 { margin-top: 0; }
  a.button { display: block; text-align: center; padding: 0.75rem 1rem; margin: 0.75rem 0; border-radius: 0.5rem; text-decoration: none; font-weight: bold; color: #fff; }
  a.play { background: #1f6feb; }
  a.host { background: #6e7681; }
  a.leaderboard { background: #2ea043; }
  .layout { display: grid; grid-template-columns: 1fr 1fr; gap: 2rem; align-items: start; max-width: 800px; margin: 0 auto; }
  .qr-col { display: flex; flex-direction: column; align-items: center; gap: 0.5rem; }
  .qr-col img { border-radius: 0.5rem; background: #fff; max-width: 100%; height: auto; }
  .qr-col p { margin: 0; font-size: 0.85rem; color: #6e7681; }
  @media (max-width: 600px) { .layout { grid-template-columns: 1fr; } }
</style>
</head>
<body>
<h1>Crossy Whale</h1>
<div class="layout">
  <div class="nav-col">
    <a class="button play" href="/play">Play the game</a>
    <a class="button host" href="/host">Host: show the QR code</a>
    <a class="button leaderboard" href="/leaderboard">View the leaderboard</a>
  </div>
  <div class="qr-col" id="qr-col">
    <img id="qr-img" src="/qr.png" alt="QR code to join" width="280" height="280"
         style="display:none"
         onload="this.style.display='';document.getElementById('qr-hint').style.display='none';document.getElementById('qr-caption').style.display=''">
    <p id="qr-hint" style="color:#aaa;font-size:0.85rem;text-align:center;max-width:220px;margin:0">
      Visit <a href="/host">/host</a> to activate the QR code
    </p>
    <p id="qr-caption" style="display:none">Scan to play on your phone</p>
  </div>
</div>
<script>
(function () {
  var img = document.getElementById('qr-img');
  setInterval(function () {
    var next = new Image();
    next.onload = function () {
      img.src = next.src;
      img.style.display = '';
      var hint = document.getElementById('qr-hint');
      if (hint) hint.style.display = 'none';
      var caption = document.getElementById('qr-caption');
      if (caption) caption.style.display = '';
    };
    next.src = '/qr.png?t=' + Date.now();
  }, 4000);
})();
</script>
</body>
</html>
`

// handleRootOrAsset serves gettingStartedPageHTML for an exact "/" request and
// delegates everything else (script.js, style.css, model assets, ...) to assets — the
// same catch-all fileServer /play's rendered page depends on for its own
// root-relative asset references.
func handleRootOrAsset(assets http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			assets.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(gettingStartedPageHTML))
	}
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

// leaderboardPageHTML is the leaderboard display page (specs/004-leaderboard-page),
// meant for an unattended wall/booth display. It fetches GET /api/leaderboard/scores
// once on load (FR-002 through FR-004), then keeps polling it on an interval for as
// long as the page stays open (FR-006) — a failed poll leaves the currently rendered
// standings untouched rather than clearing or erroring (FR-007), and a poll returning
// zero standings shows an explicit empty-state message (FR-008) rather than an empty
// list. No credential is sent or required (FR-011, FR-013).
const leaderboardPageHTML = `<!DOCTYPE html>
<html>
<head>
<title>Crossy Whale - Leaderboard</title>
<style>
  body { font-family: sans-serif; background: #0b1b2b; color: #fff; margin: 0; padding: 2rem; }
  h1 { text-align: center; }
  #standings { list-style: none; padding: 0; max-width: 480px; margin: 1.5rem auto 0; }
  #standings li { display: flex; align-items: baseline; gap: 0.75rem; padding: 0.5rem 1rem; border-bottom: 1px solid rgba(255, 255, 255, 0.15); }
  #standings .rank { opacity: 0.6; min-width: 2.5rem; }
  #standings .name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  #standings .score { font-weight: bold; }
  #status { text-align: center; opacity: 0.75; }
</style>
</head>
<body>
<h1>Crossy Whale Leaderboard</h1>
<p id="status">Loading standings&hellip;</p>
<ul id="standings"></ul>
<script>
(function () {
  var POLL_INTERVAL_MS = 4000;
  var statusEl = document.getElementById('status');
  var listEl = document.getElementById('standings');
  var hasRenderedOnce = false;

  function escapeText(value) {
    var div = document.createElement('div');
    div.textContent = value;
    return div.innerHTML;
  }

  function render(standings) {
    if (standings.length === 0) {
      listEl.innerHTML = '';
      statusEl.textContent = 'No scores yet — be the first to play!';
      statusEl.style.display = '';
      return;
    }
    statusEl.style.display = 'none';
    listEl.innerHTML = standings.map(function (s) {
      return '<li><span class="rank">#' + s.rank + '</span>' +
        '<span class="name">' + escapeText(s.name) + '</span>' +
        '<span class="score">' + s.score + '</span></li>';
    }).join('');
  }

  function refresh() {
    fetch('/api/leaderboard/scores')
      .then(function (resp) {
        if (!resp.ok) { throw new Error('leaderboard fetch failed: ' + resp.status); }
        return resp.json();
      })
      .then(function (data) {
        render(data.standings || []);
        hasRenderedOnce = true;
      })
      .catch(function () {
        // FR-007: on failure, leave whatever is already rendered alone and retry on
        // the next interval tick. Only the very first load has nothing to fall back
        // to, so the loading message simply stays up until a later poll succeeds.
        if (!hasRenderedOnce) {
          statusEl.textContent = 'Loading standings…';
        }
      });
  }

  refresh();
  setInterval(refresh, POLL_INTERVAL_MS);
})();
</script>
</body>
</html>
`

// handleLeaderboardPage serves the leaderboard display page. It has no dependency on
// App state — the page's own script does all the data fetching client-side — so it is
// a plain function rather than an App method.
func handleLeaderboardPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(leaderboardPageHTML))
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
