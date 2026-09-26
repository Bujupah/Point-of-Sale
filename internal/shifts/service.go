// Package shifts owns the shift lifecycle (open → closing → closed), cash
// drawer expected/counted reconciliation, and Z-report generation.
package shifts

import (
	"context"
	"database/sql"
	"errors"

	"pos/internal/audit"
	"pos/internal/domain"
	"pos/internal/security"
	"pos/internal/storage"
)

type Shift struct {
	ID            int64   `json:"id"`
	RegisterID    int64   `json:"register_id"`
	RegisterName  string  `json:"register_name,omitempty"`
	CashierID     int64   `json:"cashier_id"`
	CashierName   string  `json:"cashier_name,omitempty"`
	OpeningFloat  int64   `json:"opening_float"`
	OpeningAt     string  `json:"opening_at"`
	ClosingAt     *string `json:"closing_at,omitempty"`
	Status        string  `json:"status"`
	ExpectedCash  *int64  `json:"expected_cash,omitempty"`
	CountedCash   *int64  `json:"counted_cash,omitempty"`
	Difference    *int64  `json:"difference,omitempty"`
	Notes         string  `json:"notes"`
}

type Service struct {
	db  *storage.DB
	log *audit.Logger
}

func NewService(db *storage.DB, log *audit.Logger) *Service {
	return &Service{db: db, log: log}
}

func (s *Service) Open(ctx context.Context, registerID, cashierID int64, openingFloat int64, perms []string) (*Shift, error) {
	if !security.HasPermission(perms, security.PermShiftOpen) {
		return nil, domain.ErrForbidden
	}
	var existing int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM shifts WHERE register_id = ? AND status = 'OPEN'`, registerID).Scan(&existing)
	if err == nil {
		return nil, domain.NewError("SHIFT_ALREADY_OPEN", "A shift is already open for this register", 409)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO shifts (register_id, cashier_id, opening_float, status) VALUES (?, ?, ?, 'OPEN')`,
		registerID, cashierID, openingFloat)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	s.log.Write(ctx, audit.Event{Event: audit.EventShiftOpen, UserID: &cashierID, EntityType: "shift", EntityID: &id,
		Details: map[string]any{"opening_float": openingFloat}})
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id int64) (*Shift, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT sh.id, sh.register_id, r.name, sh.cashier_id, u.name, sh.opening_float, sh.opening_at, sh.closing_at, sh.status, sh.expected_cash, sh.counted_cash, sh.difference, sh.notes
		FROM shifts sh JOIN registers r ON r.id = sh.register_id JOIN users u ON u.id = sh.cashier_id
		WHERE sh.id = ?`, id)
	sh, err := scanShift(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return sh, nil
}

func (s *Service) CurrentForRegister(ctx context.Context, registerID int64) (*Shift, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT sh.id, sh.register_id, r.name, sh.cashier_id, u.name, sh.opening_float, sh.opening_at, sh.closing_at, sh.status, sh.expected_cash, sh.counted_cash, sh.difference, sh.notes
		FROM shifts sh JOIN registers r ON r.id = sh.register_id JOIN users u ON u.id = sh.cashier_id
		WHERE sh.register_id = ? AND sh.status = 'OPEN'`, registerID)
	sh, err := scanShift(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError("NO_OPEN_SHIFT", "No open shift for this register", 404)
		}
		return nil, err
	}
	return sh, nil
}

func scanShift(row interface{ Scan(...any) error }) (*Shift, error) {
	var sh Shift
	if err := row.Scan(&sh.ID, &sh.RegisterID, &sh.RegisterName, &sh.CashierID, &sh.CashierName, &sh.OpeningFloat,
		&sh.OpeningAt, &sh.ClosingAt, &sh.Status, &sh.ExpectedCash, &sh.CountedCash, &sh.Difference, &sh.Notes); err != nil {
		return nil, err
	}
	return &sh, nil
}

// ExpectedCash computes opening_float + every signed cash_movements.amount
// for the shift — the amount that should physically be in the drawer.
func (s *Service) ExpectedCash(ctx context.Context, shiftID int64) (int64, error) {
	var openingFloat int64
	if err := s.db.QueryRowContext(ctx, `SELECT opening_float FROM shifts WHERE id = ?`, shiftID).Scan(&openingFloat); err != nil {
		return 0, err
	}
	var movementSum int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM cash_movements WHERE shift_id = ?`, shiftID).Scan(&movementSum); err != nil {
		return 0, err
	}
	return openingFloat + movementSum, nil
}

func (s *Service) Close(ctx context.Context, shiftID, cashierID, countedCash int64, notes string, perms []string) (*Shift, error) {
	if !security.HasPermission(perms, security.PermShiftClose) {
		return nil, domain.ErrForbidden
	}
	sh, err := s.Get(ctx, shiftID)
	if err != nil {
		return nil, err
	}
	if sh.Status != "OPEN" {
		return nil, domain.NewError("SHIFT_NOT_OPEN", "This shift is not open", 400)
	}
	expected, err := s.ExpectedCash(ctx, shiftID)
	if err != nil {
		return nil, err
	}
	diff := countedCash - expected

	if _, err := s.db.ExecContext(ctx, `
		UPDATE shifts SET status = 'CLOSED', closing_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'),
			expected_cash = ?, counted_cash = ?, difference = ?, notes = ?
		WHERE id = ?`, expected, countedCash, diff, notes, shiftID); err != nil {
		return nil, err
	}
	s.log.Write(ctx, audit.Event{Event: audit.EventShiftClose, UserID: &cashierID, EntityType: "shift", EntityID: &shiftID,
		Details: map[string]any{"expected_cash": expected, "counted_cash": countedCash, "difference": diff}})
	return s.Get(ctx, shiftID)
}
