package sales

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"pos/internal/domain"
)

func (s *Service) GetSale(ctx context.Context, id int64) (*Sale, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT s.id, s.receipt_number, s.register_id, s.shift_id, s.cashier_id, u.name, s.customer_id, COALESCE(c.name,''),
			s.subtotal, s.discount_total, s.tax_total, s.total, s.status, s.note, s.created_at
		FROM sales s
		JOIN users u ON u.id = s.cashier_id
		LEFT JOIN customers c ON c.id = s.customer_id
		WHERE s.id = ?`, id)
	sale, err := scanSale(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if err := s.hydrateSale(ctx, sale); err != nil {
		return nil, err
	}
	return sale, nil
}

func (s *Service) GetSaleByReceipt(ctx context.Context, receiptNumber string) (*Sale, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT s.id, s.receipt_number, s.register_id, s.shift_id, s.cashier_id, u.name, s.customer_id, COALESCE(c.name,''),
			s.subtotal, s.discount_total, s.tax_total, s.total, s.status, s.note, s.created_at
		FROM sales s
		JOIN users u ON u.id = s.cashier_id
		LEFT JOIN customers c ON c.id = s.customer_id
		WHERE s.receipt_number = ?`, receiptNumber)
	sale, err := scanSale(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if err := s.hydrateSale(ctx, sale); err != nil {
		return nil, err
	}
	return sale, nil
}

func scanSale(row interface{ Scan(...any) error }) (*Sale, error) {
	var sale Sale
	var customerID sql.NullInt64
	if err := row.Scan(&sale.ID, &sale.ReceiptNumber, &sale.RegisterID, &sale.ShiftID, &sale.CashierID, &sale.CashierName,
		&customerID, &sale.CustomerName, &sale.Subtotal, &sale.DiscountTotal, &sale.TaxTotal, &sale.Total, &sale.Status, &sale.Note, &sale.CreatedAt); err != nil {
		return nil, err
	}
	if customerID.Valid {
		v := customerID.Int64
		sale.CustomerID = &v
	}
	return &sale, nil
}

// hydrateSale fills in items, per-item modifiers, and payments. Every query
// here fully drains its Rows before the next one starts — the database
// connection pool is intentionally capped at one connection (see
// storage.Open), so a query issued while another Rows is still mid-iteration
// on the same *sql.DB would block forever waiting for a second connection
// that will never be released.
func (s *Service) hydrateSale(ctx context.Context, sale *Sale) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, product_id, name_snapshot, variant_snapshot, quantity, unit_price, discount_amount, tax_amount, line_total, notes FROM sale_items WHERE sale_id = ?`, sale.ID)
	if err != nil {
		return err
	}
	itemIndex := map[int64]int{}
	for rows.Next() {
		var it SaleItemResult
		var productID sql.NullInt64
		var variantSnapshot string
		if err := rows.Scan(&it.ID, &productID, &it.Name, &variantSnapshot, &it.Quantity, &it.UnitPrice, &it.DiscountAmount, &it.TaxAmount, &it.LineTotal, &it.Notes); err != nil {
			rows.Close()
			return err
		}
		if productID.Valid {
			v := productID.Int64
			it.ProductID = &v
		}
		it.Variant = variantSnapshot
		itemIndex[it.ID] = len(sale.Items)
		sale.Items = append(sale.Items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	if len(sale.Items) > 0 {
		modRows, err := s.db.QueryContext(ctx, `
			SELECT sim.sale_item_id, sim.name_snapshot FROM sale_item_modifiers sim
			JOIN sale_items si ON si.id = sim.sale_item_id
			WHERE si.sale_id = ?`, sale.ID)
		if err != nil {
			return err
		}
		for modRows.Next() {
			var itemID int64
			var name string
			if err := modRows.Scan(&itemID, &name); err != nil {
				modRows.Close()
				return err
			}
			if idx, ok := itemIndex[itemID]; ok {
				sale.Items[idx].Modifiers = append(sale.Items[idx].Modifiers, name)
			}
		}
		if err := modRows.Err(); err != nil {
			modRows.Close()
			return err
		}
		modRows.Close()
	}

	payRows, err := s.db.QueryContext(ctx, `SELECT id, method, amount, tendered, change_due, reference FROM payments WHERE sale_id = ?`, sale.ID)
	if err != nil {
		return err
	}
	defer payRows.Close()
	for payRows.Next() {
		var p PaymentResult
		if err := payRows.Scan(&p.ID, &p.Method, &p.Amount, &p.Tendered, &p.ChangeDue, &p.Reference); err != nil {
			return err
		}
		sale.Payments = append(sale.Payments, p)
		sale.ChangeDue += p.ChangeDue
	}
	return payRows.Err()
}

type ListFilter struct {
	ReceiptNumber string
	CustomerID    *int64
	CashierID     *int64
	PaymentMethod string
	Status        string
	DateFrom      string
	DateTo        string
	Limit         int
	Offset        int
}

func (s *Service) ListSales(ctx context.Context, f ListFilter) ([]Sale, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var sb strings.Builder
	sb.WriteString(`SELECT DISTINCT s.id, s.receipt_number, s.register_id, s.shift_id, s.cashier_id, u.name, s.customer_id, COALESCE(c.name,''),
		s.subtotal, s.discount_total, s.tax_total, s.total, s.status, s.note, s.created_at
		FROM sales s
		JOIN users u ON u.id = s.cashier_id
		LEFT JOIN customers c ON c.id = s.customer_id`)
	var args []any
	var where []string
	if f.PaymentMethod != "" {
		sb.WriteString(` JOIN payments pm ON pm.sale_id = s.id`)
		where = append(where, "pm.method = ?")
		args = append(args, f.PaymentMethod)
	}
	if f.ReceiptNumber != "" {
		where = append(where, "s.receipt_number LIKE ?")
		args = append(args, "%"+f.ReceiptNumber+"%")
	}
	if f.CustomerID != nil {
		where = append(where, "s.customer_id = ?")
		args = append(args, *f.CustomerID)
	}
	if f.CashierID != nil {
		where = append(where, "s.cashier_id = ?")
		args = append(args, *f.CashierID)
	}
	if f.Status != "" {
		where = append(where, "s.status = ?")
		args = append(args, f.Status)
	}
	if f.DateFrom != "" {
		where = append(where, "s.created_at >= ?")
		args = append(args, f.DateFrom)
	}
	if f.DateTo != "" {
		where = append(where, "s.created_at <= ?")
		args = append(args, f.DateTo)
	}
	if len(where) > 0 {
		sb.WriteString(" WHERE " + strings.Join(where, " AND "))
	}
	sb.WriteString(" ORDER BY s.id DESC LIMIT ? OFFSET ?")
	args = append(args, limit, f.Offset)

	rows, err := s.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Sale{}
	for rows.Next() {
		sale, err := scanSale(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sale)
	}
	return out, rows.Err()
}
