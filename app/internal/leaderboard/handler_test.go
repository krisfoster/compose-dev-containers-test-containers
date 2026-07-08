package leaderboard_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"crossywhale/app/internal/leaderboard"
	"crossywhale/app/internal/leaderboard/leaderboardtest"
)

const testSecret = "test-leaderboard-secret"

func newTestHandler() (*leaderboard.Handler, *leaderboardtest.FakeScoreStore) {
	store := &leaderboardtest.FakeScoreStore{}
	return leaderboard.NewHandler(store, testSecret), store
}

func doRequest(h *leaderboard.Handler, method, credential string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, "/api/leaderboard/scores", &buf)
	if credential != "" {
		req.Header.Set(leaderboard.CredentialHeader, credential)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// doGetRequest issues a GET, optionally with a raw query string (e.g. "limit=2"), and
// deliberately carries no credential header — the read endpoint requires none (FR-013).
func doGetRequest(h *leaderboard.Handler, rawQuery string) *httptest.ResponseRecorder {
	target := "/api/leaderboard/scores"
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type standingsResponse struct {
	Standings []struct {
		Rank  int    `json:"rank"`
		Name  string `json:"name"`
		Score int    `json:"score"`
	} `json:"standings"`
}

func decodeStandings(t *testing.T, rec *httptest.ResponseRecorder) standingsResponse {
	t.Helper()
	var resp standingsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode standings response: %v, body: %s", err, rec.Body.String())
	}
	return resp
}

// US3: a valid request is accepted and recorded exactly once.
func TestHandlerAcceptsValidSubmission(t *testing.T) {
	h, store := newTestHandler()

	rec := doRequest(h, http.MethodPost, testSecret, map[string]any{"name": "kris", "score": 42})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if store.Len() != 1 {
		t.Fatalf("store has %d entries, want 1", store.Len())
	}
	if got := store.Entries[0]; got.Name != "kris" || got.Score != 42 {
		t.Fatalf("recorded entry = %+v, want {kris 42}", got)
	}
}

// FR-007/FR-008: repeat submissions under the identical name each produce their own
// entry, neither overwriting the other.
func TestHandlerRepeatSubmissionsUnderSameNameDoNotOverwrite(t *testing.T) {
	h, store := newTestHandler()

	doRequest(h, http.MethodPost, testSecret, map[string]any{"name": "kris", "score": 10})
	doRequest(h, http.MethodPost, testSecret, map[string]any{"name": "kris", "score": 20})

	if store.Len() != 2 {
		t.Fatalf("store has %d entries, want 2", store.Len())
	}
	if store.Entries[0].Score != 10 || store.Entries[1].Score != 20 {
		t.Fatalf("entries = %+v, want scores [10, 20] in submission order", store.Entries)
	}
}

// US5/FR-012: a missing credential is rejected with no write.
func TestHandlerRejectsMissingCredential(t *testing.T) {
	h, store := newTestHandler()

	rec := doRequest(h, http.MethodPost, "", map[string]any{"name": "forger", "score": 9999})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if store.Len() != 0 {
		t.Fatalf("store has %d entries, want 0 (no write on rejected credential)", store.Len())
	}
}

// US5/FR-012: an invalid credential is rejected identically to a missing one, with no
// write and no distinguishing information leaked.
func TestHandlerRejectsInvalidCredentialIdenticallyToMissing(t *testing.T) {
	h, store := newTestHandler()

	missing := doRequest(h, http.MethodPost, "", map[string]any{"name": "forger", "score": 9999})
	invalid := doRequest(h, http.MethodPost, "not-the-real-secret", map[string]any{"name": "forger", "score": 9999})

	if invalid.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", invalid.Code, http.StatusUnauthorized)
	}
	if missing.Code != invalid.Code || missing.Body.String() != invalid.Body.String() {
		t.Fatalf("missing-credential response (%d %q) differs from invalid-credential response (%d %q); must be identical",
			missing.Code, missing.Body.String(), invalid.Code, invalid.Body.String())
	}
	if store.Len() != 0 {
		t.Fatalf("store has %d entries, want 0 (no write on rejected credential)", store.Len())
	}
}

// A normal play-through's own submission, carrying the real credential, still
// succeeds — the credential check does not accidentally block legitimate traffic.
func TestHandlerAcceptsRealCredentialAfterRejectingBadOnes(t *testing.T) {
	h, store := newTestHandler()

	doRequest(h, http.MethodPost, "wrong", map[string]any{"name": "kris", "score": 1})
	rec := doRequest(h, http.MethodPost, testSecret, map[string]any{"name": "kris", "score": 1})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if store.Len() != 1 {
		t.Fatalf("store has %d entries, want 1", store.Len())
	}
}

// US5/FR-002, FR-003: validation failures are rejected with no write.
func TestHandlerValidationFailures(t *testing.T) {
	cases := []struct {
		name string
		body map[string]any
	}{
		{"empty name", map[string]any{"name": "", "score": 10}},
		{"whitespace-only name", map[string]any{"name": "   ", "score": 10}},
		{"name too long", map[string]any{"name": "this-name-is-definitely-longer-than-32-characters", "score": 10}},
		{"missing score", map[string]any{"name": "kris"}},
		{"negative score", map[string]any{"name": "kris", "score": -5}},
		{"non-integer score", map[string]any{"name": "kris", "score": 3.5}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, store := newTestHandler()

			rec := doRequest(h, http.MethodPost, testSecret, tc.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if store.Len() != 0 {
				t.Fatalf("store has %d entries, want 0 (no write on validation failure)", store.Len())
			}
		})
	}
}

