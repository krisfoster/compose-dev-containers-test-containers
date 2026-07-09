// Command qr-service renders QR code PNGs on demand.
// It exposes a single route: GET /qr.png?content=<url>&size=<pixels>.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"crossywhale/qr-service/internal/qrcode"
)

// Config holds environment-driven settings for the qr service.
type Config struct {
	ListenAddr string
}

func loadConfig() Config {
	return Config{
		ListenAddr: envOr("QR_LISTEN_ADDR", ":8084"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	cfg := loadConfig()

	mux := http.NewServeMux()
	mux.Handle("/qr.png", qrcode.Handler{})

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("qr-service starting on %s", cfg.ListenAddr)
	log.Fatal(srv.ListenAndServe())
}
