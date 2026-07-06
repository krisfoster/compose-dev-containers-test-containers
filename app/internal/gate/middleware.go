package gate

import (
	"net/http"
)

// windowQueryParam is the query parameter a freshly scanned QR link carries, e.g.
// https://<public-host>/play?w=<window_id>. See contracts/gate-http-contract.md.
const windowQueryParam = "w"

// rejectBody is served to any visitor who has neither a valid grant nor a valid
// window token. It is deliberately the same response whether no QR window has ever
// been activated at all (fail closed, FR-009) or the presented cookie/token is simply
// invalid — nothing about why access was denied is leaked either way, per
// contracts/gate-http-contract.md's Error responses table.
const rejectBody = `<!DOCTYPE html>
<html>
<head><title>Crossy Whale</title></head>
<body>
<h1>Scan the QR code to play</h1>
<p>This link isn't valid. Find the QR code on display and scan it with your phone's camera.</p>
</body>
</html>
`

// Gate is the public-endpoint access control described in spec.md's User Story 2 and
// contracts/gate-http-contract.md. It wraps an http.Handler so that only a visitor
// holding a valid Grant, or presenting a currently-valid QR window token, can reach it.
type Gate struct {
	store  WindowStore
	signer *Signer
}

// NewGate builds a Gate backed by store (the current QR window) and signer (grant
// cookie signing/verification, including the grant's fixed lifetime).
func NewGate(store WindowStore, signer *Signer) *Gate {
	return &Gate{store: store, signer: signer}
}

// Middleware wraps next so that it is only reached by a visitor with a valid grant,
// minting a fresh grant for a visitor presenting a currently-valid window token, and
// rejecting everyone else with rejectBody (FR-003, FR-005, FR-008, FR-009).
func (g *Gate) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(GrantCookieName); err == nil {
			if _, verifyErr := g.signer.Verify(cookie.Value); verifyErr == nil {
				next.ServeHTTP(w, r)
				return
			}
			// Invalid/tampered cookie falls through to the token check below,
			// exactly like having no cookie at all.
		}

		token := r.URL.Query().Get(windowQueryParam)
		if token != "" {
			current, err := g.store.Current(r.Context())
			if err == nil && current != "" && token == current {
				g.issueGrant(w, current)
				redirectTo := *r.URL
				q := redirectTo.Query()
				q.Del(windowQueryParam)
				redirectTo.RawQuery = q.Encode()
				http.Redirect(w, r, redirectTo.String(), http.StatusFound)
				return
			}
		}

		reject(w)
	})
}

func (g *Gate) issueGrant(w http.ResponseWriter, windowID string) {
	grant := NewGrant(windowID)
	signed, err := g.signer.Sign(grant)
	if err != nil {
		// Signing only fails on JSON marshal errors, which Grant's fixed shape
		// cannot produce; treat as unreachable rather than adding an error path
		// no test could ever exercise honestly.
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     GrantCookieName,
		Value:    signed,
		Path:     "/",
		MaxAge:   g.signer.maxAgeSeconds(),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func reject(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(rejectBody))
}
