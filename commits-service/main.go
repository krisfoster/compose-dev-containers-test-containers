// Command commits-service serves recent git commits as a JSON REST endpoint
// (GET /commits) and a Server-Sent Events stream (GET /commits/stream).
// It reads commits from the git repository mounted at GIT_REPO_PATH and serves
// no HTML — all responses are structured data.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"crossywhale/commits-service/internal/commits"
)

// Config holds environment-driven settings for the commits service.
type Config struct {
	ListenAddr  string
	GitRepoPath string
}

func loadConfig() Config {
	return Config{
		ListenAddr:  envOr("COMMITS_LISTEN_ADDR", ":8082"),
		GitRepoPath: envOr("GIT_REPO_PATH", "/repo"),
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

	handler := commits.NewHandler(cfg.GitRepoPath)

	mux := http.NewServeMux()
	mux.Handle("/commits", handler)
	mux.Handle("/commits/stream", handler)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 0, // no write timeout — SSE streams are long-lived
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("commits-service starting on %s (git repo: %s)", cfg.ListenAddr, cfg.GitRepoPath)
	log.Fatal(srv.ListenAndServe())
}
