//go:build integration

package integration

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nats-io/nats.go"

	natsmessaging "go-commerce-api/internal/messaging/nats"
	"go-commerce-api/internal/repository/postgres"
	rediscache "go-commerce-api/internal/repository/redis"
	"go-commerce-api/internal/service"
	"go-commerce-api/internal/worker"
)

func TestOrderFlow(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("set INTEGRATION_TESTS=1 to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	db := openTestDB(t, ctx)
	defer db.Close()
	applyMigration(t, ctx, db)
	resetDatabase(t, ctx, db)

	redisClient := rediscache.NewClient(getenv("REDIS_ADDR", "localhost:6379"))
	defer redisClient.Close()
	if err := redisClient.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}

	nc, err := natsmessaging.Connect(getenv("NATS_URL", "nats://localhost:4222"))
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
	resetStream(t, js)

	store := postgres.NewStore(db)
	cache := rediscache.NewCache(redisClient, time.Minute)
	svc := service.NewOrderService(store, cache)

	order, err := svc.CreateOrder(ctx, service.CreateOrderInput{
		UserID:   "u-1",
		ItemID:   "sku-1",
		Quantity: 2,
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	item, err := store.FindItemByID(ctx, "sku-1")
	if err != nil {
		t.Fatalf("find item: %v", err)
	}
	if item.Stock != 98 {
		t.Fatalf("stock = %d, want 98", item.Stock)
	}
	if exists := redisClient.Exists(ctx, "v1:order:"+order.ID).Val(); exists != 1 {
		t.Fatalf("order cache exists = %d, want 1", exists)
	}

	events, err := store.FetchPendingOutboxEvents(ctx, 1)
	if err != nil {
		t.Fatalf("fetch outbox: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("outbox events = %d, want 1", len(events))
	}

	publisher, err := natsmessaging.NewPublisher(nc)
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	if err := publisher.Publish(ctx, events[0]); err != nil {
		t.Fatalf("publish event: %v", err)
	}
	if err := store.MarkOutboxPublished(ctx, events[0].ID, time.Now().UTC()); err != nil {
		t.Fatalf("mark outbox published: %v", err)
	}

	workerRunner, err := worker.NewOrderEventWorker(publisher.JetStream(), store, log.New(testLogger{t}, "", 0), 3, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}
	workerCtx, stopWorker := context.WithCancel(ctx)
	defer stopWorker()
	go workerRunner.Run(workerCtx)

	eventID := events[0].ID
	eventually(t, 5*time.Second, func() bool {
		var count int
		err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM processed_events WHERE event_id = $1`, eventID).Scan(&count)
		return err == nil && count == 1
	})
}

func openTestDB(t *testing.T, ctx context.Context) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", getenv("DB_DSN", "postgres://commerce:commerce@localhost:5432/commerce?sslmode=disable"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Fatalf("ping db: %v", err)
	}
	return db
}

func applyMigration(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", "001_init.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(raw)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
}

func resetDatabase(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
		TRUNCATE processed_events, outbox_events, orders;
		UPDATE items SET stock = 100, updated_at = NOW() WHERE id = 'sku-1';
	`)
	if err != nil {
		t.Fatalf("reset database: %v", err)
	}
}

func resetStream(t *testing.T, js nats.JetStreamContext) {
	t.Helper()
	if _, err := js.StreamInfo(natsmessaging.StreamName); err == nil {
		if err := js.DeleteStream(natsmessaging.StreamName); err != nil {
			t.Fatalf("delete stream: %v", err)
		}
	}
	if err := natsmessaging.EnsureStream(js); err != nil {
		t.Fatalf("ensure stream: %v", err)
	}
}

func eventually(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("condition was not met within %s", timeout)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type testLogger struct {
	t *testing.T
}

func (l testLogger) Write(p []byte) (int, error) {
	l.t.Logf("%s", p)
	return len(p), nil
}
