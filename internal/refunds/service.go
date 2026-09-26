// Package refunds implements full/partial/item refunds. A refund always
// references its original sale and never mutates that sale's historical
// rows (brief §39) — it only adds refund/refund_items/refund_payments rows,
// restores stock via a RETURN movement, and reverses proportional loyalty
// points.
package refunds

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"pos/internal/audit"
	"pos/internal/domain"
	"pos/internal/inventory"
	"pos/internal/loyalty"
	"pos/internal/security"
	"pos/internal/storage"
)

type ItemInput struct {
	SaleItemID int64 `json:"sale_item_id"`
	Quantity   int64 `json:"quantity"`
}

type PaymentInput struct {
	Method    string `json:"method"`
	Amount    int64  `json:"amount"`
	Reference string `json:"reference,omitempty"`
}

type Input struct {
	SaleID       int64          `json:"sale_id"`
	Items        []ItemInput    `json:"items"`
	Reason       string         `json:"reason"`
	ApproverID   *int64         `json:"approver_id,omitempty"`
	Payments     []PaymentInput `json:"payments"`
}

type Refund struct {
	ID        int64  `json:"id"`
	SaleID    int64  `json:"sale_id"`
	CashierID int64  `json:"cashier_id"`
	Reason    string `json:"reason"`
	Total     int64  `json:"total"`
	CreatedAt string `json:"created_at"`
}

type Service struct {
	db  *storage.DB
	log *audit.Logger
}

func NewService(db *storage.DB, log *audit.Logger) *Service {
	return &Service{db: db, log: log}
}

