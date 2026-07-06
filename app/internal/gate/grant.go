package gate

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GrantCookieName is the cookie a visitor's browser carries once they've passed the
// gate. See data-model.md's Visitor Access Grant.
const GrantCookieName = "cw_grant"

// ErrInvalidGrant is returned by VerifyGrant for any cookie value that doesn't verify:
// wrong signature, malformed payload, or expired. Callers treat all of these the same
// way — as "no valid grant" — so the reason is deliberately not distinguished.
var ErrInvalidGrant = errors.New("gate: invalid grant")

// Grant is the payload carried inside a signed cw_grant cookie.
type Grant struct {
	GrantID        string    `json:"grant_id"`
	IssuedWindowID string    `json:"issued_window_id"`
	IssuedAt       time.Time `json:"issued_at"`
}

// NewGrant mints a fresh grant for a visitor who just presented a valid window token.
func NewGrant(issuedWindowID string) Grant {
	return Grant{
		GrantID:        uuid.NewString(),
		IssuedWindowID: issuedWindowID,
		IssuedAt:       time.Now().UTC(),
	}
}

// Signer signs and verifies Grant payloads for the cw_grant cookie, using a
// server-held secret so a visitor cannot forge or alter grant_id/issued_at
// client-side, and a fixed lifetime independent of the QR window's own TTL
// (per FR-008 — a grant survives its originating window rotating or expiring).
type Signer struct {
	secret   []byte
	lifetime time.Duration
}

// NewSigner builds a Signer. secret must be non-empty; lifetime is how long a minted
// grant remains valid regardless of QR window state.
func NewSigner(secret []byte, lifetime time.Duration) *Signer {
	return &Signer{secret: secret, lifetime: lifetime}
}

// Sign encodes and signs a Grant, producing the value to set as the cw_grant cookie.
func (s *Signer) Sign(g Grant) (string, error) {
	payload, err := json.Marshal(g)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	mac := s.macFor(encodedPayload)
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(mac), nil
}

// Verify checks a cw_grant cookie value's signature and lifetime, returning the
// decoded Grant if valid. Any failure — bad signature, malformed payload, or a
// grant older than the configured lifetime — returns ErrInvalidGrant.
func (s *Signer) Verify(cookieValue string) (Grant, error) {
	encodedPayload, encodedMAC, ok := strings.Cut(cookieValue, ".")
	if !ok {
		return Grant{}, ErrInvalidGrant
	}

	gotMAC, err := base64.RawURLEncoding.DecodeString(encodedMAC)
	if err != nil {
		return Grant{}, ErrInvalidGrant
	}
	wantMAC := s.macFor(encodedPayload)
	if !hmac.Equal(gotMAC, wantMAC) {
		return Grant{}, ErrInvalidGrant
	}

	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return Grant{}, ErrInvalidGrant
	}
	var g Grant
	if err := json.Unmarshal(payload, &g); err != nil {
		return Grant{}, ErrInvalidGrant
	}
	if s.lifetime > 0 && time.Since(g.IssuedAt) > s.lifetime {
		return Grant{}, ErrInvalidGrant
	}
	return g, nil
}

// maxAgeSeconds is the Max-Age to set on the grant cookie, matching the Signer's own
// verification lifetime so the browser doesn't hold onto a cookie the server would
// reject anyway.
func (s *Signer) maxAgeSeconds() int {
	return int(s.lifetime.Seconds())
}

func (s *Signer) macFor(encodedPayload string) []byte {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(encodedPayload))
	return mac.Sum(nil)
}
