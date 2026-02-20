package memory

import (
	"context"
	"sort"
	"sync"

	"go-commerce-api/internal/domain"
)

type OrderRepository struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{orders: make(map[string]domain.Order)}
}

func (r *OrderRepository) Save(ctx context.Context, o domain.Order) (domain.Order, error) {
	select {
	case <-ctx.Done():
		return domain.Order{}, ctx.Err()
	default:
	}

	r.mu.Lock()
	r.orders[o.ID] = o
	r.mu.Unlock()
	return o, nil
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (domain.Order, error) {
	select {
	case <-ctx.Done():
		return domain.Order{}, ctx.Err()
	default:
	}

	r.mu.RLock()
	o, ok := r.orders[id]
	r.mu.RUnlock()
	if !ok {
		return domain.Order{}, domain.ErrNotFound
	}
	return o, nil
}

func (r *OrderRepository) List(ctx context.Context) ([]domain.Order, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	r.mu.RLock()
	orders := make([]domain.Order, 0, len(r.orders))
	for _, o := range r.orders {
		orders = append(orders, o)
	}
	r.mu.RUnlock()

	sort.Slice(orders, func(i, j int) bool {
		return orders[i].CreatedAt.Before(orders[j].CreatedAt)
	})
	return orders, nil
}
