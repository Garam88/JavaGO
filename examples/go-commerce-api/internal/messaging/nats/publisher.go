package natsmessaging

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"go-commerce-api/internal/domain"
)

const (
	StreamName              = "ORDER_EVENTS"
	SubjectOrderCreated     = "order.created"
	DurableOrderEventWorker = "order-event-worker"
)

type Publisher struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func Connect(url string) (*nats.Conn, error) {
	return nats.Connect(url, nats.Name("go-commerce-api"))
}

func NewPublisher(nc *nats.Conn) (*Publisher, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	if err := EnsureStream(js); err != nil {
		return nil, err
	}
	return &Publisher{nc: nc, js: js}, nil
}

func EnsureStream(js nats.JetStreamContext) error {
	if _, err := js.StreamInfo(StreamName); err == nil {
		return nil
	}
	_, err := js.AddStream(&nats.StreamConfig{
		Name:     StreamName,
		Subjects: []string{SubjectOrderCreated},
		Storage:  nats.FileStorage,
	})
	return err
}

func (p *Publisher) Publish(ctx context.Context, event domain.OutboxEvent) error {
	if event.Topic == "" {
		return fmt.Errorf("event topic is required")
	}
	if len(event.Payload) == 0 {
		return fmt.Errorf("event payload is required")
	}
	_, err := p.js.Publish(event.Topic, event.Payload, nats.MsgId(event.ID), nats.Context(ctx))
	return err
}

func (p *Publisher) Ping(ctx context.Context) error {
	timeout := time.Second
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 {
			timeout = remaining
		}
	}
	return p.nc.FlushTimeout(timeout)
}

func (p *Publisher) JetStream() nats.JetStreamContext {
	return p.js
}
