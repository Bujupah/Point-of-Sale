// Package audit records the append-only trail required by every sensitive
// operation in the POS (logins, sales, refunds, discounts, cash movements,
// settings changes, backups). It never fails the caller's operation: a
// logging failure is reported to the application log, not propagated.
package audit

import (
	"context"
	"encoding/json"
	"log"

	"pos/internal/storage"
)

const (
	EventLogin              = "LOGIN"
	EventLogout             = "LOGOUT"
	EventLock               = "LOCK"
	EventUnlock             = "UNLOCK"
	EventCashierSwitch      = "CASHIER_SWITCH"
	EventSaleCompleted      = "SALE_COMPLETED"
	EventSaleVoided         = "SALE_VOIDED"
	EventSaleHeld           = "SALE_HELD"
	EventSaleResumed        = "SALE_RESUMED"
	EventRefund             = "REFUND"
	EventDiscount           = "DISCOUNT"
	EventDiscountOverride   = "DISCOUNT_OVERRIDE"
	EventPriceOverride      = "PRICE_OVERRIDE"
	EventShiftOpen          = "SHIFT_OPEN"
	EventShiftClose         = "SHIFT_CLOSE"
	EventCashIn             = "CASH_IN"
	EventCashOut            = "CASH_OUT"
	EventDrawerOpen         = "DRAWER_OPEN"
	EventInventoryAdjust    = "INVENTORY_ADJUSTMENT"
	EventSettingsChanged    = "SETTINGS_CHANGED"
	EventBackup             = "BACKUP"
	EventRestore            = "RESTORE"
)

type Event struct {
	Event      string
	UserID     *int64
	ApproverID *int64
	EntityType string
	EntityID   *int64
	Details    map[string]any
}

type Logger struct {
	db *storage.DB
}

func NewLogger(db *storage.DB) *Logger {
	return &Logger{db: db}
}

// Write persists an audit event. Failures are logged, never returned, so a
// broken audit write can never block or roll back the business operation
// that triggered it (the sale/refund/etc. transaction has already committed
// by the time this is typically called; for events inside a transaction,
// use WriteTx instead).
func (l *Logger) Write(ctx context.Context, e Event) {
	details := "{}"
	if e.Details != nil {
		if b, err := json.Marshal(e.Details); err == nil {
			details = string(b)
		}
	}
	if _, err := l.db.ExecContext(ctx, `
		INSERT INTO audit_logs (event, user_id, approver_id, entity_type, entity_id, details_json)
		VALUES (?, ?, ?, ?, ?, ?)`,
		e.Event, e.UserID, e.ApproverID, e.EntityType, e.EntityID, details); err != nil {
		log.Printf("audit: failed to write event %s: %v", e.Event, err)
	}
}

// WriteTxRaw inserts the audit row using the given exec function, letting
// callers already inside a transaction (e.g. a sale commit) have the audit
// row committed atomically with the rest of the operation, without an
// interface adapter dance around *sql.Tx.
// caller pass tx.ExecContext directly without an interface adapter dance.
func WriteTxRaw(ctx context.Context, exec func(query string, args ...any) error, e Event) error {
	details := "{}"
	if e.Details != nil {
		if b, err := json.Marshal(e.Details); err == nil {
			details = string(b)
		}
	}
	return exec(`
		INSERT INTO audit_logs (event, user_id, approver_id, entity_type, entity_id, details_json)
		VALUES (?, ?, ?, ?, ?, ?)`,
		e.Event, e.UserID, e.ApproverID, e.EntityType, e.EntityID, details)
}
