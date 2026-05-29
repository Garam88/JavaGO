package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/nats-io/nats.go"

	"go-commerce-api/internal/domain"
	natsmessaging "go-commerce-api/internal/messaging/nats"
)

type ProcessedEventStore interface {
	MarkEventProcessed(ctx context.Context, eventID, topic string, processedAt time.Time) (bool, error)
}

type OrderEventWorker struct {
	sub         *nats.Subscription
	store       ProcessedEventStore
	logger      *log.Logger
	maxRetries  int
	baseBackoff time.Duration
}

func NewOrderEventWorker(
	js nats.JetStreamContext,
	store ProcessedEventStore,
	logger *log.Logger,
	maxRetries int,
	baseBackoff time.Duration,
) (*OrderEventWorker, error) {
	if maxRetries < 0 {
		maxRetries = 0
	}
	sub, err := js.PullSubscribe(
		natsmessaging.SubjectOrderCreated,
		natsmessaging.DurableOrderEventWorker,
		nats.BindStream(natsmessaging.StreamName),
		nats.ManualAck(),
		nats.AckWait(baseBackoff*time.Duration(maxRetries+1)),
		nats.MaxDeliver(maxRetries+1),
	)
	if err != nil {
		return nil, err
	}
	return &OrderEventWorker{
		sub:         sub,
		store:       store,
		logger:      logger,
		maxRetries:  maxRetries,
		baseBackoff: baseBackoff,
	}, nil
}

func (w *OrderEventWorker) Run(ctx context.Context) {
	w.logger.Printf("order event worker start")
	defer w.logger.Printf("order event worker stop")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgs, err := w.sub.Fetch(1, nats.Context(ctx), nats.MaxWait(time.Second))
		if errors.Is(err, nats.ErrTimeout) {
			continue
		}
		if err != nil {
			if ctx.Err() == nil {
				w.logger.Printf("order event fetch failed err=%v", err)
			}
			continue
		}
		for _, msg := range msgs {
			w.handleMessage(ctx, msg)
		}
	}
}

func (w *OrderEventWorker) handleMessage(ctx context.Context, msg *nats.Msg) {
	var event domain.OrderCreatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		w.logger.Printf("order event invalid payload err=%v", err)
		_ = msg.Term()
		return
	}
	if event.EventID == "" || event.OrderID == "" {
		w.logger.Printf("order event missing required id event_id=%q order_id=%q", event.EventID, event.OrderID)
		_ = msg.Term()
		return
	}

	first, err := w.store.MarkEventProcessed(ctx, event.EventID, domain.OutboxTopicOrderCreated, time.Now().UTC())
	if err != nil {
		if w.exhausted(msg) {
			w.logger.Printf("order event failed event_id=%s order_id=%s err=%v", event.EventID, event.OrderID, err)
			_ = msg.Term()
			return
		}
		delay := w.retryDelay(msg)
		w.logger.Printf("order event retry event_id=%s order_id=%s delay=%s err=%v", event.EventID, event.OrderID, delay, err)
		_ = msg.NakWithDelay(delay)
		return
	}

	if !first {
		w.logger.Printf("order event duplicate event_id=%s order_id=%s", event.EventID, event.OrderID)
		_ = msg.Ack()
		return
	}

	w.logger.Printf("order event handled event_id=%s order_id=%s", event.EventID, event.OrderID)
	_ = msg.Ack()
}

func (w *OrderEventWorker) exhausted(msg *nats.Msg) bool {
	metadata, err := msg.Metadata()
	if err != nil {
		return false
	}
	return int(metadata.NumDelivered) > w.maxRetries
}

func (w *OrderEventWorker) retryDelay(msg *nats.Msg) time.Duration {
	metadata, err := msg.Metadata()
	if err != nil || metadata.NumDelivered == 0 {
		return w.baseBackoff
	}
	return w.baseBackoff * time.Duration(metadata.NumDelivered)
}
