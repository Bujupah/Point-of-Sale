package printing

import (
	"strconv"
	"strings"
)

// AsciiPreview renders the same content as BuildESCPOS but as plain
// monospaced text with no printer control codes, for the on-screen receipt
// preview (brief §33 — a representative preview, not a pixel-exact one).
func AsciiPreview(r Receipt, width int) string {
	var b strings.Builder
	b.WriteString(centerLine(r.StoreName, width) + "\n")
	if r.StoreAddress != "" {
		b.WriteString(wrapCenter(r.StoreAddress, width))
	}
	if r.StoreTaxID != "" {
		b.WriteString(centerLine(r.StoreTaxID, width) + "\n")
	}
	b.WriteString(strings.Repeat("-", width) + "\n")
	b.WriteString("Receipt: " + r.ReceiptNumber + "\n")
	b.WriteString("Date: " + r.CreatedAt + "\n")
	b.WriteString("Cashier: " + r.CashierName + "\n")
	if r.CustomerName != "" {
		b.WriteString("Customer: " + r.CustomerName + "\n")
	}
	b.WriteString(strings.Repeat("-", width) + "\n")

	for _, line := range r.Lines {
		name := line.Name
		if line.Variant != "" {
			name += " (" + line.Variant + ")"
		}
		b.WriteString(name + "\n")
		for _, m := range line.Modifiers {
			b.WriteString("  + " + m + "\n")
		}
		qtyPrice := formatQty(line.Quantity) + " x " + formatMoney(line.UnitPrice, r.CurrencySymbol, r.CurrencyDecimals)
		total := formatMoney(line.LineTotal, r.CurrencySymbol, r.CurrencyDecimals)
		b.WriteString(twoColumn(qtyPrice, total, width) + "\n")
	}

	b.WriteString(strings.Repeat("-", width) + "\n")
	b.WriteString(twoColumn("Subtotal", formatMoney(r.Subtotal, r.CurrencySymbol, r.CurrencyDecimals), width) + "\n")
	if r.Discount != 0 {
		b.WriteString(twoColumn("Discount", "-"+formatMoney(r.Discount, r.CurrencySymbol, r.CurrencyDecimals), width) + "\n")
	}
	b.WriteString(twoColumn("Tax", formatMoney(r.Tax, r.CurrencySymbol, r.CurrencyDecimals), width) + "\n")
	b.WriteString(twoColumn("TOTAL", formatMoney(r.Total, r.CurrencySymbol, r.CurrencyDecimals), width) + "\n")
	b.WriteString(strings.Repeat("-", width) + "\n")

	for _, p := range r.Payments {
		b.WriteString(twoColumn(methodLabel(p.Method), formatMoney(p.Amount, r.CurrencySymbol, r.CurrencyDecimals), width) + "\n")
	}
	if r.ChangeDue > 0 {
		b.WriteString(twoColumn("Change", formatMoney(r.ChangeDue, r.CurrencySymbol, r.CurrencyDecimals), width) + "\n")
	}
	if r.Note != "" {
		b.WriteString(strings.Repeat("-", width) + "\n")
		b.WriteString(r.Note + "\n")
	}
	b.WriteString("\n")
	footer := r.FooterMessage
	if footer == "" {
		footer = "Thank you"
	}
	b.WriteString(centerLine(footer, width) + "\n")
	return b.String()
}

func formatQty(q int64) string {
	return strconv.FormatInt(q, 10)
}
