// Package qrcode provides an HTTP handler that renders QR code PNGs on demand.
package qrcode

import (
	"net/http"
	"strconv"

	qr "github.com/skip2/go-qrcode"
)

const (
	defaultSize = 320
	minSize     = 64
	maxSize     = 1024
)

// Handler serves GET /qr.png?content=<url>&size=<pixels>.
type Handler struct{}

func (Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	content := r.URL.Query().Get("content")
	if content == "" {
		http.Error(w, "content parameter required", http.StatusBadRequest)
		return
	}

	size := defaultSize
	if raw := r.URL.Query().Get("size"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			size = v
		}
	}
	if size < minSize {
		size = defaultSize
	}
	if size > maxSize {
		size = maxSize
	}

	png, err := qr.Encode(content, qr.Medium, size)
	if err != nil {
		http.Error(w, "failed to render QR code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(png)
}
