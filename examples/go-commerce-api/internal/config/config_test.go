package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Setenv("DB_DSN", "postgres://commerce:commerce@localhost:5432/commerce?sslmode=disable")
	t.Setenv("REDIS_ADDR", "localhost:6379")
	t.Setenv("NATS_URL", "nats://localhost:4222")
	t.Setenv("REQUEST_TIMEOUT", "3s")
	t.Setenv("CACHE_TTL", "10s")
	t.Setenv("OUTBOX_POLL_INTERVAL", "250ms")
	t.Setenv("WORKER_MAX_RETRIES", "5")
	t.Setenv("WORKER_BASE_BACKOFF", "100ms")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBDSN == "" || cfg.RedisAddr == "" || cfg.NATSURL == "" {
		t.Fatalf("expected external dependency settings, got %+v", cfg)
	}
	if cfg.RequestTimeout != 3*time.Second {
		t.Fatalf("RequestTimeout = %s", cfg.RequestTimeout)
	}
	if cfg.WorkerMaxRetries != 5 {
		t.Fatalf("WorkerMaxRetries = %d", cfg.WorkerMaxRetries)
	}
}

func TestLoad_MissingDBDSN(t *testing.T) {
	t.Setenv("DB_DSN", "")

	if _, err := Load(); err == nil {
		t.Fatalf("expected missing DB_DSN error")
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	t.Setenv("DB_DSN", "postgres://commerce:commerce@localhost:5432/commerce?sslmode=disable")
	t.Setenv("CACHE_TTL", "0s")

	if _, err := Load(); err == nil {
		t.Fatalf("expected invalid CACHE_TTL error")
	}
}
