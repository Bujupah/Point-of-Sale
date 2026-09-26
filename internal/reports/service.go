// Package reports powers the Reports dashboard: summary KPIs, top products,
// and payment mix over an arbitrary date range (today, shift, yesterday,
// 7 days, custom).
package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"

	"pos/internal/storage"
)

type Summary struct {
	Revenue       int64 `json:"revenue"`
	OrderCount    int64 `json:"order_count"`
	AverageTicket int64 `json:"average_ticket"`
	DiscountTotal int64 `json:"discount_total"`
	TaxTotal      int64 `json:"tax_total"`
	RefundsTotal  int64 `json:"refunds_total"`
	CashAmount    int64 `json:"cash_amount"`
	CardAmount    int64 `json:"card_amount"`
	OtherAmount   int64 `json:"other_amount"`
	CashPercent   int64 `json:"cash_percent_bps"`
	CardPercent   int64 `json:"card_percent_bps"`
}

type TopProduct struct {
	ProductID   *int64 `json:"product_id,omitempty"`
	Name        string `json:"name"`
	QuantitySold int64 `json:"quantity_sold"`
	SalesAmount  int64 `json:"sales_amount"`
	PercentOfRevenueBps int64 `json:"percent_of_revenue_bps"`
}

type PaymentMix struct {
	Method string `json:"method"`
	Amount int64  `json:"amount"`
	Count  int64  `json:"count"`
}

type CashierMetric struct {
	CashierID   int64  `json:"cashier_id"`
	CashierName string `json:"cashier_name"`
	OrderCount  int64  `json:"order_count"`
	Revenue     int64  `json:"revenue"`
}

type Service struct {
	db *storage.DB
}

func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

// DateRange filters by sales.created_at (ISO-8601 strings, lexically
// comparable). Empty bounds mean unbounded.
type DateRange struct {
	From string
	To   string
}

// apply appends WHERE fragments qualified against alias "s" — every caller
// aliases the sales table as s, including single-table queries, so this can
// be shared unchanged with queries that JOIN payments (which has its own
// created_at/status columns and would otherwise make the bare column names
// ambiguous).
func (r DateRange) apply(where *[]string, args *[]any) {
	if r.From != "" {
		*where = append(*where, "s.created_at >= ?")
		*args = append(*args, r.From)
	}
	if r.To != "" {
		*where = append(*where, "s.created_at <= ?")
		*args = append(*args, r.To)
	}
}

func (s *Service) Summary(ctx context.Context, r DateRange) (*Summary, error) {
	var where []string
	var args []any
	where = append(where, "s.status != 'VOIDED'")
	r.apply(&where, &args)
	whereClause := "WHERE " + join(where, " AND ")

	sum := &Summary{}
	row := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(total),0), COUNT(*), COALESCE(SUM(discount_total),0), COALESCE(SUM(tax_total),0) FROM sales s `+whereClause, args...)
	if err := row.Scan(&sum.Revenue, &sum.OrderCount, &sum.DiscountTotal, &sum.TaxTotal); err != nil {
		return nil, err
	}
	if sum.OrderCount > 0 {
		sum.AverageTicket = sum.Revenue / sum.OrderCount
	}

	var refundWhere []string
	var refundArgs []any
	r.apply(&refundWhere, &refundArgs)
	refundClause := ""
	if len(refundWhere) > 0 {
		refundClause = "WHERE " + join(refundWhere, " AND ")
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(total),0) FROM refunds s `+refundClause, refundArgs...).Scan(&sum.RefundsTotal); err != nil {
		return nil, err
	}

	payRows, err := s.db.QueryContext(ctx, `
		SELECT p.method, COALESCE(SUM(p.amount),0) FROM payments p
		JOIN sales s ON s.id = p.sale_id `+whereClause+` GROUP BY p.method`, args...)
	if err != nil {
		return nil, err
	}
	defer payRows.Close()
	for payRows.Next() {
		var method string
		var amt int64
		if err := payRows.Scan(&method, &amt); err != nil {
			return nil, err
		}
		switch method {
		case "CASH":
			sum.CashAmount = amt
		case "CARD":
			sum.CardAmount = amt
		default:
			sum.OtherAmount += amt
		}
	}
	if sum.Revenue > 0 {
		sum.CashPercent = sum.CashAmount * 10000 / sum.Revenue
		sum.CardPercent = sum.CardAmount * 10000 / sum.Revenue
	}
	return sum, nil
}

