package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-commerce-api/internal/config"
	"go-commerce-api/internal/messaging/inmemory"
	"go-commerce-api/internal/repository/memory"
	"go-commerce-api/internal/service"
	httptransport "go-commerce-api/internal/transport/http"
	"go-commerce-api/internal/worker"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)

	repo := memory.NewOrderRepository()
	bus := inmemory.NewBus(cfg.EventBufferSize)
	svc := service.NewOrderService(repo, bus)
	handler := httptransport.NewOrderHandler(svc, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	workerRunner := worker.NewOrderEventWorker(
		bus.Subscribe(),
		logger,
		cfg.WorkerMaxRetries,
		cfg.WorkerBaseBackoff,
	)
	go workerRunner.Run(ctx)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      httptransport.WithMiddleware(mux, httptransport.RecoverMiddleware(logger), httptransport.LoggingMiddleware(logger)),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		bus.Close()
	}()

	logger.Printf("server start addr=%s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	logger.Printf("server stopped")
	return nil
}
