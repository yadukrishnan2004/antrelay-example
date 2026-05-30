package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yadukrishnan2004/antrelay-example/internal/store"
)

var validStatuses = map[string]bool{
	"pending":    true,
	"processing": true,
	"completed":  true,
	"failed":     true,
}

type OrderService struct {
	store *store.OrderStore
}

func New(s *store.OrderStore) *OrderService {
	return &OrderService{store: s}
}

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

	return o, nil
}

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

func (s *OrderService) ListOrders() ([]*store.Order, error) {
	orders, err := s.store.List()
	if err != nil {
		return nil, fmt.Errorf("service: failed to list orders: %w", err)
	}
	return orders, nil
}

func (s *OrderService) UpdateStatus(id string, status string) error {
	if id == "" {
		return fmt.Errorf("service: id is required")
	}

	if status == "" {
		return fmt.Errorf("service: status is required")
	}

	if !validStatuses[status] {
		return fmt.Errorf("service: invalid status %q — must be one of: pending, processing, completed, failed", status)
	}

	if err := s.store.UpdateStatus(id, status); err != nil {
		return fmt.Errorf("service: %w", err)
	}

	return nil
}

func (s *OrderService) DeleteOrder(id string) error {
	if id == "" {
		return fmt.Errorf("service: id is required")
	}

	if err := s.store.Delete(id); err != nil {
		return fmt.Errorf("service: %w", err)
	}

	return nil
}