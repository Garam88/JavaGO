package httptransport

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"go-commerce-api/internal/domain"
	"go-commerce-api/internal/service"
)

type OrderHandler struct {
	svc    *service.OrderService
	logger *log.Logger
}

type createOrderRequest struct {
	UserID   string `json:"user_id"`
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

func NewOrderHandler(svc *service.OrderService, logger *log.Logger) *OrderHandler {
	return &OrderHandler{svc: svc, logger: logger}
}

func (h *OrderHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("POST /orders", h.createOrder)
	mux.HandleFunc("GET /orders", h.listOrders)
	mux.HandleFunc("GET /orders/{id}", h.getOrder)
}

func (h *OrderHandler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *OrderHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req createOrderRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
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
	id := r.PathValue("id")
	order, err := h.svc.GetOrder(r.Context(), id)
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

func (h *OrderHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "order not found"})
	default:
		h.logger.Printf("service error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
