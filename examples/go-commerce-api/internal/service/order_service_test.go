package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-commerce-api/internal/domain"
)

func TestCreateOrder(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(store)

	order, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		UserID:   " u-1 ",
		ItemID:   " sku-1 ",
		Quantity: 2,
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if order.ID != "ord_1" {
		t.Fatalf("id = %q, want %q", order.ID, "ord_1")
	}
	if order.UserID != "u-1" || order.ItemID != "sku-1" {
		t.Fatalf("order was not normalized: %+v", order)
	}
	if order.Status != domain.OrderStatusPending {
		t.Fatalf("status = %q, want %q", order.Status, domain.OrderStatusPending)
	}
	if len(store.events) != 1 {
		t.Fatalf("outbox events = %d, want 1", len(store.events))
	}
	if store.events[0].EventID != "evt_2" || store.events[0].OrderID != order.ID {
		t.Fatalf("event = %+v", store.events[0])
	}
}

func TestCreateOrder_InvalidInput(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(store)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		UserID:   "u-1",
		ItemID:   "sku-1",
		Quantity: 0,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestCreateOrder_OutOfStock(t *testing.T) {
	store := newFakeStore()
	store.items["sku-1"] = domain.Item{ID: "sku-1", Stock: 1}
	svc := newTestService(store)

	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		UserID:   "u-1",
		ItemID:   "sku-1",
		Quantity: 2,
	})
	if !errors.Is(err, domain.ErrOutOfStock) {
		t.Fatalf("error = %v, want ErrOutOfStock", err)
	}
	if len(store.orders) != 0 || len(store.events) != 0 {
		t.Fatalf("order or event was created on stock failure")
	}
}

func TestGetOrder_UsesCache(t *testing.T) {
	store := newFakeStore()
	cache := &fakeCache{orders: map[string]domain.Order{
		"ord-1": {ID: "ord-1", UserID: "cached"},
	}}
	svc := NewOrderService(store, cache)

	order, err := svc.GetOrder(context.Background(), "ord-1")
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if order.UserID != "cached" {
		t.Fatalf("order = %+v, want cached order", order)
	}
	if store.findOrderCalls != 0 {
		t.Fatalf("store lookup calls = %d, want 0", store.findOrderCalls)
	}
}

func newTestService(store *fakeStore) *OrderService {
	svc := NewOrderService(store, nil)
	svc.now = func() time.Time { return time.Date(2026, 5, 29, 1, 2, 3, 0, time.UTC) }
	seq := 0
	svc.idGenerator = func(prefix string) (string, error) {
		seq++
		return prefix + "_" + string(rune('0'+seq)), nil
	}
	return svc
}

type fakeStore struct {
	items          map[string]domain.Item
	orders         map[string]domain.Order
	events         []domain.OrderCreatedEvent
	findOrderCalls int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		items: map[string]domain.Item{
			"sku-1": {ID: "sku-1", Name: "Go T-Shirt", Stock: 10, PriceCents: 2900},
		},
		orders: make(map[string]domain.Order),
	}
}

func (s *fakeStore) CreateOrder(_ context.Context, command CreateOrderCommand) (domain.Order, error) {
	item, ok := s.items[command.Order.ItemID]
	if !ok {
		return domain.Order{}, domain.ErrNotFound
	}
	if item.Stock < command.Order.Quantity {
		return domain.Order{}, domain.ErrOutOfStock
	}
	item.Stock -= command.Order.Quantity
	s.items[item.ID] = item
	s.orders[command.Order.ID] = command.Order
	s.events = append(s.events, command.Event)
	return command.Order, nil
}

func (s *fakeStore) FindOrderByID(_ context.Context, id string) (domain.Order, error) {
	s.findOrderCalls++
	order, ok := s.orders[id]
	if !ok {
		return domain.Order{}, domain.ErrNotFound
	}
	return order, nil
}

func (s *fakeStore) ListOrders(context.Context) ([]domain.Order, error) {
	orders := make([]domain.Order, 0, len(s.orders))
	for _, order := range s.orders {
		orders = append(orders, order)
	}
	return orders, nil
}

func (s *fakeStore) FindItemByID(_ context.Context, id string) (domain.Item, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.Item{}, domain.ErrNotFound
	}
	return item, nil
}

func (s *fakeStore) ListItems(context.Context) ([]domain.Item, error) {
	items := make([]domain.Item, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

type fakeCache struct {
	items  map[string]domain.Item
	orders map[string]domain.Order
}

func (c *fakeCache) GetItem(_ context.Context, id string) (domain.Item, bool, error) {
	item, ok := c.items[id]
	return item, ok, nil
}

func (c *fakeCache) SetItem(_ context.Context, item domain.Item) error {
	if c.items == nil {
		c.items = make(map[string]domain.Item)
	}
	c.items[item.ID] = item
	return nil
}

func (c *fakeCache) DeleteItem(_ context.Context, id string) error {
	delete(c.items, id)
	return nil
}

func (c *fakeCache) GetOrder(_ context.Context, id string) (domain.Order, bool, error) {
	order, ok := c.orders[id]
	return order, ok, nil
}

func (c *fakeCache) SetOrder(_ context.Context, order domain.Order) error {
	if c.orders == nil {
		c.orders = make(map[string]domain.Order)
	}
	c.orders[order.ID] = order
	return nil
}
