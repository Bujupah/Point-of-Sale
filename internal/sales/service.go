package sales

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"pos/internal/audit"
	"pos/internal/catalog"
	"pos/internal/customers"
	"pos/internal/domain"
	"pos/internal/giftcards"
	"pos/internal/inventory"
	"pos/internal/loyalty"
	"pos/internal/security"
	"pos/internal/settings"
	"pos/internal/storage"
)

var validPaymentMethods = map[string]bool{
	"CASH": true, "CARD": true, "BANK_TRANSFER": true, "GIFT_CARD": true,
	"OTHER": true, "MOBILE": true, "VOUCHER": true, "CHEQUE": true, "CUSTOM": true,
}

type Service struct {
	db       *storage.DB
	catalog  *catalog.Service
	settings *settings.Service
	log      *audit.Logger
}

func NewService(db *storage.DB, cat *catalog.Service, set *settings.Service, log *audit.Logger) *Service {
	return &Service{db: db, catalog: cat, settings: set, log: log}
}

// Checkout resolves, prices, and atomically commits a sale: items, payments,
// stock movements, cash movements, loyalty, and audit log all in one
// transaction. Nothing about the sale is visible to the caller until COMMIT
// succeeds (brief §30).
func (s *Service) Checkout(ctx context.Context, registerID, cashierID int64, perms []string, in CheckoutInput) (*Sale, error) {
	if !security.HasPermission(perms, security.PermSaleCreate) {
		return nil, domain.ErrForbidden
	}
	if len(in.Items) == 0 {
		return nil, domain.NewError("INVALID_INPUT", "A sale must have at least one item", 400)
	}
	if len(in.Payments) == 0 {
		return nil, domain.NewError("INVALID_INPUT", "A sale must have at least one payment", 400)
	}

	var shiftID int64
	row := s.db.QueryRowContext(ctx, `SELECT id FROM shifts WHERE register_id = ? AND status = 'OPEN'`, registerID)
	if err := row.Scan(&shiftID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError("NO_OPEN_SHIFT", "Open a shift before selling", 409)
		}
		return nil, err
	}

	canOverride := security.HasPermission(perms, security.PermPriceOverride)
	canOpenItem := security.HasPermission(perms, security.PermOpenItem)

	var lines []*resolvedLine
	hasLineDiscount := false
	for _, item := range in.Items {
		l, err := resolveItem(ctx, s.catalog, item, canOverride, canOpenItem)
		if err != nil {
			return nil, err
		}
		if l.LineDiscount > 0 {
			hasLineDiscount = true
		}
		if l.TrackStock && l.AvailableStock < l.Quantity {
			return nil, domain.NewError("INSUFFICIENT_STOCK", fmt.Sprintf("Not enough stock for %s (available: %d)", l.Name, l.AvailableStock), 409)
		}
		lines = append(lines, l)
	}

	hasOrderDiscount := in.OrderDiscountType != ""
	if (hasLineDiscount || hasOrderDiscount) && !security.HasPermission(perms, security.PermDiscountApply) {
		return nil, domain.NewError("FORBIDDEN", "You do not have permission to apply discounts", 403)
	}
	maxBps := s.settings.GetInt64(ctx, settings.KeyDiscountMaxPercent, 2000)
	if hasOrderDiscount && in.OrderDiscountType == "PERCENT" && in.OrderDiscountPercentBps > maxBps {
		if in.ManagerApproverID == nil || !security.HasPermission(perms, security.PermDiscountOverride) {
			return nil, domain.NewError("DISCOUNT_LIMIT_EXCEEDED", "This discount exceeds the cashier limit and requires manager approval", 403)
		}
	}

	t, err := finalize(lines, in.OrderDiscountAmount, in.OrderDiscountType, in.OrderDiscountPercentBps)
	if err != nil {
		return nil, err
	}

	var sumApplied domain.Money
	for _, p := range in.Payments {
		if !validPaymentMethods[p.Method] {
			return nil, domain.NewError("INVALID_INPUT", fmt.Sprintf("Unknown payment method %q", p.Method), 400)
		}
		if p.Amount <= 0 {
			return nil, domain.NewError("INVALID_INPUT", "Payment amount must be positive", 400)
		}
		sumApplied += p.Amount
	}
	if sumApplied < t.Total {
		return nil, domain.NewError("INSUFFICIENT_PAYMENT", "Payments do not cover the total due", 400)
	}
	if sumApplied > t.Total {
		return nil, domain.NewError("OVERPAYMENT", "Payments exceed the total due; express change via cash tendered instead", 400)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `INSERT INTO sales
		(receipt_number, register_id, shift_id, cashier_id, customer_id, subtotal, discount_total, discount_type, discount_reason, discount_approved_by, tax_total, total, note)
		VALUES ('PENDING', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		registerID, shiftID, cashierID, in.CustomerID, int64(t.Subtotal), int64(t.DiscountTotal),
		in.OrderDiscountType, in.OrderDiscountReason, in.ManagerApproverID, int64(t.TaxTotal), int64(t.Total), in.Note)
	if err != nil {
		return nil, err
	}
	saleID, _ := res.LastInsertId()
	receiptNumber := fmt.Sprintf("%s-%06d", time.Now().Format("20060102"), saleID)
	if _, err := tx.ExecContext(ctx, `UPDATE sales SET receipt_number = ? WHERE id = ?`, receiptNumber, saleID); err != nil {
		return nil, err
	}

	itemResults := make([]SaleItemResult, 0, len(lines))
	for _, l := range lines {
		variantSnapshot := ""
		if len(l.VariantNames) > 0 {
			if b, err := json.Marshal(l.VariantNames); err == nil {
				variantSnapshot = string(b)
			}
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO sale_items
			(sale_id, product_id, sku_snapshot, barcode_snapshot, name_snapshot, variant_snapshot, quantity, unit_price, cost_snapshot, discount_amount, discount_type, tax_amount, line_total, notes)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			saleID, l.ProductID, l.SKU, l.Barcode, l.Name, variantSnapshot, l.Quantity, int64(l.UnitPrice), int64(l.CostSnapshot),
			int64(l.LineDiscount+l.OrderCut), l.DiscountType, int64(l.TaxAmount), int64(l.LineTotal), l.Notes)
		if err != nil {
			return nil, err
		}
		itemID, _ := res.LastInsertId()

		for _, mn := range l.ModifierNames {
			if _, err := tx.ExecContext(ctx, `INSERT INTO sale_item_modifiers (sale_item_id, name_snapshot, price_adjustment) VALUES (?, ?, 0)`, itemID, mn); err != nil {
				return nil, err
			}
		}

		if l.ProductID != nil && l.TrackStock {
			pid := *l.ProductID
			if err := inventory.ApplyMovement(ctx, tx, pid, inventory.TypeSale, -l.Quantity, "sale", &saleID, "", &cashierID); err != nil {
				return nil, err
			}
		}

		itemResults = append(itemResults, SaleItemResult{
			ID: itemID, ProductID: l.ProductID, Name: l.Name, Modifiers: l.ModifierNames,
			Quantity: l.Quantity, UnitPrice: l.UnitPrice, DiscountAmount: l.LineDiscount + l.OrderCut,
			TaxAmount: l.TaxAmount, LineTotal: l.LineTotal, Notes: l.Notes,
		})
	}

	var changeDue domain.Money
	paymentResults := make([]PaymentResult, 0, len(in.Payments))
	for _, p := range in.Payments {
		tendered := p.Amount
		if p.Method == "CASH" {
			if p.Tendered > 0 {
				tendered = p.Tendered
			}
			if tendered < p.Amount {
				return nil, domain.NewError("INVALID_INPUT", "Cash tendered cannot be less than the amount applied", 400)
			}
		} else {
			tendered = p.Amount
		}
		change := tendered - p.Amount
		changeDue += change

		res, err := tx.ExecContext(ctx, `INSERT INTO payments (sale_id, method, amount, tendered, change_due, reference, status) VALUES (?,?,?,?,?,?, 'APPROVED')`,
			saleID, p.Method, int64(p.Amount), int64(tendered), int64(change), p.Reference)
		if err != nil {
			return nil, err
		}
		paymentID, _ := res.LastInsertId()

		switch p.Method {
		case "CASH":
			if _, err := tx.ExecContext(ctx, `INSERT INTO cash_movements (shift_id, type, amount, reason, cashier_id, reference_type, reference_id) VALUES (?, 'SALE_CASH', ?, 'Sale payment', ?, 'sale', ?)`,
				shiftID, int64(p.Amount), cashierID, saleID); err != nil {
				return nil, err
			}
		case "GIFT_CARD":
			if err := giftcards.RedeemTx(ctx, tx, p.Reference, p.Amount, saleID); err != nil {
				return nil, err
			}
		}

		paymentResults = append(paymentResults, PaymentResult{ID: paymentID, Method: p.Method, Amount: p.Amount, Tendered: tendered, ChangeDue: change, Reference: p.Reference})
	}

	var loyaltyEarned int64
	if in.CustomerID != nil {
		if err := customers.RecordVisit(ctx, tx, *in.CustomerID, int64(t.Total)); err != nil {
			return nil, err
		}
		rate := s.settings.GetInt64(ctx, settings.KeyLoyaltyRatePer100, 0)
		loyaltyEarned, err = loyalty.EarnForSale(ctx, tx, *in.CustomerID, saleID, int64(t.Total), rate)
		if err != nil {
			return nil, err
		}
	}

	auditExec := func(query string, args ...any) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}
	if err := audit.WriteTxRaw(ctx, auditExec, audit.Event{
		Event: audit.EventSaleCompleted, UserID: &cashierID, EntityType: "sale", EntityID: &saleID,
		Details: map[string]any{"receipt_number": receiptNumber, "total": int64(t.Total)},
	}); err != nil {
		return nil, err
	}
	if hasOrderDiscount {
		event := audit.EventDiscount
		if in.ManagerApproverID != nil {
			event = audit.EventDiscountOverride
		}
		if err := audit.WriteTxRaw(ctx, auditExec, audit.Event{
			Event: event, UserID: &cashierID, ApproverID: in.ManagerApproverID, EntityType: "sale", EntityID: &saleID,
			Details: map[string]any{"reason": in.OrderDiscountReason, "discount_total": int64(t.DiscountTotal)},
		}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Sale{
		ID: saleID, ReceiptNumber: receiptNumber, RegisterID: registerID, ShiftID: shiftID, CashierID: cashierID,
		CustomerID: in.CustomerID, Subtotal: t.Subtotal, DiscountTotal: t.DiscountTotal, TaxTotal: t.TaxTotal, Total: t.Total,
		Status: "COMPLETED", Note: in.Note, Items: itemResults, Payments: paymentResults, LoyaltyEarned: loyaltyEarned, ChangeDue: changeDue,
	}, nil
}
