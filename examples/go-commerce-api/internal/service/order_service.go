package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"go-commerce-api/internal/domain"
)

type OrderRepository interface {
	Save(ctx context.Context, o domain.Order) (domain.Order, error)
	FindByID(ctx context.Context, id string) (domain.Order, error)
	List(ctx context.Context) ([]domain.Order, error)
}

type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, event domain.OrderCreatedEvent) error
}

type OrderService struct {
	repo       OrderRepository
	publisher  EventPublisher
	now        func() time.Time
	idSequence atomic.Uint64
}

type CreateOrderInput struct {
	UserID   string
	ItemID   string
	Quantity int
}

func NewOrderService(repo OrderRepository, publisher EventPublisher) *OrderService {
	s := &OrderService{
		repo:      repo,
		publisher: publisher,
		now:       time.Now,
	}
	s.idSequence.Store(1000)
	return s
}

func (s *OrderService) CreateOrder(ctx context.Context, in CreateOrderInput) (domain.Order, error) {
	if err := validateCreateOrderInput(in); err != nil {
		return domain.Order{}, err
	}

	id := strconv.FormatUint(s.idSequence.Add(1), 10)
	createdAt := s.now().UTC()

	order := domain.Order{
		ID:        id,
		UserID:    strings.TrimSpace(in.UserID),
		ItemID:    strings.TrimSpace(in.ItemID),
		Quantity:  in.Quantity,
		Status:    domain.OrderStatusPending,
		CreatedAt: createdAt,
	}

	saved, err := s.repo.Save(ctx, order)
	if err != nil {
		return domain.Order{}, fmt.Errorf("save order: %w", err)
	}

	event := domain.OrderCreatedEvent{
		OrderID:   saved.ID,
		UserID:    saved.UserID,
		ItemID:    saved.ItemID,
		Quantity:  saved.Quantity,
		CreatedAt: saved.CreatedAt,
	}
	if err := s.publisher.PublishOrderCreated(ctx, event); err != nil {
		return domain.Order{}, fmt.Errorf("publish order.created: %w", err)
	}

	return saved, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Order{}, domain.ErrInvalidInput
	}
	order, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

func (s *OrderService) ListOrders(ctx context.Context) ([]domain.Order, error) {
	orders, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	return orders, nil
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
