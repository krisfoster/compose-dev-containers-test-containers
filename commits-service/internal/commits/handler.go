// Package commits implements the HTTP handlers for the commits microservice.
// It exposes GET /commits (REST JSON) and GET /commits/stream (SSE) over the
// git repository mounted at the configured path.
package commits

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	commitLimit    = 20
	sseInterval    = 30 * time.Second
	maxAuthorLen   = 64
)

// commitEntry is the JSON shape for one git commit returned by the REST and SSE endpoints.
type commitEntry struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Message string `json:"message"`
}

// commitFeed is the top-level wrapper returned by GET /commits and carried in
// SSE event data, matching the schema in contracts/commits-openapi.yaml.
type commitFeed struct {
	Commits []commitEntry `json:"commits"`
}

// Handler serves GET /commits (REST) and GET /commits/stream (SSE).
type Handler struct {
	gitRepoPath string
}

// NewHandler returns a Handler that reads commits from gitRepoPath.
func NewHandler(gitRepoPath string) *Handler {
	return &Handler{gitRepoPath: gitRepoPath}
}

// ServeHTTP routes requests to the appropriate handler method.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	switch {
	case r.Method == http.MethodOptions:
		h.serveOptions(w, r)
	case r.URL.Path == "/commits/stream" && r.Method == http.MethodGet:
		h.serveStream(w, r)
	case r.URL.Path == "/commits" && r.Method == http.MethodGet:
		h.serveList(w, r)
	default:
		http.NotFound(w, r)
	}
}

// serveOptions handles CORS preflight requests.
func (h *Handler) serveOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusNoContent)
}

// serveList handles GET /commits: reads up to commitLimit commits from the
// git repo and returns them as a JSON CommitFeed object, newest first.
func (h *Handler) serveList(w http.ResponseWriter, r *http.Request) {
	feed, err := h.readFeed()
	if err != nil {
		http.Error(w, "git repo unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(feed)
}

// serveStream handles GET /commits/stream: pushes the commit feed as an SSE
// stream. It emits one "commits" event immediately on connect, then re-emits
// every sseInterval. It returns cleanly when the client disconnects.
func (h *Handler) serveStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sendEvent := func() {
		feed, err := h.readFeed()
		if err != nil {
			feed = commitFeed{Commits: []commitEntry{}}
		}
		data, _ := json.Marshal(feed)
		fmt.Fprintf(w, "event: commits\ndata: %s\n\n", data)
		flusher.Flush()
	}

	sendEvent()

	ticker := time.NewTicker(sseInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			sendEvent()
		}
	}
}

// readFeed reads up to commitLimit commits from the git repo at h.gitRepoPath
// and returns them as a CommitFeed, newest first.
// Returns an error only when the git directory cannot be opened (repo missing or
// corrupt). An empty commit log (no commits yet, or HEAD unresolvable in an
// otherwise valid repo) returns a feed with an empty slice — not an error —
// so the leaderboard shows the "no commits" empty state rather than an error.
func (h *Handler) readFeed() (commitFeed, error) {
	repo, err := gogit.PlainOpen(h.gitRepoPath)
	if err != nil {
		return commitFeed{}, err
	}
	ref, err := repo.Head()
	if err != nil {
		// Empty repo (no commits yet) — return empty feed, not an error.
		return commitFeed{Commits: []commitEntry{}}, nil
	}
	iter, err := repo.Log(&gogit.LogOptions{From: ref.Hash()})
	if err != nil {
		return commitFeed{}, err
	}
	defer iter.Close()

	entries := make([]commitEntry, 0, commitLimit)
	_ = iter.ForEach(func(c *object.Commit) error {
		if len(entries) >= commitLimit {
			return fmt.Errorf("done")
		}
		entries = append(entries, commitEntry{
			Hash:    c.Hash.String()[:7],
			Author:  truncate(c.Author.Name, maxAuthorLen),
			Date:    c.Author.When.UTC().Format("2006-01-02 15:04"),
			Message: subjectLine(c.Message),
		})
		return nil
	})

	return commitFeed{Commits: entries}, nil
}

// setCORSHeaders sets permissive CORS headers on every response.
// The commits API is public read-only data; wildcard CORS is safe
// (see research.md §4).
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

// subjectLine returns the first line of a commit message.
func subjectLine(msg string) string {
	for i, ch := range msg {
		if ch == '\n' {
			return msg[:i]
		}
	}
	return msg
}

// truncate returns s truncated to maxLen characters (by Unicode code point).
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
