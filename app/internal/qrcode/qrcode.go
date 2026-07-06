// Package qrcode renders the QR code attendees scan to reach Crossy Whale.
package qrcode

import (
	"net/url"

	qr "github.com/skip2/go-qrcode"
)

// BuildPlayURL is the URL a QR code encodes: the public host's /play route carrying
// the current window token (FR-001; see contracts/gate-http-contract.md).
func BuildPlayURL(publicHost, windowID string) string {
	u := url.URL{
		Scheme: "https",
		Host:   publicHost,
		Path:   "/play",
	}
	q := u.Query()
	q.Set("w", windowID)
	u.RawQuery = q.Encode()
	return u.String()
}

// RenderPNG encodes content (typically the output of BuildPlayURL) as a QR code PNG
// of roughly size x size pixels.
func RenderPNG(content string, size int) ([]byte, error) {
	return qr.Encode(content, qr.Medium, size)
}
