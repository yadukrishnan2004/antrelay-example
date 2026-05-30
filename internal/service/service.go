package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/yadukrishnan2004/antrelay-example/internal/store"
)

// validStatuses defines the allowed values for order status.
// Using a map here gives O(1) lookup — faster than looping a slice.
var validStatuses = map[string]bool{
	"pending":    true,
	"processing": true,
	"completed":  true,
	"failed":     true,
}

// AntRelayConfig holds the connection details for the AntRelay server.
// The service needs this to queue tasks after creating an order.
type AntRelayConfig struct {
	ServerURL string
	Queue     string
}

// OrderService contains the business logic for orders.
// It owns both the store (for persistence) and the AntRelay config
// (for queuing background tasks).
type OrderService struct {
	store      *store.OrderStore
	antrelay   AntRelayConfig
	httpClient *http.Client
}

// New creates an OrderService with sensible HTTP client defaults.
func New(s *store.OrderStore, cfg AntRelayConfig) *OrderService {
	return &OrderService{
		store:    s,
		antrelay: cfg,
		// always set a timeout on HTTP clients
		// without this, a slow AntRelay server would hang your request forever
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// CreateOrder validates the input, saves the order, and queues a background task.
func (s *OrderService) CreateOrder(itemName string, quantity int) (*store.Order, error) {
	if itemName == "" {
		return nil, fmt.Errorf("service: item_name is required")
	}

	if quantity <= 0 {
		return nil, fmt.Errorf("service: quantity must be greater than zero")
	}

	now := time.Now().UTC()

	o := &store.Order{
		ID:        uuid.NewString(),
		ItemName:  itemName,
		Quantity:  quantity,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Create(o); err != nil {
		return nil, fmt.Errorf("service: failed to save order: %w", err)
	}

	// queue the background task — we do this AFTER saving
	// if AntRelay is down, the order still exists in our database
	// the task submission is best-effort and we log the error but don't fail the request
	if err := s.queueTask(o); err != nil {
		fmt.Printf("service: warning — failed to queue task for order %s: %v\n", o.ID, err)
	}

	return o, nil
}

// GetOrder fetches a single order by ID.
// Returns an explicit error if the order doesn't exist
// so the handler can return a clean 404.
func (s *OrderService) GetOrder(id string) (*store.Order, error) {
	if id == "" {
		return nil, fmt.Errorf("service: id is required")
	}

	o, err := s.store.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("service: failed to fetch order: %w", err)
	}

	if o == nil {
		return nil, fmt.Errorf("service: order %q not found", id)
	}

	return o, nil
}

// ListOrders returns all orders.
func (s *OrderService) ListOrders() ([]*store.Order, error) {
	orders, err := s.store.List()
	if err != nil {
		return nil, fmt.Errorf("service: failed to list orders: %w", err)
	}
	return orders, nil
}

// UpdateStatus validates the new status and updates the order.
func (s *OrderService) UpdateStatus(id string, status string) error {
	if id == "" {
		return fmt.Errorf("service: id is required")
	}

	if status == "" {
		return fmt.Errorf("service: status is required")
	}

	// reject any status value that isn't in our allowed set
	if !validStatuses[status] {
		return fmt.Errorf("service: invalid status %q — must be one of: pending, processing, completed, failed", status)
	}

	if err := s.store.UpdateStatus(id, status); err != nil {
		return fmt.Errorf("service: %w", err)
	}

	return nil
}

// DeleteOrder removes an order by ID.
func (s *OrderService) DeleteOrder(id string) error {
	if id == "" {
		return fmt.Errorf("service: id is required")
	}

	if err := s.store.Delete(id); err != nil {
		return fmt.Errorf("service: %w", err)
	}

	return nil
}

// queueTask submits a processOrder task to the AntRelay server.
// This is the bridge between the orders server and the worker system.
//
// We send the order ID and item details as the task input — the worker
// receives this JSON and can use it to do real processing work.
func (s *OrderService) queueTask(o *store.Order) error {
	// this is the input the worker's processOrder handler will receive
	input, err := json.Marshal(map[string]any{
		"order_id":  o.ID,
		"item_name": o.ItemName,
		"quantity":  o.Quantity,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal task input: %w", err)
	}

	// this matches the request shape antrelay-server expects
	body, err := json.Marshal(map[string]any{
		"function_name": "processOrder",
		"input":         input,
		"max_retries":   3,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal task request: %w", err)
	}

	url := fmt.Sprintf("%s/queues/%s/tasks", s.antrelay.ServerURL, s.antrelay.Queue)

	resp, err := s.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to reach antrelay server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("antrelay server returned unexpected status %d", resp.StatusCode)
	}

	return nil
}