// Package loyalty implements a small, configurable points ledger. The
// earning formula itself lives in settings (points per currency unit spent),
// never hard-coded, so a merchant can change it without a code change.
package loyalty

import (
	"context"
	"database/sql"

	"pos/internal/storage"
)

const (
	TypeEarn   = "EARN"
	TypeRedeem = "REDEEM"
	TypeAdjust = "ADJUST"
	TypeExpire = "EXPIRE"
	TypeRefund = "REFUND"
)

type Transaction struct {
	ID         int64  `json:"id"`
	CustomerID int64  `json:"customer_id"`
	Points     int64  `json:"points"`
	Type       string `json:"type"`
	SaleID     *int64 `json:"sale_id,omitempty"`
	Reason     string `json:"reason"`
	CreatedAt  string `json:"created_at"`
}

type Service struct {
	db *storage.DB
}

func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

// EarnForSale computes points earned for a completed sale using the
// configured "points per currency unit" formula and posts EARN inside the
// sale-commit transaction. ratePer100 is points earned per 100 minor units
// spent (e.g. rate=1 means "1 point per 1.00 of any currency").
func EarnForSale(ctx context.Context, tx *sql.Tx, customerID, saleID, totalMinorUnits, ratePer100 int64) (int64, error) {
	if ratePer100 <= 0 || totalMinorUnits <= 0 {
		return 0, nil
	}
	points := (totalMinorUnits * ratePer100) / 100
	if points <= 0 {
		return 0, nil
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO customer_loyalty_transactions (customer_id, points, type, sale_id, reason) VALUES (?, ?, ?, ?, 'Sale earn')`,
		customerID, points, TypeEarn, saleID); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customers SET loyalty_points_cached = loyalty_points_cached + ? WHERE id = ?`, points, customerID); err != nil {
		return 0, err
	}
	return points, nil
}

// ReverseForSale posts a REFUND ledger entry that removes previously earned
// points proportional to a refund, inside the refund's own transaction.
func ReverseForSale(ctx context.Context, tx *sql.Tx, customerID, saleID, points int64) error {
	if points == 0 {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO customer_loyalty_transactions (customer_id, points, type, sale_id, reason) VALUES (?, ?, ?, ?, 'Refund reversal')`,
		customerID, -points, TypeRefund, saleID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE customers SET loyalty_points_cached = loyalty_points_cached - ? WHERE id = ?`, points, customerID)
	return err
}

// Redeem posts a manual REDEEM entry (e.g. cashier redeems points for a
// discount) as its own atomic operation.
func (s *Service) Redeem(ctx context.Context, customerID, points int64, reason string, userID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO customer_loyalty_transactions (customer_id, points, type, reason, created_by) VALUES (?, ?, ?, ?, ?)`,
		customerID, -points, TypeRedeem, reason, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customers SET loyalty_points_cached = loyalty_points_cached - ? WHERE id = ?`, points, customerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) Adjust(ctx context.Context, customerID, points int64, reason string, userID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO customer_loyalty_transactions (customer_id, points, type, reason, created_by) VALUES (?, ?, ?, ?, ?)`,
		customerID, points, TypeAdjust, reason, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customers SET loyalty_points_cached = loyalty_points_cached + ? WHERE id = ?`, points, customerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) History(ctx context.Context, customerID int64) ([]Transaction, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, customer_id, points, type, sale_id, reason, created_at FROM customer_loyalty_transactions WHERE customer_id = ? ORDER BY id DESC`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.CustomerID, &t.Points, &t.Type, &t.SaleID, &t.Reason, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
