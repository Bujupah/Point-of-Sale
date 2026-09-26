// Package inventory records why stock changed, never mutating a balance
// without a corresponding stock_movements row.
package inventory

import (
	"context"
	"database/sql"

	"pos/internal/audit"
	"pos/internal/domain"
	"pos/internal/storage"
)

const (
	TypeInitial     = "INITIAL"
	TypePurchase    = "PURCHASE"
	TypeSale        = "SALE"
	TypeReturn      = "RETURN"
	TypeAdjustment  = "ADJUSTMENT"
	TypeDamage      = "DAMAGE"
	TypeTransferIn  = "TRANSFER_IN"
	TypeTransferOut = "TRANSFER_OUT"
)

type Movement struct {
	ID            int64  `json:"id"`
	ProductID     int64  `json:"product_id"`
	ProductName   string `json:"product_name"`
	Type          string `json:"type"`
	Quantity      int64  `json:"quantity"`
	ReferenceType string `json:"reference_type"`
	ReferenceID   *int64 `json:"reference_id,omitempty"`
	Note          string `json:"note"`
	CreatedBy     *int64 `json:"created_by,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type Service struct {
	db  *storage.DB
	log *audit.Logger
}

func NewService(db *storage.DB, log *audit.Logger) *Service {
	return &Service{db: db, log: log}
}

// ApplyMovement records a stock movement and updates the cached balance
// inside an existing transaction. Sale commit, refunds, and held-sale resume
// all call this so every stock change — whatever triggered it — goes
// through the same insert-then-upsert pair.
func ApplyMovement(ctx context.Context, tx *sql.Tx, productID int64, movementType string, quantity int64, referenceType string, referenceID *int64, note string, userID *int64) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO stock_movements (product_id, type, quantity, reference_type, reference_id, note, created_by) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		productID, movementType, quantity, referenceType, referenceID, note, userID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO inventory_balances (product_id, quantity, updated_at) VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		ON CONFLICT(product_id) DO UPDATE SET quantity = quantity + excluded.quantity, updated_at = excluded.updated_at`,
		productID, quantity)
	return err
}

// AdjustStock performs a standalone, audited stock adjustment (manual count
// correction, damage, etc.) as its own atomic transaction.
func (s *Service) AdjustStock(ctx context.Context, productID int64, movementType string, quantity int64, note string, userID int64) error {
	if quantity == 0 {
		return domain.NewError("INVALID_INPUT", "Adjustment quantity cannot be zero", 400)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	uid := userID
	if err := ApplyMovement(ctx, tx, productID, movementType, quantity, "manual", nil, note, &uid); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.log.Write(ctx, audit.Event{
		Event: audit.EventInventoryAdjust, UserID: &uid, EntityType: "product", EntityID: &productID,
		Details: map[string]any{"type": movementType, "quantity": quantity, "note": note},
	})
	return nil
}

func (s *Service) ListMovements(ctx context.Context, productID *int64, limit int) ([]Movement, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT sm.id, sm.product_id, p.name, sm.type, sm.quantity, sm.reference_type, sm.reference_id, sm.note, sm.created_by, sm.created_at
		FROM stock_movements sm JOIN products p ON p.id = sm.product_id WHERE 1=1`
	var args []any
	if productID != nil {
		query += ` AND sm.product_id = ?`
		args = append(args, *productID)
	}
	query += ` ORDER BY sm.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Movement{}
	for rows.Next() {
		var m Movement
		if err := rows.Scan(&m.ID, &m.ProductID, &m.ProductName, &m.Type, &m.Quantity, &m.ReferenceType, &m.ReferenceID, &m.Note, &m.CreatedBy, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type LowStockItem struct {
	ProductID    int64  `json:"product_id"`
	Name         string `json:"name"`
	SKU          string `json:"sku"`
	Stock        int64  `json:"stock"`
	ReorderLevel int64  `json:"reorder_level"`
}

func (s *Service) LowStock(ctx context.Context) ([]LowStockItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.name, p.sku, COALESCE(ib.quantity,0), p.reorder_level
		FROM products p LEFT JOIN inventory_balances ib ON ib.product_id = p.id
		WHERE p.track_stock = 1 AND p.active = 1 AND COALESCE(ib.quantity,0) <= p.reorder_level
		ORDER BY COALESCE(ib.quantity,0) ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LowStockItem{}
	for rows.Next() {
		var i LowStockItem
		if err := rows.Scan(&i.ProductID, &i.Name, &i.SKU, &i.Stock, &i.ReorderLevel); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