func (s *Service) TopProducts(ctx context.Context, r DateRange, limit int) ([]TopProduct, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var where []string
	var args []any
	where = append(where, "s.status != 'VOIDED'")
	r.apply(&where, &args)
	whereClause := "WHERE " + join(where, " AND ")

	var totalRevenue int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(total),0) FROM sales s `+whereClause, args...).Scan(&totalRevenue); err != nil {
		return nil, err
	}

	query := `
		SELECT si.product_id, si.name_snapshot, SUM(si.quantity), SUM(si.line_total)
		FROM sale_items si JOIN sales s ON s.id = si.sale_id
		` + whereClause + `
		GROUP BY COALESCE(si.product_id, -si.id), si.name_snapshot
		ORDER BY SUM(si.line_total) DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TopProduct{}
	for rows.Next() {
		var p TopProduct
		var productID *int64
		if err := rows.Scan(&productID, &p.Name, &p.QuantitySold, &p.SalesAmount); err != nil {
			return nil, err
		}
		p.ProductID = productID
		if totalRevenue > 0 {
			p.PercentOfRevenueBps = p.SalesAmount * 10000 / totalRevenue
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) PaymentMix(ctx context.Context, r DateRange) ([]PaymentMix, error) {
	var where []string
	var args []any
	where = append(where, "s.status != 'VOIDED'")
	r.apply(&where, &args)
	whereClause := "WHERE " + join(where, " AND ")

	rows, err := s.db.QueryContext(ctx, `
		SELECT p.method, COALESCE(SUM(p.amount),0), COUNT(*)
		FROM payments p JOIN sales s ON s.id = p.sale_id `+whereClause+` GROUP BY p.method ORDER BY SUM(p.amount) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PaymentMix{}
	for rows.Next() {
		var m PaymentMix
		if err := rows.Scan(&m.Method, &m.Amount, &m.Count); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) CashierMetrics(ctx context.Context, r DateRange) ([]CashierMetric, error) {
	var where []string
	var args []any
	where = append(where, "s.status != 'VOIDED'")
	r.apply(&where, &args)
	whereClause := "WHERE " + join(where, " AND ")

	rows, err := s.db.QueryContext(ctx, `
		SELECT s.cashier_id, u.name, COUNT(*), COALESCE(SUM(s.total),0)
		FROM sales s JOIN users u ON u.id = s.cashier_id
		`+whereClause+` GROUP BY s.cashier_id, u.name ORDER BY SUM(s.total) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CashierMetric{}
	for rows.Next() {
		var m CashierMetric
		if err := rows.Scan(&m.CashierID, &m.CashierName, &m.OrderCount, &m.Revenue); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func join(items []string, sep string) string {
	out := ""
	for i, it := range items {
		if i > 0 {
			out += sep
		}
		out += it
	}
	return out
}

// SummaryCSV renders a summary report as CSV for the §52/§93 export path.
func SummaryCSV(sum *Summary, top []TopProduct) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write([]string{"Metric", "Value"})
	w.WriteAll([][]string{
		{"Revenue", fmt.Sprintf("%d", sum.Revenue)},
		{"Orders", fmt.Sprintf("%d", sum.OrderCount)},
		{"Average Ticket", fmt.Sprintf("%d", sum.AverageTicket)},
		{"Discounts", fmt.Sprintf("%d", sum.DiscountTotal)},
		{"Tax", fmt.Sprintf("%d", sum.TaxTotal)},
		{"Refunds", fmt.Sprintf("%d", sum.RefundsTotal)},
		{"Cash", fmt.Sprintf("%d", sum.CashAmount)},
		{"Card", fmt.Sprintf("%d", sum.CardAmount)},
		{"Other", fmt.Sprintf("%d", sum.OtherAmount)},
	})
	w.Write([]string{})
	w.Write([]string{"Product", "Quantity Sold", "Sales Amount"})
	for _, p := range top {
		w.Write([]string{p.Name, fmt.Sprintf("%d", p.QuantitySold), fmt.Sprintf("%d", p.SalesAmount)})
	}
	w.Flush()
	return buf.Bytes()
}
