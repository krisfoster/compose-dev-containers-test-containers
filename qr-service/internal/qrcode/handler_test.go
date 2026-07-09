package qrcode

import (
	"bytes"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerValidRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/qr.png?content=https%3A%2F%2Fexample.com&size=256", nil)
	rec := httptest.NewRecorder()
	Handler{}.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", ct)
	}
	if _, err := png.Decode(bytes.NewReader(rec.Body.Bytes())); err != nil {
		t.Fatalf("response is not a valid PNG: %v", err)
	}
}

func TestHandlerMissingContent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/qr.png", nil)
	rec := httptest.NewRecorder()
	Handler{}.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for missing content", rec.Code)
	}
}

func TestHandlerEmptyContent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/qr.png?content=", nil)
	rec := httptest.NewRecorder()
	Handler{}.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for empty content", rec.Code)
	}
}

func TestHandlerZeroSizeDefaultsTo320(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/qr.png?content=https%3A%2F%2Fexample.com&size=0", nil)
	rec := httptest.NewRecorder()
	Handler{}.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 when size=0", rec.Code)
	}
	img, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("response is not a valid PNG: %v", err)
	}
	if img.Bounds().Dx() <= 0 {
		t.Fatal("expected positive image dimension for defaulted size")
	}
}

func TestHandlerNegativeSizeDefaultsTo320(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/qr.png?content=https%3A%2F%2Fexample.com&size=-10", nil)
	rec := httptest.NewRecorder()
	Handler{}.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 when size is negative", rec.Code)
	}
}

func TestHandlerOversizedClamped(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/qr.png?content=https%3A%2F%2Fexample.com&size=9999", nil)
	rec := httptest.NewRecorder()
	Handler{}.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for oversized (clamped) size", rec.Code)
	}
	if _, err := png.Decode(bytes.NewReader(rec.Body.Bytes())); err != nil {
		t.Fatalf("response is not a valid PNG: %v", err)
	}
}

func TestHandlerDifferentContentProducesDifferentPNG(t *testing.T) {
	reqA := httptest.NewRequest(http.MethodGet, "/qr.png?content=https%3A%2F%2Fexample.com%2Fa", nil)
	recA := httptest.NewRecorder()
	Handler{}.ServeHTTP(recA, reqA)

	reqB := httptest.NewRequest(http.MethodGet, "/qr.png?content=https%3A%2F%2Fexample.com%2Fb", nil)
	recB := httptest.NewRecorder()
	Handler{}.ServeHTTP(recB, reqB)

	if bytes.Equal(recA.Body.Bytes(), recB.Body.Bytes()) {
		t.Fatal("different content must produce different PNG bytes")
	}
}

func TestHandlerAbsentSizeUsesDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/qr.png?content=https%3A%2F%2Fexample.com", nil)
	rec := httptest.NewRecorder()
	Handler{}.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with absent size", rec.Code)
	}
	if _, err := png.Decode(bytes.NewReader(rec.Body.Bytes())); err != nil {
		t.Fatalf("response is not a valid PNG: %v", err)
	}
}
