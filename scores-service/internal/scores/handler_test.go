package scores

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeStore struct {
	standings []Standing
	err       error
}

func (f *fakeStore) ReadBest(_ context.Context) ([]Standing, error) {
	return f.standings, f.err
}

func (f *fakeStore) Subscribe(_ context.Context, _ chan<- struct{}) {}
func (f *fakeStore) Write(_ context.Context, _ Entry) error  { return f.err }
func (f *fakeStore) Notify(_ context.Context) error           { return nil }

func newTestHandler(standings []Standing) *Handler {
	return &Handler{store: &fakeStore{standings: standings}}
}

func TestHandlerGetScoresEmpty(t *testing.T) {
	h := newTestHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/scores", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, `"standings":[]`) {
		t.Fatalf("body = %q, want empty standings array (not null)", body)
	}
}

func TestHandlerGetScoresWithStandings(t *testing.T) {
	standings := []Standing{
		{Rank: 1, Name: "alice", Score: 100},
		{Rank: 2, Name: "bob", Score: 50},
	}
	h := newTestHandler(standings)
	req := httptest.NewRequest(http.MethodGet, "/scores", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp standingsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Standings) != 2 {
		t.Fatalf("got %d standings, want 2", len(resp.Standings))
	}
	if resp.Standings[0].Name != "alice" || resp.Standings[1].Name != "bob" {
		t.Fatalf("standings = %+v, want [alice, bob]", resp.Standings)
	}
}

func TestHandlerCORSHeader(t *testing.T) {
	h := newTestHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/scores", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("CORS header = %q, want *", got)
	}
}

func TestHandlerPostScoreCreated(t *testing.T) {
	h := newTestHandler(nil)
	body := strings.NewReader(`{"name":"Alice","score":42}`)
	req := httptest.NewRequest(http.MethodPost, "/scores", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
}

func TestHandlerPostScoreRejectsMissingName(t *testing.T) {
	h := newTestHandler(nil)
	body := strings.NewReader(`{"name":"","score":42}`)
	req := httptest.NewRequest(http.MethodPost, "/scores", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for empty name", rec.Code)
	}
}

func TestHandlerPostScoreRejectsMissingScore(t *testing.T) {
	h := newTestHandler(nil)
	body := strings.NewReader(`{"name":"Alice"}`)
	req := httptest.NewRequest(http.MethodPost, "/scores", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for missing score", rec.Code)
	}
}

func TestHandlerPostScoreRejectsNegativeScore(t *testing.T) {
	h := newTestHandler(nil)
	body := strings.NewReader(`{"name":"Alice","score":-1}`)
	req := httptest.NewRequest(http.MethodPost, "/scores", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for negative score", rec.Code)
	}
}

func TestHandlerJSONContentType(t *testing.T) {
	h := newTestHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/scores", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
}

func TestHandlerSSEContentType(t *testing.T) {
	h := newTestHandler(nil)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/scores/stream")
	if err != nil {
		t.Fatalf("GET /scores/stream: %v", err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
}

func TestHandlerSSEOnConnectEmitsStandingsEvent(t *testing.T) {
	standings := []Standing{{Rank: 1, Name: "alice", Score: 99}}
	h := newTestHandler(standings)

	// Use a pipe-backed server so we can read partial SSE output before the
	// handler blocks waiting for notifications.
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/scores/stream")
	if err != nil {
		t.Fatalf("GET /scores/stream: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	// Read until we find a "standings" event.
	scanner := bufio.NewScanner(resp.Body)
	var eventType, dataLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
		} else if line == "" && eventType == "standings" {
			break
		}
	}

	if eventType != "standings" {
		t.Fatalf("first SSE event type = %q, want standings", eventType)
	}

	var payload standingsResponse
	if err := json.Unmarshal([]byte(dataLine), &payload); err != nil {
		t.Fatalf("parse SSE data as JSON: %v, data: %q", err, dataLine)
	}
	if len(payload.Standings) != 1 || payload.Standings[0].Name != "alice" {
		t.Fatalf("SSE payload standings = %+v, want [{alice 99}]", payload.Standings)
	}
}
