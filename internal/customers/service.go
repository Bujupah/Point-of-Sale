// Package customers owns the customer directory. Lifetime totals are cached
// on the row for fast display but are always derived from — and can be
// recomputed from — the auditable sales/loyalty history, never the only
// source of truth (brief §45).
package customers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"pos/internal/domain"
	"pos/internal/storage"
)

type Customer struct {
	ID             int64   `json:"id"`
	CustomerNumber string  `json:"customer_number"`
	Name           string  `json:"name"`
	Phone          string  `json:"phone"`
	Email          string  `json:"email"`
	Notes          string  `json:"notes"`
	LoyaltyPoints  int64   `json:"loyalty_points"`
	VisitCount     int64   `json:"visit_count"`
	LifetimeSpend  int64   `json:"lifetime_spend"`
	LastVisitAt    *string `json:"last_visit_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

type Service struct {
	db *storage.DB
}

func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Search(ctx context.Context, q string, limit int) ([]Customer, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `SELECT id, customer_number, name, phone, email, notes, loyalty_points_cached, visit_count_cached, lifetime_spend_cached, last_visit_at, created_at FROM customers`
	var args []any
	if q != "" {
		query += ` WHERE name LIKE ? OR phone LIKE ? OR email LIKE ? OR customer_number LIKE ?`
		like := "%" + q + "%"
		args = append(args, like, like, like, like)
	}
	query += ` ORDER BY name LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Customer{}
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func scanCustomer(row interface{ Scan(...any) error }) (*Customer, error) {
	var c Customer
	if err := row.Scan(&c.ID, &c.CustomerNumber, &c.Name, &c.Phone, &c.Email, &c.Notes,
		&c.LoyaltyPoints, &c.VisitCount, &c.LifetimeSpend, &c.LastVisitAt, &c.CreatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Customer, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, customer_number, name, phone, email, notes, loyalty_points_cached, visit_count_cached, lifetime_spend_cached, last_visit_at, created_at FROM customers WHERE id = ?`, id)
	c, err := scanCustomer(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

type CustomerInput struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
	Notes string `json:"notes"`
}

func (s *Service) Create(ctx context.Context, in CustomerInput) (*Customer, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, domain.NewError("INVALID_INPUT", "Customer name is required", 400)
	}
	number := fmt.Sprintf("C%d", time.Now().UnixNano()%1000000000)
	res, err := s.db.ExecContext(ctx, `INSERT INTO customers (customer_number, name, phone, email, notes) VALUES (?, ?, ?, ?, ?)`,
		number, in.Name, in.Phone, in.Email, in.Notes)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, in CustomerInput) (*Customer, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE customers SET name=?, phone=?, email=?, notes=? WHERE id=?`,
		in.Name, in.Phone, in.Email, in.Notes, id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s *Service) AddNote(ctx context.Context, customerID int64, note string, userID int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO customer_notes (customer_id, note, created_by) VALUES (?, ?, ?)`, customerID, note, userID)
	return err
}

// History returns the customer's recent sales, most recent first.
func (s *Service) History(ctx context.Context, customerID int64, limit int) ([]map[string]any, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, receipt_number, total, status, created_at FROM sales WHERE customer_id = ? ORDER BY id DESC LIMIT ?`, customerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var receipt, status, createdAt string
		var total int64
		if err := rows.Scan(&id, &receipt, &total, &status, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "receipt_number": receipt, "total": total, "status": status, "created_at": createdAt})
	}
	return out, rows.Err()
}

// RecordVisit bumps visit_count/lifetime_spend/last_visit_at after a
// completed sale. Called from within the sale-commit transaction.
func RecordVisit(ctx context.Context, tx *sql.Tx, customerID int64, spend int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE customers SET
			visit_count_cached = visit_count_cached + 1,
			lifetime_spend_cached = lifetime_spend_cached + ?,
			last_visit_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = ?`, spend, customerID)
	return err
}
