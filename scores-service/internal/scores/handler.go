package scores

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"
)

// scoreStore is the interface handler needs from the store.
// Using an interface instead of *Store directly lets handler_test.go
// inject a fakeStore without a real Redis connection.
type scoreStore interface {
	ReadBest(ctx context.Context) ([]Standing, error)
	Subscribe(ctx context.Context, ch chan<- struct{})
	Write(ctx context.Context, entry Entry) error
	Notify(ctx context.Context) error
}

// MaxNameLength is the longest accepted player name in characters (trimmed).
const MaxNameLength = 32

type standingsResponse struct {
	Standings []Standing `json:"standings"`
}

// Handler serves GET /scores and GET /scores/stream.
type Handler struct {
	store scoreStore
}

// NewHandler returns a Handler backed by store.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// ServeHTTP routes requests to serveList or serveStream.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	switch r.URL.Path {
	case "/scores":
		switch r.Method {
		case http.MethodGet:
			h.serveList(w, r)
		case http.MethodPost:
			h.serveSubmit(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case "/scores/stream":
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.serveStream(w, r)
	default:
		http.NotFound(w, r)
	}
}

// serveList handles GET /scores — returns JSON standings.
func (h *Handler) serveList(w http.ResponseWriter, r *http.Request) {
	standings, err := h.store.ReadBest(r.Context())
	if err != nil {
		http.Error(w, "failed to load standings", http.StatusInternalServerError)
		return
	}
	if standings == nil {
		standings = []Standing{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(standingsResponse{Standings: standings})
}

// serveStream handles GET /scores/stream — pushes SSE standings events.
func (h *Handler) serveStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	// Emit initial state on connect.
	if err := h.sendStandingsEvent(w, r.Context()); err != nil {
		return
	}
	flusher.Flush()

	// Subscribe to notifications and push on each one.
	notify := make(chan struct{}, 4)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go h.store.Subscribe(ctx, notify)

	for {
		select {
		case <-ctx.Done():
			return
		case <-notify:
			if err := h.sendStandingsEvent(w, ctx); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

type submitScoreRequest struct {
	Name  string `json:"name"`
	Score *int   `json:"score"`
}

// serveSubmit handles POST /scores: validates the payload and writes to Redis.
// Authorization is enforced upstream by nginx auth_request before this runs.
func (h *Handler) serveSubmit(w http.ResponseWriter, r *http.Request) {
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

	if err := h.store.Notify(r.Context()); err != nil {
		log.Printf("scores: notify: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]bool{"recorded": true})
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}

func (h *Handler) sendStandingsEvent(w http.ResponseWriter, ctx context.Context) error {
	standings, err := h.store.ReadBest(ctx)
	if err != nil {
		return err
	}
	if standings == nil {
		standings = []Standing{}
	}
	data, err := json.Marshal(standingsResponse{Standings: standings})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: standings\ndata: %s\n\n", data)
	return err
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
