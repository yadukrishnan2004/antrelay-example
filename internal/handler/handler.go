package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yadukrishnan2004/antrelay-example/internal/service"
)

// OrderHandler holds a reference to the service layer.
// It never talks to the store directly — always through the service.
type OrderHandler struct {
	service *service.OrderService
}

// New creates an OrderHandler.
func New(s *service.OrderService) *OrderHandler {
	return &OrderHandler{service: s}
}

// --- Request / Response types ---

// These types exist only in the handler package.
// They represent what comes in over HTTP and what goes out.
// They are deliberately separate from store.Order —
// the HTTP shape and the database shape should be able to evolve independently.

type createOrderRequest struct {
	ItemName string `json:"item_name"`
	Quantity int    `json:"quantity"`
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

// --- Handlers ---

// CreateOrder handles POST /orders
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.service.CreateOrder(req.ItemName, req.Quantity)
	if err != nil {
		// service validation errors are the client's fault — 400
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

// GetOrder handles GET /orders/{id}
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	order, err := h.service.GetOrder(id)
	if err != nil {
		// check if it's a not-found error by inspecting the message
		// a more advanced approach would use sentinel errors — we'll keep it simple for now
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, order)
}

// ListOrders handles GET /orders
func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.service.ListOrders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

// UpdateStatus handles PATCH /orders/{id}
func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateStatus(id, req.Status); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		// invalid status value is a client error
		if strings.Contains(err.Error(), "invalid status") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 204 No Content — update succeeded, nothing to return
	w.WriteHeader(http.StatusNoContent)
}

// DeleteOrder handles DELETE /orders/{id}
func (h *OrderHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.service.DeleteOrder(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 204 No Content — deleted successfully
	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---

// writeJSON serializes v to JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError writes a consistent JSON error response.
// Every error from this server looks the same: {"error": "message"}
// Consistency here matters — API clients can always expect this shape.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}