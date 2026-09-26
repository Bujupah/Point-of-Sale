package shifts

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
)

type PaymentBreakdown struct {
	Method string `json:"method"`
	Amount int64  `json:"amount"`
	Count  int    `json:"count"`
}

type ZReport struct {
	Shift Shift `json:"shift"`

	GrossSales    int64 `json:"gross_sales"`
	NetSales      int64 `json:"net_sales"`
	DiscountTotal int64 `json:"discount_total"`
	TaxTotal      int64 `json:"tax_total"`
	RefundsTotal  int64 `json:"refunds_total"`

	Payments []PaymentBreakdown `json:"payments"`

	OpeningFloat int64 `json:"opening_float"`
	CashIn       int64 `json:"cash_in"`
	CashOut      int64 `json:"cash_out"`
	CashDrop     int64 `json:"cash_drop"`
	CashSales    int64 `json:"cash_sales"`
	CashRefunds  int64 `json:"cash_refunds"`
	ExpectedCash int64  `json:"expected_cash"`
	CountedCash  *int64 `json:"counted_cash,omitempty"`
	Difference   *int64 `json:"difference,omitempty"`

	OrderCount    int64 `json:"order_count"`
	AverageTicket int64 `json:"average_ticket"`
}

// Report builds the full Z-report for a shift, whether it is still open
// (a live preview for the counting UI) or already closed (the historical
// record).
func (s *Service) Report(ctx context.Context, shiftID int64) (*ZReport, error) {
	sh, err := s.Get(ctx, shiftID)
	if err != nil {
		return nil, err
	}
	r := &ZReport{Shift: *sh, OpeningFloat: sh.OpeningFloat, CountedCash: sh.CountedCash, Difference: sh.Difference}

	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(subtotal),0), COALESCE(SUM(total),0), COALESCE(SUM(discount_total),0), COALESCE(SUM(tax_total),0), COUNT(*)
		FROM sales WHERE shift_id = ? AND status != 'VOIDED'`, shiftID)
	if err := row.Scan(&r.GrossSales, &r.NetSales, &r.DiscountTotal, &r.TaxTotal, &r.OrderCount); err != nil {
		return nil, err
	}
	if r.OrderCount > 0 {
		r.AverageTicket = r.NetSales / r.OrderCount
	}

	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(total),0) FROM refunds WHERE shift_id = ?`, shiftID).Scan(&r.RefundsTotal); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT p.method, COALESCE(SUM(p.amount),0), COUNT(*)
		FROM payments p JOIN sales s ON s.id = p.sale_id
		WHERE s.shift_id = ? GROUP BY p.method`, shiftID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var b PaymentBreakdown
		if err := rows.Scan(&b.Method, &b.Amount, &b.Count); err != nil {
			rows.Close()
			return nil, err
		}
		if b.Method == "CASH" {
			r.CashSales = b.Amount
		}
		r.Payments = append(r.Payments, b)
	}
	rows.Close()

	movRows, err := s.db.QueryContext(ctx, `SELECT type, COALESCE(SUM(amount),0) FROM cash_movements WHERE shift_id = ? GROUP BY type`, shiftID)
	if err != nil {
		return nil, err
	}
	for movRows.Next() {
		var t string
		var amt int64
		if err := movRows.Scan(&t, &amt); err != nil {
			movRows.Close()
			return nil, err
		}
		switch t {
		case "CASH_IN":
			r.CashIn = amt
		case "CASH_OUT":
			r.CashOut = -amt
		case "CASH_DROP":
			r.CashDrop = -amt
		case "REFUND_CASH":
			r.CashRefunds = -amt
		}
	}
	movRows.Close()

	expected, err := s.ExpectedCash(ctx, shiftID)
	if err != nil {
		return nil, err
	}
	r.ExpectedCash = expected

	return r, nil
}

// CSV renders the Z-report as a flat CSV suitable for §43's "Print / CSV"
// export.
func (r *ZReport) CSV() []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	rows := [][]string{
		{"Shift", fmt.Sprintf("%d", r.Shift.ID)},
		{"Register", r.Shift.RegisterName},
		{"Cashier", r.Shift.CashierName},
		{"Opened", r.Shift.OpeningAt},
		{"Gross Sales", fmt.Sprintf("%d", r.GrossSales)},
		{"Discounts", fmt.Sprintf("%d", r.DiscountTotal)},
		{"Tax", fmt.Sprintf("%d", r.TaxTotal)},
		{"Net Sales", fmt.Sprintf("%d", r.NetSales)},
		{"Refunds", fmt.Sprintf("%d", r.RefundsTotal)},
		{"Opening Float", fmt.Sprintf("%d", r.OpeningFloat)},
		{"Cash In", fmt.Sprintf("%d", r.CashIn)},
		{"Cash Out", fmt.Sprintf("%d", r.CashOut)},
		{"Cash Drop", fmt.Sprintf("%d", r.CashDrop)},
		{"Expected Cash", fmt.Sprintf("%d", r.ExpectedCash)},
		{"Order Count", fmt.Sprintf("%d", r.OrderCount)},
		{"Average Ticket", fmt.Sprintf("%d", r.AverageTicket)},
	}
	if r.CountedCash != nil {
		rows = append(rows, []string{"Counted Cash", fmt.Sprintf("%d", *r.CountedCash)})
	}
	if r.Difference != nil {
		rows = append(rows, []string{"Difference", fmt.Sprintf("%d", *r.Difference)})
	}
	for _, p := range r.Payments {
		rows = append(rows, []string{"Payment: " + p.Method, fmt.Sprintf("%d", p.Amount)})
	}
	w.WriteAll(rows)
	w.Flush()
	return buf.Bytes()
}
