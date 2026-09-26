// Package giftcards implements the minimal gift card ledger described in
// the brief: issue, redeem, balance lookup. Optional feature, isolated so
// the sales package can call it without owning gift card business rules.
package giftcards

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"pos/internal/domain"
	"pos/internal/storage"
)

type GiftCard struct {
	ID            int64  `json:"id"`
	Code          string `json:"code"`
	InitialValue  int64  `json:"initial_value"`
	Balance       int64  `json:"balance"`
	Status        string `json:"status"`
	ExpiresAt     *string `json:"expires_at,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type Service struct {
	db *storage.DB
}

func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

func generateCode() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "GC-" + strings.ToUpper(hex.EncodeToString(buf)), nil
}

func (s *Service) Issue(ctx context.Context, value domain.Money) (*GiftCard, error) {
	code, err := generateCode()
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO gift_cards (code, initial_value, balance, status) VALUES (?, ?, ?, 'ACTIVE')`, code, int64(value), int64(value))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if _, err := tx.ExecContext(ctx, `INSERT INTO gift_card_transactions (gift_card_id, type, amount) VALUES (?, 'ISSUE', ?)`, id, int64(value)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.ByCode(ctx, code)
}

func (s *Service) ByCode(ctx context.Context, code string) (*GiftCard, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, code, initial_value, balance, status, expires_at, created_at FROM gift_cards WHERE code = ?`, code)
	var g GiftCard
	if err := row.Scan(&g.ID, &g.Code, &g.InitialValue, &g.Balance, &g.Status, &g.ExpiresAt, &g.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError("GIFT_CARD_NOT_FOUND", "No gift card matches this code", 404)
		}
		return nil, err
	}
	return &g, nil
}

// RedeemTx debits amount from the gift card identified by code, inside the
// caller's transaction (the sale-commit transaction). It never allows a
// negative balance.
func RedeemTx(ctx context.Context, tx *sql.Tx, code string, amount domain.Money, saleID int64) error {
	row := tx.QueryRowContext(ctx, `SELECT id, balance, status FROM gift_cards WHERE code = ?`, code)
	var id, balance int64
	var status string
	if err := row.Scan(&id, &balance, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NewError("GIFT_CARD_NOT_FOUND", "No gift card matches this code", 404)
		}
		return err
	}
	if status != "ACTIVE" {
		return domain.NewError("GIFT_CARD_INACTIVE", fmt.Sprintf("Gift card is %s", strings.ToLower(status)), 400)
	}
	if int64(amount) > balance {
		return domain.NewError("GIFT_CARD_INSUFFICIENT_BALANCE", "Gift card balance is insufficient", 400)
	}
	newBalance := balance - int64(amount)
	newStatus := status
	if newBalance == 0 {
		newStatus = "DEPLETED"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE gift_cards SET balance = ?, status = ? WHERE id = ?`, newBalance, newStatus, id); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO gift_card_transactions (gift_card_id, type, amount, sale_id) VALUES (?, 'REDEEM', ?, ?)`, id, -int64(amount), saleID)
	return err
}
