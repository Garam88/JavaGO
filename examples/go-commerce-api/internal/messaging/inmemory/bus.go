package inmemory

import (
	"context"
	"fmt"

	"go-commerce-api/internal/domain"
)

type Bus struct {
	events chan domain.OrderCreatedEvent
}

func NewBus(bufferSize int) *Bus {
	return &Bus{events: make(chan domain.OrderCreatedEvent, bufferSize)}
}

func (b *Bus) PublishOrderCreated(ctx context.Context, event domain.OrderCreatedEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case b.events <- event:
		return nil
	default:
		return fmt.Errorf("event buffer is full")
	}
}

func (b *Bus) Subscribe() <-chan domain.OrderCreatedEvent {
	return b.events
}

func (b *Bus) Close() {
	defer func() {
		_ = recover()
	}()
	close(b.events)
}
