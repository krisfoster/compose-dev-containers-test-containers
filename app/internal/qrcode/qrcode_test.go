package qrcode

import (
	"bytes"
	"image/png"
	"testing"
)

func TestBuildPlayURL(t *testing.T) {
	got := BuildPlayURL("abc123.ngrok-free.app", "window-xyz")
	want := "https://abc123.ngrok-free.app/play?w=window-xyz"
	if got != want {
		t.Fatalf("BuildPlayURL = %q, want %q", got, want)
	}
}

func TestBuildPlayURLEscapesWindowID(t *testing.T) {
	got := BuildPlayURL("example.com", "has space&and=chars")
	want := "https://example.com/play?w=has+space%26and%3Dchars"
	if got != want {
		t.Fatalf("BuildPlayURL = %q, want %q", got, want)
	}
}

func TestRenderPNGProducesValidImage(t *testing.T) {
	data, err := RenderPNG("https://example.com/play?w=abc", 256)
	if err != nil {
		t.Fatalf("RenderPNG: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("RenderPNG returned no data")
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode PNG: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		t.Fatalf("decoded image has non-positive dimensions: %v", bounds)
	}
}

func TestRenderPNGDiffersForDifferentContent(t *testing.T) {
	a, err := RenderPNG("https://example.com/play?w=aaa", 128)
	if err != nil {
		t.Fatalf("RenderPNG(a): %v", err)
	}
	b, err := RenderPNG("https://example.com/play?w=bbb", 128)
	if err != nil {
		t.Fatalf("RenderPNG(b): %v", err)
	}
	if bytes.Equal(a, b) {
		t.Fatal("RenderPNG produced identical output for different content")
	}
}
