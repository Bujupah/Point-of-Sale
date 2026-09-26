package sales

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"pos/internal/audit"
	"pos/internal/domain"
	"pos/internal/security"
)

type HoldInput struct {
	CustomerID *int64      `json:"customer_id,omitempty"`
	Note       string      `json:"note,omitempty"`
	TableName  string      `json:"table_name,omitempty"`
	TicketName string      `json:"ticket_name,omitempty"`
	Items      []ItemInput `json:"items"`
}

type HeldSale struct {
	ID         int64            `json:"id"`
	RegisterID int64            `json:"register_id"`
	CashierID  int64            `json:"cashier_id"`
	CustomerID *int64           `json:"customer_id,omitempty"`
	Note       string           `json:"note"`
	TableName  string           `json:"table_name"`
	TicketName string           `json:"ticket_name"`
	Subtotal   domain.Money     `json:"subtotal"`
	Total      domain.Money     `json:"total"`
	ItemCount  int              `json:"item_count"`
	CreatedAt  string           `json:"created_at"`
	Items      []SaleItemResult `json:"items,omitempty"`
}

// Hold parks the current cart as a durable row in SQLite (never only in
// frontend memory, per brief §35/§76) so a crash or restart never loses it.
func (s *Service) Hold(ctx context.Context, registerID, cashierID int64, perms []string, in HoldInput) (*HeldSale, error) {
	if !security.HasPermission(perms, security.PermSaleHold) {
		return nil, domain.ErrForbidden
	}
	if len(in.Items) == 0 {
		return nil, domain.NewError("INVALID_INPUT", "Cannot hold an empty cart", 400)
	}
	canOverride := security.HasPermission(perms, security.PermPriceOverride)
	canOpenItem := security.HasPermission(perms, security.PermOpenItem)

	var lines []*resolvedLine
	for _, item := range in.Items {
		l, err := resolveItem(ctx, s.catalog, item, canOverride, canOpenItem)
		if err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	t, err := finalize(lines, 0, "", 0)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `INSERT INTO held_sales (register_id, cashier_id, customer_id, note, table_name, ticket_name, subtotal, discount_total, tax_total, total)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		registerID, cashierID, in.CustomerID, in.Note, in.TableName, in.TicketName, int64(t.Subtotal), int64(t.DiscountTotal), int64(t.TaxTotal), int64(t.Total))
	if err != nil {
		return nil, err
	}
	heldID, _ := res.LastInsertId()

	for _, l := range lines {
		variantSnapshot := ""
		if len(l.VariantNames) > 0 {
			if b, err := json.Marshal(append(append([]string{}, l.VariantNames...), l.ModifierNames...)); err == nil {
				variantSnapshot = string(b)
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO held_sale_items (held_sale_id, product_id, name_snapshot, variant_snapshot, quantity, unit_price, notes) VALUES (?,?,?,?,?,?,?)`,
			heldID, l.ProductID, l.Name, variantSnapshot, l.Quantity, int64(l.UnitPrice), l.Notes); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.log.Write(ctx, audit.Event{Event: audit.EventSaleHeld, UserID: &cashierID, EntityType: "held_sale", EntityID: &heldID})
	return s.GetHeld(ctx, heldID)
}

func (s *Service) ListHeld(ctx context.Context, registerID int64) ([]HeldSale, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, register_id, cashier_id, customer_id, note, table_name, ticket_name, subtotal, total, created_at,
		(SELECT COUNT(*) FROM held_sale_items WHERE held_sale_id = held_sales.id)
		FROM held_sales WHERE register_id = ? ORDER BY id DESC`, registerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HeldSale{}
	for rows.Next() {
		var h HeldSale
		var customerID sql.NullInt64
		if err := rows.Scan(&h.ID, &h.RegisterID, &h.CashierID, &customerID, &h.Note, &h.TableName, &h.TicketName, &h.Subtotal, &h.Total, &h.CreatedAt, &h.ItemCount); err != nil {
			return nil, err
		}
		if customerID.Valid {
			v := customerID.Int64
			h.CustomerID = &v
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Service) GetHeld(ctx context.Context, id int64) (*HeldSale, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, register_id, cashier_id, customer_id, note, table_name, ticket_name, subtotal, total, created_at FROM held_sales WHERE id = ?`, id)
	var h HeldSale
	var customerID sql.NullInt64
	if err := row.Scan(&h.ID, &h.RegisterID, &h.CashierID, &customerID, &h.Note, &h.TableName, &h.TicketName, &h.Subtotal, &h.Total, &h.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if customerID.Valid {
		v := customerID.Int64
		h.CustomerID = &v
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, product_id, name_snapshot, variant_snapshot, quantity, unit_price, notes FROM held_sale_items WHERE held_sale_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it SaleItemResult
		var productID sql.NullInt64
		if err := rows.Scan(&it.ID, &productID, &it.Name, &it.Variant, &it.Quantity, &it.UnitPrice, &it.Notes); err != nil {
			return nil, err
		}
		if productID.Valid {
			v := productID.Int64
			it.ProductID = &v
		}
		h.Items = append(h.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	h.ItemCount = len(h.Items)
	return &h, nil
}

// DeleteHeld removes a parked sale (resumed into the cart client-side, or
// explicitly discarded).
func (s *Service) DeleteHeld(ctx context.Context, id, userID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM held_sale_items WHERE held_sale_id = ?`, id); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM held_sales WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.log.Write(ctx, audit.Event{Event: audit.EventSaleResumed, UserID: &userID, EntityType: "held_sale", EntityID: &id})
	return nil
}
