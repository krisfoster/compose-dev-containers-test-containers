package scores

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// storeReader is the minimal interface handler needs from the store.
// Using an interface instead of *Store directly lets handler_test.go
// inject a fakeStore without a real Redis connection.
type storeReader interface {
	ReadBest(ctx context.Context) ([]Standing, error)
	Subscribe(ctx context.Context, ch chan<- struct{})
}

type standingsResponse struct {
	Standings []Standing `json:"standings"`
}

// Handler serves GET /scores and GET /scores/stream.
type Handler struct {
	store storeReader
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
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.serveList(w, r)
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
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
