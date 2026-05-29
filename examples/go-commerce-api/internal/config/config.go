package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort           string
	DBDSN              string
	RedisAddr          string
	NATSURL            string
	RequestTimeout     time.Duration
	CacheTTL           time.Duration
	OutboxPollInterval time.Duration
	WorkerMaxRetries   int
	WorkerBaseBackoff  time.Duration
}

func Load() (Config, error) {
	requestTimeout, err := getenvDuration("REQUEST_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("REQUEST_TIMEOUT: %w", err)
	}
	if requestTimeout <= 0 {
		return Config{}, fmt.Errorf("REQUEST_TIMEOUT must be > 0")
	}

	cacheTTL, err := getenvDuration("CACHE_TTL", 30*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("CACHE_TTL: %w", err)
	}
	if cacheTTL <= 0 {
		return Config{}, fmt.Errorf("CACHE_TTL must be > 0")
	}

	outboxPollInterval, err := getenvDuration("OUTBOX_POLL_INTERVAL", 500*time.Millisecond)
	if err != nil {
		return Config{}, fmt.Errorf("OUTBOX_POLL_INTERVAL: %w", err)
	}
	if outboxPollInterval <= 0 {
		return Config{}, fmt.Errorf("OUTBOX_POLL_INTERVAL must be > 0")
	}

	workerMaxRetries, err := getenvInt("WORKER_MAX_RETRIES", 3)
	if err != nil {
		return Config{}, fmt.Errorf("WORKER_MAX_RETRIES: %w", err)
	}
	if workerMaxRetries < 0 {
		return Config{}, fmt.Errorf("WORKER_MAX_RETRIES must be >= 0")
	}

	workerBaseBackoff, err := getenvDuration("WORKER_BASE_BACKOFF", 200*time.Millisecond)
	if err != nil {
		return Config{}, fmt.Errorf("WORKER_BASE_BACKOFF: %w", err)
	}
	if workerBaseBackoff <= 0 {
		return Config{}, fmt.Errorf("WORKER_BASE_BACKOFF must be > 0")
	}

	cfg := Config{
		HTTPPort:           getenv("HTTP_PORT", "8080"),
		DBDSN:              os.Getenv("DB_DSN"),
		RedisAddr:          getenv("REDIS_ADDR", "localhost:6379"),
		NATSURL:            getenv("NATS_URL", "nats://localhost:4222"),
		RequestTimeout:     requestTimeout,
		CacheTTL:           cacheTTL,
		OutboxPollInterval: outboxPollInterval,
		WorkerMaxRetries:   workerMaxRetries,
		WorkerBaseBackoff:  workerBaseBackoff,
	}
	if cfg.DBDSN == "" {
		return Config{}, errors.New("DB_DSN is required")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}

	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func getenvDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}

	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, err
	}
	return v, nil
}
