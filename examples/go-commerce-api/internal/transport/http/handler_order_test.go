package httptransport

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-commerce-api/internal/domain"
	"go-commerce-api/internal/service"
)

func TestCreateOrder(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"user_id":"u-1","item_id":"sku-1","quantity":2}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"item_id":"sku-1"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestCreateOrder_RejectsUnknownField(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"user_id":"u-1","item_id":"sku-1","quantity":2,"oops":true}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateOrder_RejectsTrailingJSON(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"user_id":"u-1","item_id":"sku-1","quantity":2}{}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateOrder_RejectsLargeBody(t *testing.T) {
	handler := newTestHandler()
	body := strings.NewReader(`{"user_id":"` + strings.Repeat("x", maxJSONBodyBytes) + `","item_id":"sku-1","quantity":2}`)
	req := httptest.NewRequest(http.MethodPost, "/orders", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetItem_NotFound(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/items/missing", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func newTestHandler() http.Handler {
	store := &handlerFakeStore{
		items: map[string]domain.Item{
			"sku-1": {ID: "sku-1", Name: "Go T-Shirt", Stock: 10, PriceCents: 2900},
		},
		orders: make(map[string]domain.Order),
	}
	svc := service.NewOrderService(store, nil)

	mux := http.NewServeMux()
	NewOrderHandler(svc, log.New(ioDiscard{}, "", 0), nil).RegisterRoutes(mux)
	return mux
}

type handlerFakeStore struct {
	items  map[string]domain.Item
	orders map[string]domain.Order
}

func (s *handlerFakeStore) CreateOrder(_ context.Context, command service.CreateOrderCommand) (domain.Order, error) {
	item, ok := s.items[command.Order.ItemID]
	if !ok {
		return domain.Order{}, domain.ErrNotFound
	}
	if item.Stock < command.Order.Quantity {
		return domain.Order{}, domain.ErrOutOfStock
	}
	s.orders[command.Order.ID] = command.Order
	return command.Order, nil
}

func (s *handlerFakeStore) FindOrderByID(_ context.Context, id string) (domain.Order, error) {
	order, ok := s.orders[id]
	if !ok {
		return domain.Order{}, domain.ErrNotFound
	}
	return order, nil
}

func (s *handlerFakeStore) ListOrders(context.Context) ([]domain.Order, error) {
	orders := make([]domain.Order, 0, len(s.orders))
	for _, order := range s.orders {
		orders = append(orders, order)
	}
	return orders, nil
}

func (s *handlerFakeStore) FindItemByID(_ context.Context, id string) (domain.Item, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.Item{}, domain.ErrNotFound
	}
	return item, nil
}

func (s *handlerFakeStore) ListItems(context.Context) ([]domain.Item, error) {
	items := make([]domain.Item, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