func (s *Service) Create(ctx context.Context, registerID, cashierID int64, perms []string, in Input) (*Refund, error) {
	if !security.HasPermission(perms, security.PermRefund) {
		return nil, domain.ErrForbidden
	}
	if len(in.Items) == 0 {
		return nil, domain.NewError("INVALID_INPUT", "A refund must include at least one item", 400)
	}

	var saleStatus string
	var saleTotal int64
	var customerID sql.NullInt64
	row := s.db.QueryRowContext(ctx, `SELECT status, total, customer_id FROM sales WHERE id = ?`, in.SaleID)
	if err := row.Scan(&saleStatus, &saleTotal, &customerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError("SALE_NOT_FOUND", "Original sale not found", 404)
		}
		return nil, err
	}
	if saleStatus != "COMPLETED" && saleStatus != "PARTIALLY_REFUNDED" {
		return nil, domain.NewError("SALE_NOT_REFUNDABLE", fmt.Sprintf("Sale is %s and cannot be refunded", saleStatus), 400)
	}

	var shiftID int64
	shiftRow := s.db.QueryRowContext(ctx, `SELECT id FROM shifts WHERE register_id = ? AND status = 'OPEN'`, registerID)
	if err := shiftRow.Scan(&shiftID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError("NO_OPEN_SHIFT", "Open a shift before processing refunds", 409)
		}
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var refundTotal int64
	type resolvedItem struct {
		saleItemID int64
		productID  sql.NullInt64
		trackStock bool
		quantity   int64
		amount     int64
	}
	var resolved []resolvedItem

	for _, item := range in.Items {
		var productID sql.NullInt64
		var origQty, unitPrice, taxAmount, discountAmount, lineTotal int64
		itemRow := tx.QueryRowContext(ctx, `SELECT product_id, quantity, unit_price, tax_amount, discount_amount, line_total FROM sale_items WHERE id = ? AND sale_id = ?`,
			item.SaleItemID, in.SaleID)
		if err := itemRow.Scan(&productID, &origQty, &unitPrice, &taxAmount, &discountAmount, &lineTotal); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, domain.NewError("SALE_ITEM_NOT_FOUND", "Sale item not found on this sale", 404)
			}
			return nil, err
		}
		var alreadyRefunded int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(quantity),0) FROM refund_items WHERE sale_item_id = ?`, item.SaleItemID).Scan(&alreadyRefunded); err != nil {
			return nil, err
		}
		remaining := origQty - alreadyRefunded
		if item.Quantity <= 0 || item.Quantity > remaining {
			return nil, domain.NewError("INVALID_INPUT", fmt.Sprintf("Cannot refund %d units; only %d remain refundable", item.Quantity, remaining), 400)
		}

		amount := lineTotal * item.Quantity / origQty
		refundTotal += amount

		trackStock := false
		if productID.Valid {
			if err := tx.QueryRowContext(ctx, `SELECT track_stock FROM products WHERE id = ?`, productID.Int64).Scan(&trackStock); err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
		}
		resolved = append(resolved, resolvedItem{item.SaleItemID, productID, trackStock, item.Quantity, amount})
	}

	var paidTotal int64
	for _, p := range in.Payments {
		paidTotal += p.Amount
	}
	if paidTotal != refundTotal {
		return nil, domain.NewError("INVALID_INPUT", "Refund payment total must equal the refunded amount", 400)
	}

	res, err := tx.ExecContext(ctx, `INSERT INTO refunds (sale_id, shift_id, cashier_id, approver_id, reason, total) VALUES (?,?,?,?,?,?)`,
		in.SaleID, shiftID, cashierID, in.ApproverID, in.Reason, refundTotal)
	if err != nil {
		return nil, err
	}
	refundID, _ := res.LastInsertId()

	for _, r := range resolved {
		if _, err := tx.ExecContext(ctx, `INSERT INTO refund_items (refund_id, sale_item_id, quantity, amount) VALUES (?,?,?,?)`,
			refundID, r.saleItemID, r.quantity, r.amount); err != nil {
			return nil, err
		}
		if r.productID.Valid && r.trackStock {
			pid := r.productID.Int64
			rid := refundID
			if err := inventory.ApplyMovement(ctx, tx, pid, inventory.TypeReturn, r.quantity, "refund", &rid, "", &cashierID); err != nil {
				return nil, err
			}
		}
	}

	for _, p := range in.Payments {
		if _, err := tx.ExecContext(ctx, `INSERT INTO refund_payments (refund_id, method, amount, reference) VALUES (?,?,?,?)`,
			refundID, p.Method, p.Amount, p.Reference); err != nil {
			return nil, err
		}
		if p.Method == "CASH" {
			if _, err := tx.ExecContext(ctx, `INSERT INTO cash_movements (shift_id, type, amount, reason, cashier_id, reference_type, reference_id) VALUES (?, 'REFUND_CASH', ?, 'Refund', ?, 'refund', ?)`,
				shiftID, -p.Amount, cashierID, refundID); err != nil {
				return nil, err
			}
		}
	}

	var totalRefundedSoFar int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(total),0) FROM refunds WHERE sale_id = ?`, in.SaleID).Scan(&totalRefundedSoFar); err != nil {
		return nil, err
	}
	newStatus := "PARTIALLY_REFUNDED"
	if totalRefundedSoFar >= saleTotal {
		newStatus = "REFUNDED"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sales SET status = ? WHERE id = ?`, newStatus, in.SaleID); err != nil {
		return nil, err
	}

	if customerID.Valid && saleTotal > 0 {
		var earned int64
		_ = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(points),0) FROM customer_loyalty_transactions WHERE sale_id = ? AND type = 'EARN'`, in.SaleID).Scan(&earned)
		if earned > 0 {
			reversal := earned * refundTotal / saleTotal
			if err := loyalty.ReverseForSale(ctx, tx, customerID.Int64, in.SaleID, reversal); err != nil {
				return nil, err
			}
		}
	}

	auditExec := func(query string, args ...any) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}
	if err := audit.WriteTxRaw(ctx, auditExec, audit.Event{
		Event: audit.EventRefund, UserID: &cashierID, ApproverID: in.ApproverID, EntityType: "sale", EntityID: &in.SaleID,
		Details: map[string]any{"refund_id": refundID, "total": refundTotal, "reason": in.Reason},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Refund{ID: refundID, SaleID: in.SaleID, CashierID: cashierID, Reason: in.Reason, Total: refundTotal}, nil
}
