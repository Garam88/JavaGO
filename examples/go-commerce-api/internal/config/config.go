package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort          string
	EventBufferSize   int
	WorkerMaxRetries  int
	WorkerBaseBackoff time.Duration
}

func Load() (Config, error) {
	eventBufferSize, err := getenvInt("EVENT_BUFFER_SIZE", 64)
	if err != nil {
		return Config{}, fmt.Errorf("EVENT_BUFFER_SIZE: %w", err)
	}
	if eventBufferSize < 1 {
		return Config{}, fmt.Errorf("EVENT_BUFFER_SIZE must be >= 1")
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

	return Config{
		HTTPPort:          getenv("HTTP_PORT", "8080"),
		EventBufferSize:   eventBufferSize,
		WorkerMaxRetries:  workerMaxRetries,
		WorkerBaseBackoff: workerBaseBackoff,
	}, nil
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
