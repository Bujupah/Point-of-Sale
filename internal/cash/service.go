// Package cash records manual cash drawer movements (cash in/out/drop, float
// adjustments). Sale/refund cash movements are written directly by the
// sales/refunds packages inside their own commit transaction; this package
// covers the standalone, cashier-initiated operations from the Cash
// Register screen.
package cash

import (
	"context"
	"database/sql"
	"errors"

	"pos/internal/audit"
	"pos/internal/domain"
	"pos/internal/security"
	"pos/internal/storage"
)

type Movement struct {
	ID         int64  `json:"id"`
	ShiftID    int64  `json:"shift_id"`
	Type       string `json:"type"`
	Amount     int64  `json:"amount"`
	Reason     string `json:"reason"`
	CashierID  int64  `json:"cashier_id"`
	CashierName string `json:"cashier_name,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type Service struct {
	db  *storage.DB
	log *audit.Logger
}

func NewService(db *storage.DB, log *audit.Logger) *Service {
	return &Service{db: db, log: log}
}

func (s *Service) requireOpenShift(ctx context.Context, registerID int64) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM shifts WHERE register_id = ? AND status = 'OPEN'`, registerID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, domain.NewError("NO_OPEN_SHIFT", "Open a shift before recording cash movements", 409)
	}
	return id, err
}

// Record posts a signed cash movement (positive = cash added to the drawer,
// negative = cash removed) after checking the permission appropriate to its
// direction.
func (s *Service) Record(ctx context.Context, registerID, cashierID int64, perms []string, movementType string, amount int64, reason string) (*Movement, error) {
	if amount <= 0 {
		return nil, domain.NewError("INVALID_INPUT", "Amount must be positive", 400)
	}
	signed := amount
	var requiredPerm, event string
	switch movementType {
	case "CASH_IN":
		requiredPerm, event = security.PermCashIn, audit.EventCashIn
	case "CASH_OUT":
		signed, requiredPerm, event = -amount, security.PermCashOut, audit.EventCashOut
	case "CASH_DROP":
		signed, requiredPerm, event = -amount, security.PermCashOut, audit.EventCashOut
	case "FLOAT_ADJUSTMENT":
		requiredPerm, event = security.PermCashOut, audit.EventCashOut
	default:
		return nil, domain.NewError("INVALID_INPUT", "Unknown cash movement type", 400)
	}
	if !security.HasPermission(perms, requiredPerm) {
		return nil, domain.ErrForbidden
	}
	if reason == "" {
		return nil, domain.NewError("INVALID_INPUT", "A reason is required for cash movements", 400)
	}

	shiftID, err := s.requireOpenShift(ctx, registerID)
	if err != nil {
		return nil, err
	}

	res, err := s.db.ExecContext(ctx, `INSERT INTO cash_movements (shift_id, type, amount, reason, cashier_id) VALUES (?,?,?,?,?)`,
		shiftID, movementType, signed, reason, cashierID)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	s.log.Write(ctx, audit.Event{Event: event, UserID: &cashierID, EntityType: "cash_movement", EntityID: &id,
		Details: map[string]any{"type": movementType, "amount": signed, "reason": reason}})

	return &Movement{ID: id, ShiftID: shiftID, Type: movementType, Amount: signed, Reason: reason, CashierID: cashierID}, nil
}

func (s *Service) List(ctx context.Context, shiftID int64) ([]Movement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT cm.id, cm.shift_id, cm.type, cm.amount, cm.reason, cm.cashier_id, u.name, cm.created_at
		FROM cash_movements cm JOIN users u ON u.id = cm.cashier_id
		WHERE cm.shift_id = ? ORDER BY cm.id DESC`, shiftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Movement{}
	for rows.Next() {
		var m Movement
		if err := rows.Scan(&m.ID, &m.ShiftID, &m.Type, &m.Amount, &m.Reason, &m.CashierID, &m.CashierName, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
