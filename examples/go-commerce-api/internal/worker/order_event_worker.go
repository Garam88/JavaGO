package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-commerce-api/internal/domain"
)

type OrderEventWorker struct {
	events       <-chan domain.OrderCreatedEvent
	logger       *log.Logger
	maxRetries   int
	baseBackoff  time.Duration
}

func NewOrderEventWorker(
	events <-chan domain.OrderCreatedEvent,
	logger *log.Logger,
	maxRetries int,
	baseBackoff time.Duration,
) *OrderEventWorker {
	return &OrderEventWorker{
		events:      events,
		logger:      logger,
		maxRetries:  maxRetries,
		baseBackoff: baseBackoff,
	}
}

func (w *OrderEventWorker) Run(ctx context.Context) {
	w.logger.Printf("worker start")
	defer w.logger.Printf("worker stop")

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.events:
			if !ok {
				return
			}
			w.handleWithRetry(ctx, event)
		}
	}
}

func (w *OrderEventWorker) handleWithRetry(ctx context.Context, event domain.OrderCreatedEvent) {
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		err := w.processEvent(ctx, event)
		if err == nil {
			w.logger.Printf("event handled topic=order.created order_id=%s attempts=%d", event.OrderID, attempt+1)
			return
		}

		if attempt == w.maxRetries {
			w.logger.Printf("event failed topic=order.created order_id=%s err=%v", event.OrderID, err)
			return
		}

		wait := w.baseBackoff * time.Duration(attempt+1)
		w.logger.Printf("event retry topic=order.created order_id=%s attempt=%d wait=%s err=%v", event.OrderID, attempt+1, wait, err)

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (w *OrderEventWorker) processEvent(ctx context.Context, event domain.OrderCreatedEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if event.OrderID == "" {
		return fmt.Errorf("empty order id")
	}
	return nil
}
