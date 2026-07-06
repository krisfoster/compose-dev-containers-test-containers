package gate

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestSignerVerifyRoundTrip(t *testing.T) {
	s := NewSigner([]byte("test-secret"), time.Hour)
	g := NewGrant("window-1")

	signed, err := s.Sign(g)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	got, err := s.Verify(signed)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.GrantID != g.GrantID || got.IssuedWindowID != g.IssuedWindowID {
		t.Fatalf("Verify returned %+v, want %+v", got, g)
	}
}

func TestSignerVerifyRejectsTamperedPayload(t *testing.T) {
	s := NewSigner([]byte("test-secret"), time.Hour)
	signed, err := s.Sign(NewGrant("window-1"))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	tampered := signed + "x"
	if _, err := s.Verify(tampered); err != ErrInvalidGrant {
		t.Fatalf("Verify(tampered) = %v, want ErrInvalidGrant", err)
	}
}

func TestSignerVerifyRejectsWrongSecret(t *testing.T) {
	signed, err := NewSigner([]byte("secret-a"), time.Hour).Sign(NewGrant("window-1"))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if _, err := NewSigner([]byte("secret-b"), time.Hour).Verify(signed); err != ErrInvalidGrant {
		t.Fatalf("Verify with wrong secret = %v, want ErrInvalidGrant", err)
	}
}

func TestSignerVerifyRejectsExpiredGrant(t *testing.T) {
	s := NewSigner([]byte("test-secret"), time.Hour)
	g := NewGrant("window-1")
	g.IssuedAt = time.Now().Add(-2 * time.Hour) // older than the 1h lifetime

	signed, err := s.Sign(g)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if _, err := s.Verify(signed); err != ErrInvalidGrant {
		t.Fatalf("Verify(expired) = %v, want ErrInvalidGrant", err)
	}
}

func TestSignerVerifyRejectsGarbage(t *testing.T) {
	s := NewSigner([]byte("test-secret"), time.Hour)
	for _, v := range []string{"", "no-dot-here", "bad.base64!!", "..", "a.b"} {
		if _, err := s.Verify(v); err != ErrInvalidGrant {
			t.Errorf("Verify(%q) = %v, want ErrInvalidGrant", v, err)
		}
	}
}

// A correctly-signed payload that isn't validly base64 (whitebox: crafted with the
// package-private macFor so the signature check passes and the decode failure that
// follows is what's under test).
func TestSignerVerifyRejectsPayloadThatFailsBase64Decode(t *testing.T) {
	s := NewSigner([]byte("test-secret"), time.Hour)
	badPayload := "not-valid-base64!!!"
	mac := s.macFor(badPayload)
	cookieValue := badPayload + "." + base64.RawURLEncoding.EncodeToString(mac)

	if _, err := s.Verify(cookieValue); err != ErrInvalidGrant {
		t.Fatalf("Verify = %v, want ErrInvalidGrant", err)
	}
}

// A correctly-signed payload that decodes as base64 but isn't valid JSON.
func TestSignerVerifyRejectsPayloadThatFailsJSONUnmarshal(t *testing.T) {
	s := NewSigner([]byte("test-secret"), time.Hour)
	encodedPayload := base64.RawURLEncoding.EncodeToString([]byte("not json"))
	mac := s.macFor(encodedPayload)
	cookieValue := encodedPayload + "." + base64.RawURLEncoding.EncodeToString(mac)

	if _, err := s.Verify(cookieValue); err != ErrInvalidGrant {
		t.Fatalf("Verify = %v, want ErrInvalidGrant", err)
	}
}

func TestNewGrantProducesUniqueIDs(t *testing.T) {
	a := NewGrant("window-1")
	b := NewGrant("window-1")
	if a.GrantID == b.GrantID {
		t.Fatalf("NewGrant produced colliding IDs: %q", a.GrantID)
	}
}
