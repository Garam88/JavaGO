package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"go-commerce-api/internal/config"
	natsmessaging "go-commerce-api/internal/messaging/nats"
	"go-commerce-api/internal/repository/postgres"
	rediscache "go-commerce-api/internal/repository/redis"
	"go-commerce-api/internal/service"
	httptransport "go-commerce-api/internal/transport/http"
	"go-commerce-api/internal/worker"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(healthcheck())
	}
	if err := run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func healthcheck() int {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/livez")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := sql.Open("pgx", cfg.DBDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(30 * time.Minute)

	store := postgres.NewStore(db)
	if err := store.Ping(ctx); err != nil {
		return err
	}

	redisClient := rediscache.NewClient(cfg.RedisAddr)
	defer redisClient.Close()
	cache := rediscache.NewCache(redisClient, cfg.CacheTTL)
	if err := cache.Ping(ctx); err != nil {
		return err
	}

	natsConn, err := natsmessaging.Connect(cfg.NATSURL)
	if err != nil {
		return err
	}
	defer natsConn.Close()
	natsPublisher, err := natsmessaging.NewPublisher(natsConn)
	if err != nil {
		return err
	}

	svc := service.NewOrderService(store, cache)
	handler := httptransport.NewOrderHandler(
		svc,
		logger,
		map[string]httptransport.HealthChecker{
			"postgres": store,
			"redis":    cache,
			"nats":     natsPublisher,
		},
	)

	outboxPublisher := worker.NewOutboxPublisher(store, natsPublisher, logger, cfg.OutboxPollInterval, cfg.WorkerMaxRetries+1)
	go outboxPublisher.Run(ctx)

	orderEventWorker, err := worker.NewOrderEventWorker(natsPublisher.JetStream(), store, logger, cfg.WorkerMaxRetries, cfg.WorkerBaseBackoff)
	if err != nil {
		return err
	}
	go orderEventWorker.Run(ctx)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	srv := &http.Server{
		Addr: ":" + cfg.HTTPPort,
		Handler: httptransport.WithMiddleware(
			mux,
			httptransport.RecoverMiddleware(logger),
			httptransport.RequestIDMiddleware(),
			httptransport.LoggingMiddleware(logger),
			httptransport.TimeoutMiddleware(cfg.RequestTimeout),
		),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Printf("server start addr=%s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	logger.Printf("server stopped")
	return nil
}
