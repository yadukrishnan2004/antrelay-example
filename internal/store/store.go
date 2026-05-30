package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Order represents a single order row in the database.
// Status tracks where the order is in its lifecycle:
// pending → processing → completed or failed
type Order struct {
	ID        string
	ItemName  string
	Quantity  int
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// OrderStore handles all database operations for orders.
type OrderStore struct {
	db *sql.DB
}

// New opens the SQLite database and runs migrations.
func New(dbPath string) (*OrderStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: failed to open database: %w", err)
	}

	s := &OrderStore{db: db}

	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("store: migration failed: %w", err)
	}

	return s, nil
}

// migrate creates the orders table if it doesn't exist.
// We use IF NOT EXISTS so this is safe to call every time the server starts.
func (s *OrderStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
			id         TEXT PRIMARY KEY,
			item_name  TEXT NOT NULL,
			quantity   INTEGER NOT NULL,
			status     TEXT NOT NULL DEFAULT 'pending',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
	`)
	return err
}

// Create inserts a new order into the database.
func (s *OrderStore) Create(o *Order) error {
	_, err := s.db.Exec(`
		INSERT INTO orders (id, item_name, quantity, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, o.ID, o.ItemName, o.Quantity, o.Status, o.CreatedAt, o.UpdatedAt)

	if err != nil {
		return fmt.Errorf("store: failed to create order: %w", err)
	}
	return nil
}

// GetByID fetches a single order by its ID.
// Returns nil if no order exists with that ID — not an error.
func (s *OrderStore) GetByID(id string) (*Order, error) {
	row := s.db.QueryRow(`
		SELECT id, item_name, quantity, status, created_at, updated_at
		FROM orders
		WHERE id = ?
	`, id)

	o := &Order{}
	err := row.Scan(
		&o.ID,
		&o.ItemName,
		&o.Quantity,
		&o.Status,
		&o.CreatedAt,
		&o.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("store: failed to get order: %w", err)
	}

	return o, nil
}

// List returns all orders, newest first.
func (s *OrderStore) List() ([]*Order, error) {
	rows, err := s.db.Query(`
		SELECT id, item_name, quantity, status, created_at, updated_at
		FROM orders
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("store: failed to list orders: %w", err)
	}
	defer rows.Close()

	// we always return a slice, never nil
	// an empty list is [] in JSON, nil would be null — that surprises API clients
	orders := make([]*Order, 0)

	for rows.Next() {
		o := &Order{}
		if err := rows.Scan(
			&o.ID,
			&o.ItemName,
			&o.Quantity,
			&o.Status,
			&o.CreatedAt,
			&o.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: failed to scan order: %w", err)
		}
		orders = append(orders, o)
	}

	// rows.Err() catches any error that happened during iteration
	// rows.Next() silently stops on error — you must check this
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: row iteration error: %w", err)
	}

	return orders, nil
}

// UpdateStatus changes an order's status and updates the timestamp.
func (s *OrderStore) UpdateStatus(id string, status string) error {
	result, err := s.db.Exec(`
		UPDATE orders
		SET status = ?, updated_at = ?
		WHERE id = ?
	`, status, time.Now().UTC(), id)

	if err != nil {
		return fmt.Errorf("store: failed to update order: %w", err)
	}

	// RowsAffected tells us if the id actually existed
	// without this check, updating a non-existent id silently succeeds
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("store: order %q not found", id)
	}

	return nil
}

// Delete removes an order by ID.
func (s *OrderStore) Delete(id string) error {
	result, err := s.db.Exec(`
		DELETE FROM orders WHERE id = ?
	`, id)

	if err != nil {
		return fmt.Errorf("store: failed to delete order: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("store: order %q not found", id)
	}

	return nil
}