// A name at exactly the 32-character limit is accepted, and surrounding whitespace is
// trimmed before the limit and emptiness are checked.
func TestHandlerAcceptsNameAtLengthLimitAndTrimsWhitespace(t *testing.T) {
	h, store := newTestHandler()
	exactly32 := "12345678901234567890123456789012"
	if len(exactly32) != leaderboard.MaxNameLength {
		t.Fatalf("test fixture name is %d chars, want %d", len(exactly32), leaderboard.MaxNameLength)
	}

	rec := doRequest(h, http.MethodPost, testSecret, map[string]any{"name": "  kris  ", "score": 7})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if store.Entries[0].Name != "kris" {
		t.Fatalf("stored name = %q, want trimmed %q", store.Entries[0].Name, "kris")
	}

	rec = doRequest(h, http.MethodPost, testSecret, map[string]any{"name": exactly32, "score": 1})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status at exactly %d chars = %d, want %d", leaderboard.MaxNameLength, rec.Code, http.StatusCreated)
	}
}

// GET is now a supported method (the read endpoint added by
// specs/004-leaderboard-page); only genuinely unsupported verbs are rejected.
func TestHandlerRejectsUnsupportedMethod(t *testing.T) {
	h, _ := newTestHandler()

	rec := doRequest(h, http.MethodPut, testSecret, nil)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// FR-008 (specs/004-leaderboard-page/spec.md): an empty store yields an empty,
// non-null standings list, not an error.
func TestHandlerListReturnsEmptyStandingsWhenStoreEmpty(t *testing.T) {
	h, _ := newTestHandler()

	rec := doGetRequest(h, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"standings":[]`) {
		t.Fatalf("body = %q, want an explicit empty array, not null", rec.Body.String())
	}
}

// FR-002, FR-003: standings are ranked highest score first, with 1-based rank.
func TestHandlerListReturnsRankedStandings(t *testing.T) {
	h, store := newTestHandler()
	store.Entries = []leaderboard.Entry{
		{Name: "low", Score: 5},
		{Name: "high", Score: 50},
		{Name: "mid", Score: 20},
	}

	resp := decodeStandings(t, doGetRequest(h, ""))

	if len(resp.Standings) != 3 {
		t.Fatalf("got %d standings, want 3: %+v", len(resp.Standings), resp.Standings)
	}
	wantOrder := []string{"high", "mid", "low"}
	for i, name := range wantOrder {
		if resp.Standings[i].Name != name || resp.Standings[i].Rank != i+1 {
			t.Fatalf("standings[%d] = %+v, want name %q at rank %d", i, resp.Standings[i], name, i+1)
		}
	}
}

// FR-004: an out-of-range limit is clamped, never rejected.
func TestHandlerListClampsOutOfRangeLimit(t *testing.T) {
	h, store := newTestHandler()
	for i := 0; i < 60; i++ {
		store.Entries = append(store.Entries, leaderboard.Entry{Name: "p", Score: i})
	}

	tooHigh := decodeStandings(t, doGetRequest(h, "limit=1000"))
	if len(tooHigh.Standings) != 50 {
		t.Fatalf("limit=1000 returned %d standings, want clamped to 50", len(tooHigh.Standings))
	}

	tooLow := decodeStandings(t, doGetRequest(h, "limit=0"))
	if len(tooLow.Standings) != 1 {
		t.Fatalf("limit=0 returned %d standings, want clamped to 1", len(tooLow.Standings))
	}

	invalid := decodeStandings(t, doGetRequest(h, "limit=not-a-number"))
	if len(invalid.Standings) != 20 {
		t.Fatalf("limit=not-a-number returned %d standings, want the default of 20", len(invalid.Standings))
	}
}

// FR-013: the read endpoint requires no credential — doGetRequest never sends one, so
// a 200 here already proves this, but assert explicitly for clarity.
func TestHandlerListRequiresNoCredential(t *testing.T) {
	h, store := newTestHandler()
	store.Entries = []leaderboard.Entry{{Name: "kris", Score: 1}}

	rec := doGetRequest(h, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (no credential should be required)", rec.Code, http.StatusOK)
	}
}

func TestHandlerListReturns500WhenStoreTopFails(t *testing.T) {
	store := &leaderboardtest.FakeScoreStore{TopErr: errors.New("intentional test failure")}
	h := leaderboard.NewHandler(store, testSecret)

	rec := doGetRequest(h, "")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

// A misconfigured deployment (empty LEADERBOARD_API_SECRET) must fail closed rather
// than accepting an empty credential header as a match.
func TestHandlerFailsClosedWithEmptyConfiguredSecret(t *testing.T) {
	store := &leaderboardtest.FakeScoreStore{}
	h := leaderboard.NewHandler(store, "")

	rec := doRequest(h, http.MethodPost, "", map[string]any{"name": "kris", "score": 1})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if store.Len() != 0 {
		t.Fatalf("store has %d entries, want 0", store.Len())
	}
}

func TestHandlerReturns500WhenStoreWriteFails(t *testing.T) {
	store := &leaderboardtest.FakeScoreStore{WriteErr: errors.New("intentional test failure")}
	h := leaderboard.NewHandler(store, testSecret)

	rec := doRequest(h, http.MethodPost, testSecret, map[string]any{"name": "kris", "score": 1})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
