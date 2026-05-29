package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"go-commerce-api/internal/domain"
	"go-commerce-api/internal/service"
)

const maxJSONBodyBytes = 1 << 20

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type OrderHandler struct {
	svc    *service.OrderService
	logger *log.Logger
	checks map[string]HealthChecker
}

type createOrderRequest struct {
	UserID   string `json:"user_id"`
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

func NewOrderHandler(svc *service.OrderService, logger *log.Logger, checks map[string]HealthChecker) *OrderHandler {
	return &OrderHandler{svc: svc, logger: logger, checks: checks}
}

func (h *OrderHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /livez", h.liveness)
	mux.HandleFunc("GET /readyz", h.readiness)
	mux.HandleFunc("GET /items", h.listItems)
	mux.HandleFunc("GET /items/{id}", h.getItem)
	mux.HandleFunc("POST /orders", h.createOrder)
	mux.HandleFunc("GET /orders", h.listOrders)
	mux.HandleFunc("GET /orders/{id}", h.getOrder)
}

func (h *OrderHandler) health(w http.ResponseWriter, r *http.Request) {
	h.liveness(w, r)
}

func (h *OrderHandler) liveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *OrderHandler) readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	results := make(map[string]string, len(h.checks))
	ready := true
	for name, check := range h.checks {
		if err := check.Ping(ctx); err != nil {
			results[name] = "error"
			ready = false
			h.logger.Printf("readiness check failed component=%s err=%v", name, err)
			continue
		}
		results[name] = "ok"
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, map[string]any{"status": statusText(ready), "checks": results})
}

func (h *OrderHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req createOrderRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	order, err := h.svc.CreateOrder(r.Context(), service.CreateOrderInput{
		UserID:   req.UserID,
		ItemID:   req.ItemID,
		Quantity: req.Quantity,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"order": order})
}

func (h *OrderHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	order, err := h.svc.GetOrder(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"order": order})
}

func (h *OrderHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.svc.ListOrders(r.Context())
	if err != nil {
		h.logger.Printf("list orders failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (h *OrderHandler) listItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListItems(r.Context())
	if err != nil {
		h.logger.Printf("list items failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *OrderHandler) getItem(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.GetItem(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *OrderHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "resource not found"})
	case errors.Is(err, domain.ErrOutOfStock):
		writeJSON(w, http.StatusConflict, map[string]any{"error": "item stock is not enough"})
	default:
		h.logger.Printf("service error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dest); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return false
	}

	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "request body must contain a single JSON object"})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func statusText(ok bool) string {
	if ok {
		return "ok"
	}
	return "error"
}
