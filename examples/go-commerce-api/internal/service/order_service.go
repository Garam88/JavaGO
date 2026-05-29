package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go-commerce-api/internal/domain"
)

type Store interface {
	CreateOrder(ctx context.Context, command CreateOrderCommand) (domain.Order, error)
	FindOrderByID(ctx context.Context, id string) (domain.Order, error)
	ListOrders(ctx context.Context) ([]domain.Order, error)
	FindItemByID(ctx context.Context, id string) (domain.Item, error)
	ListItems(ctx context.Context) ([]domain.Item, error)
}

type Cache interface {
	GetItem(ctx context.Context, id string) (domain.Item, bool, error)
	SetItem(ctx context.Context, item domain.Item) error
	DeleteItem(ctx context.Context, id string) error
	GetOrder(ctx context.Context, id string) (domain.Order, bool, error)
	SetOrder(ctx context.Context, order domain.Order) error
}

type CreateOrderCommand struct {
	Order domain.Order
	Event domain.OrderCreatedEvent
}

type OrderService struct {
	store       Store
	cache       Cache
	now         func() time.Time
	idGenerator func(prefix string) (string, error)
}

type CreateOrderInput struct {
	UserID   string
	ItemID   string
	Quantity int
}

func NewOrderService(store Store, cache Cache) *OrderService {
	if cache == nil {
		cache = noopCache{}
	}
	return &OrderService{
		store:       store,
		cache:       cache,
		now:         time.Now,
		idGenerator: newID,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, in CreateOrderInput) (domain.Order, error) {
	if err := validateCreateOrderInput(in); err != nil {
		return domain.Order{}, err
	}

	orderID, err := s.idGenerator("ord")
	if err != nil {
		return domain.Order{}, fmt.Errorf("generate order id: %w", err)
	}
	eventID, err := s.idGenerator("evt")
	if err != nil {
		return domain.Order{}, fmt.Errorf("generate event id: %w", err)
	}

	createdAt := s.now().UTC()
	order := domain.Order{
		ID:        orderID,
		UserID:    strings.TrimSpace(in.UserID),
		ItemID:    strings.TrimSpace(in.ItemID),
		Quantity:  in.Quantity,
		Status:    domain.OrderStatusPending,
		CreatedAt: createdAt,
	}
	event := domain.OrderCreatedEvent{
		EventID:   eventID,
		OrderID:   order.ID,
		UserID:    order.UserID,
		ItemID:    order.ItemID,
		Quantity:  order.Quantity,
		CreatedAt: order.CreatedAt,
	}

	saved, err := s.store.CreateOrder(ctx, CreateOrderCommand{Order: order, Event: event})
	if err != nil {
		return domain.Order{}, fmt.Errorf("create order: %w", err)
	}
	_ = s.cache.DeleteItem(ctx, saved.ItemID)
	_ = s.cache.SetOrder(ctx, saved)

	return saved, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Order{}, domain.ErrInvalidInput
	}
	if order, ok, err := s.cache.GetOrder(ctx, id); err == nil && ok {
		return order, nil
	}

	order, err := s.store.FindOrderByID(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	_ = s.cache.SetOrder(ctx, order)
	return order, nil
}

func (s *OrderService) ListOrders(ctx context.Context) ([]domain.Order, error) {
	orders, err := s.store.ListOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	return orders, nil
}

func (s *OrderService) GetItem(ctx context.Context, id string) (domain.Item, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Item{}, domain.ErrInvalidInput
	}
	if item, ok, err := s.cache.GetItem(ctx, id); err == nil && ok {
		return item, nil
	}

	item, err := s.store.FindItemByID(ctx, id)
	if err != nil {
		return domain.Item{}, err
	}
	_ = s.cache.SetItem(ctx, item)
	return item, nil
}

func (s *OrderService) ListItems(ctx context.Context) ([]domain.Item, error) {
	items, err := s.store.ListItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	for _, item := range items {
		_ = s.cache.SetItem(ctx, item)
	}
	return items, nil
}

func validateCreateOrderInput(in CreateOrderInput) error {
	if strings.TrimSpace(in.UserID) == "" {
		return fmt.Errorf("user_id is required: %w", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(in.ItemID) == "" {
		return fmt.Errorf("item_id is required: %w", domain.ErrInvalidInput)
	}
	if in.Quantity <= 0 {
		return fmt.Errorf("quantity must be > 0: %w", domain.ErrInvalidInput)
	}
	return nil
}

func newID(prefix string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(b[:]), nil
}

type noopCache struct{}

func (noopCache) GetItem(context.Context, string) (domain.Item, bool, error) {
	return domain.Item{}, false, nil
}

func (noopCache) SetItem(context.Context, domain.Item) error {
	return nil
}

func (noopCache) DeleteItem(context.Context, string) error {
	return nil
}

func (noopCache) GetOrder(context.Context, string) (domain.Order, bool, error) {
	return domain.Order{}, false, nil
}

func (noopCache) SetOrder(context.Context, domain.Order) error {
	return nil
}
