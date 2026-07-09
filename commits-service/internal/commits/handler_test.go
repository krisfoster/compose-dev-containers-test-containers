package commits

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// makeFixtureRepo creates an in-memory git repository with n commits and
// returns the path to a temp directory containing a bare clone that the
// Handler can open with PlainOpen.
func makeFixtureRepo(t *testing.T, n int) string {
	t.Helper()
	dir := t.TempDir()

	repo, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	sig := &object.Signature{Name: "Test Author", Email: "test@example.com", When: time.Now()}
	for i := range n {
		_, err = wt.Commit("test commit "+string(rune('A'+i)), &gogit.CommitOptions{
			Author:            sig,
			AllowEmptyCommits: true,
		})
		if err != nil {
			t.Fatalf("commit %d: %v", i, err)
		}
	}
	return dir
}

func TestServeList_WithCommits(t *testing.T) {
	dir := makeFixtureRepo(t, 2)
	h := NewHandler(dir)

	req := httptest.NewRequest(http.MethodGet, "/commits", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var feed commitFeed
	if err := json.NewDecoder(w.Body).Decode(&feed); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(feed.Commits) != 2 {
		t.Fatalf("want 2 commits, got %d", len(feed.Commits))
	}
	// Verify field shapes.
	for _, c := range feed.Commits {
		if len(c.Hash) != 7 {
			t.Errorf("hash %q: want 7 chars, got %d", c.Hash, len(c.Hash))
		}
		if c.Author == "" {
			t.Errorf("commit has empty author")
		}
		if c.Date == "" {
			t.Errorf("commit has empty date")
		}
	}
}

func TestServeList_EmptyRepo(t *testing.T) {
	dir := makeFixtureRepo(t, 0)
	h := NewHandler(dir)

	req := httptest.NewRequest(http.MethodGet, "/commits", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	// An empty repo has no HEAD, but the service should return 200 with an empty
	// commits array — not 503 — so the leaderboard shows the empty-state message.
	if w.Code != http.StatusOK {
		t.Fatalf("empty repo: want 200, got %d", w.Code)
	}
	var feed commitFeed
	if err := json.NewDecoder(w.Body).Decode(&feed); err != nil {
		t.Fatalf("decode empty repo response: %v", err)
	}
	if feed.Commits == nil {
		t.Fatal("empty repo: commits must be a non-nil empty slice, got nil")
	}
	if len(feed.Commits) != 0 {
		t.Fatalf("empty repo: want 0 commits, got %d", len(feed.Commits))
	}
}

func TestServeList_CORSHeader(t *testing.T) {
	dir := makeFixtureRepo(t, 1)
	h := NewHandler(dir)

	req := httptest.NewRequest(http.MethodGet, "/commits", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS header: want *, got %q", got)
	}
}

func TestServeOptions(t *testing.T) {
	h := NewHandler(t.TempDir())

	req := httptest.NewRequest(http.MethodOptions, "/commits", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("preflight: want 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS on preflight: want *, got %q", got)
	}
}

func TestSubjectLine(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"single line", "single line"},
		{"first\nsecond", "first"},
		{"first\n\nsecond", "first"},
		{"", ""},
	}
	for _, tt := range tests {
		got := subjectLine(tt.in)
		if got != tt.want {
			t.Errorf("subjectLine(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 3); got != "hel" {
		t.Errorf("truncate: want hel, got %q", got)
	}
	if got := truncate("hi", 10); got != "hi" {
		t.Errorf("truncate short: want hi, got %q", got)
	}
}
