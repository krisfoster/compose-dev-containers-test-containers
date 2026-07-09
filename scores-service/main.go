package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"

	"crossywhale/scores-service/internal/scores"
)

type Config struct {
	ListenAddr    string
	RedisAddr     string
	ScoresLimit   int
	PubSubChannel string
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadConfig() Config {
	limit := 10
	if raw := os.Getenv("SCORES_LIMIT"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	return Config{
		ListenAddr:    envOr("SCORES_LISTEN_ADDR", ":8083"),
		RedisAddr:     envOr("REDIS_ADDR", "redis:6379"),
		ScoresLimit:   limit,
		PubSubChannel: envOr("REDIS_PUBSUB_CHANNEL", "leaderboard:score-updated"),
	}
}

func main() {
	cfg := loadConfig()

	client := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	store := scores.NewStore(client, cfg.ScoresLimit, cfg.PubSubChannel)
	handler := scores.NewHandler(store)

	mux := http.NewServeMux()
	mux.Handle("/scores", handler)
	mux.Handle("/scores/stream", handler)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		WriteTimeout: 0, // SSE connections are long-lived
	}

	log.Printf("scores-service listening on %s (redis=%s limit=%d channel=%s)",
		cfg.ListenAddr, cfg.RedisAddr, cfg.ScoresLimit, cfg.PubSubChannel)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("scores-service: %v", err)
	}
}
