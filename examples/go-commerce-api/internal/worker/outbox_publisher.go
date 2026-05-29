package worker

import (
	"context"
	"log"
	"time"

	"go-commerce-api/internal/domain"
)

type OutboxStore interface {
	FetchPendingOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	MarkOutboxPublished(ctx context.Context, id string, publishedAt time.Time) error
	MarkOutboxFailed(ctx context.Context, id string, errText string, maxAttempts int) error
}

type EventPublisher interface {
	Publish(ctx context.Context, event domain.OutboxEvent) error
}

type OutboxPublisher struct {
	store       OutboxStore
	publisher   EventPublisher
	logger      *log.Logger
	interval    time.Duration
	maxAttempts int
	batchSize   int
}

func NewOutboxPublisher(
	store OutboxStore,
	publisher EventPublisher,
	logger *log.Logger,
	interval time.Duration,
	maxAttempts int,
) *OutboxPublisher {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	return &OutboxPublisher{
		store:       store,
		publisher:   publisher,
		logger:      logger,
		interval:    interval,
		maxAttempts: maxAttempts,
		batchSize:   16,
	}
}

func (p *OutboxPublisher) Run(ctx context.Context) {
	p.logger.Printf("outbox publisher start")
	defer p.logger.Printf("outbox publisher stop")

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		p.publishBatch(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *OutboxPublisher) publishBatch(ctx context.Context) {
	events, err := p.store.FetchPendingOutboxEvents(ctx, p.batchSize)
	if err != nil {
		if ctx.Err() == nil {
			p.logger.Printf("outbox fetch failed err=%v", err)
		}
		return
	}

	for _, event := range events {
		if err := p.publisher.Publish(ctx, event); err != nil {
			if markErr := p.store.MarkOutboxFailed(ctx, event.ID, err.Error(), p.maxAttempts); markErr != nil {
				p.logger.Printf("outbox mark failed id=%s publish_err=%v mark_err=%v", event.ID, err, markErr)
				continue
			}
			p.logger.Printf("outbox publish failed id=%s topic=%s attempts=%d err=%v", event.ID, event.Topic, event.Attempts, err)
			continue
		}

		if err := p.store.MarkOutboxPublished(ctx, event.ID, time.Now().UTC()); err != nil {
			p.logger.Printf("outbox mark published failed id=%s err=%v", event.ID, err)
			continue
		}
		p.logger.Printf("outbox published id=%s topic=%s attempts=%d", event.ID, event.Topic, event.Attempts)
	}
}
