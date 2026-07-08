package leaderboard

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

// CredentialHeader is the HTTP header a score-submission request must carry a valid
// credential in (FR-012). See specs/004-leaderboard-page/contracts/leaderboard-openapi.yaml.
const CredentialHeader = "X-Leaderboard-Token"

// MaxNameLength is the longest accepted player name, in characters, after trimming
// leading/trailing whitespace (FR-003).
const MaxNameLength = 32

// defaultStandingsLimit and maxStandingsLimit bound the GET response size (FR-004 of
// specs/004-leaderboard-page/spec.md) — defaultStandingsLimit applies when the
// `limit` query parameter is absent or invalid; maxStandingsLimit is a hard ceiling
// regardless of what a caller requests.
const (
	defaultStandingsLimit = 20
	maxStandingsLimit     = 50
)

// Handler serves both GET and POST on /api/leaderboard/scores.
type Handler struct {
	store  ScoreStore
	secret string
}

// NewHandler builds a Handler backed by store and secret (the configured
// LEADERBOARD_API_SECRET).
func NewHandler(store ScoreStore, secret string) *Handler {
	return &Handler{store: store, secret: secret}
}

type submitScoreRequest struct {
	Name  string `json:"name"`
	Score *int   `json:"score"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type standing struct {
	Rank  int    `json:"rank"`
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type standingsResponse struct {
	Standings []standing `json:"standings"`
}

// ServeHTTP implements http.Handler for /api/leaderboard/scores: POST submits a score
// (FR-006 through FR-013 of specs/003-leaderboard-score-submission/spec.md) and GET
// retrieves current standings (FR-002 through FR-005, FR-012, FR-013 of
// specs/004-leaderboard-page/spec.md) — see
// specs/004-leaderboard-page/contracts/leaderboard-openapi.yaml for the full contract.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.serveSubmit(w, r)
	case http.MethodGet:
		h.serveList(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// serveSubmit handles POST: a request is rejected, with no write, if it lacks a valid
// credential (401) or fails validation (400).
func (h *Handler) serveSubmit(w http.ResponseWriter, r *http.Request) {
	// Credential check first: an unauthorized caller learns nothing about whether
	// its payload would otherwise have been valid (FR-012).
	if !validCredential(r.Header.Get(CredentialHeader), h.secret) {
		writeError(w, http.StatusUnauthorized, "invalid credential")
		return
	}

	var req submitScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name must not be empty")
		return
	}
	if utf8.RuneCountInString(name) > MaxNameLength {
		writeError(w, http.StatusBadRequest, "name too long")
		return
	}
	if req.Score == nil {
		writeError(w, http.StatusBadRequest, "score is required")
		return
	}
	if *req.Score < 0 {
		writeError(w, http.StatusBadRequest, "score must not be negative")
		return
	}

	if err := h.store.Write(r.Context(), Entry{Name: name, Score: *req.Score}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record score")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]bool{"recorded": true})
}

// serveList handles GET: no credential is required (FR-013). An out-of-range or
// missing/invalid `limit` query parameter is clamped rather than rejected, so this
// endpoint never errors on its own input (FR-004).
func (h *Handler) serveList(w http.ResponseWriter, r *http.Request) {
	limit := defaultStandingsLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > maxStandingsLimit {
		limit = maxStandingsLimit
	}

	entries, err := h.store.Top(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load standings")
		return
	}

	standings := make([]standing, len(entries))
	for i, entry := range entries {
		standings[i] = standing{Rank: i + 1, Name: entry.Name, Score: entry.Score}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(standingsResponse{Standings: standings})
}

// validCredential reports whether got matches want, using a constant-time comparison
// so response timing cannot be used to guess the secret. An empty want always fails
// closed, in case of a misconfigured deployment.
func validCredential(got, want string) bool {
	if want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}
