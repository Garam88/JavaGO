package service

import (
	"context"
	"testing"

	"go-commerce-api/internal/domain"
	"go-commerce-api/internal/messaging/inmemory"
	"go-commerce-api/internal/repository/memory"
)

func TestCreateOrder(t *testing.T) {
	repo := memory.NewOrderRepository()
	bus := inmemory.NewBus(10)
	svc := NewOrderService(repo, bus)

	order, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		UserID:   "u-1",
		ItemID:   "sku-1",
		Quantity: 2,
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if order.ID == "" {
		t.Fatalf("expected id to be generated")
	}
	if order.Status != domain.OrderStatusPending {
		t.Fatalf("status = %q, want %q", order.Status, domain.OrderStatusPending)
	}
}

func TestCreateOrder_InvalidInput(t *testing.T) {
	repo := memory.NewOrderRepository()
	bus := inmemory.NewBus(10)
	svc := NewOrderService(repo, bus)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		UserID:   "u-1",
		ItemID:   "sku-1",
		Quantity: 0,
	})
	if err == nil {
		t.Fatalf("expected error for invalid input")
	}
}
