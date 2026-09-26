package printing

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	escInit       = "\x1b\x40"
	escAlignLeft  = "\x1b\x61\x00"
	escAlignCenter = "\x1b\x61\x01"
	escBoldOn     = "\x1b\x45\x01"
	escBoldOff    = "\x1b\x45\x00"
	escDoubleOn   = "\x1d\x21\x11"
	escDoubleOff  = "\x1d\x21\x00"
	escCutPartial = "\x1d\x56\x42\x00"
	// ESC p m t1 t2 — kick pin m for t1*2ms on, t2*2ms off. These are the
	// values almost every EPSON-compatible thermal printer's cash drawer
	// port accepts out of the box (brief §23).
	escDrawerKick = "\x1b\x70\x00\x19\xfa"
)

// BuildESCPOS renders a Receipt into a raw ESC/POS byte stream for a printer
// with the given character width (32 for 58mm paper, 48 for 80mm).
func BuildESCPOS(r Receipt, width int) []byte {
	var b bytes.Buffer
	b.WriteString(escInit)
	b.WriteString(escAlignCenter)
	b.WriteString(escDoubleOn)
	b.WriteString(centerLine(r.StoreName, width) + "\n")
	b.WriteString(escDoubleOff)
	if r.StoreAddress != "" {
		b.WriteString(wrapCenter(r.StoreAddress, width))
	}
	if r.StoreTaxID != "" {
		b.WriteString(centerLine(r.StoreTaxID, width) + "\n")
	}
	b.WriteString(strings.Repeat("-", width) + "\n")
	b.WriteString(escAlignLeft)
	b.WriteString(fmt.Sprintf("Receipt: %s\n", r.ReceiptNumber))
	b.WriteString(fmt.Sprintf("Date: %s\n", r.CreatedAt))
	b.WriteString(fmt.Sprintf("Cashier: %s\n", r.CashierName))
	if r.CustomerName != "" {
		b.WriteString(fmt.Sprintf("Customer: %s\n", r.CustomerName))
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
		qtyPrice := fmt.Sprintf("%d x %s", line.Quantity, formatMoney(line.UnitPrice, r.CurrencySymbol, r.CurrencyDecimals))
		total := formatMoney(line.LineTotal, r.CurrencySymbol, r.CurrencyDecimals)
		b.WriteString(twoColumn(qtyPrice, total, width) + "\n")
	}

	b.WriteString(strings.Repeat("-", width) + "\n")
	b.WriteString(twoColumn("Subtotal", formatMoney(r.Subtotal, r.CurrencySymbol, r.CurrencyDecimals), width) + "\n")
	if r.Discount != 0 {
		b.WriteString(twoColumn("Discount", "-"+formatMoney(r.Discount, r.CurrencySymbol, r.CurrencyDecimals), width) + "\n")
	}
	b.WriteString(twoColumn("Tax", formatMoney(r.Tax, r.CurrencySymbol, r.CurrencyDecimals), width) + "\n")
	b.WriteString(escBoldOn)
	b.WriteString(twoColumn("TOTAL", formatMoney(r.Total, r.CurrencySymbol, r.CurrencyDecimals), width) + "\n")
	b.WriteString(escBoldOff)
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
	b.WriteString(escAlignCenter)
	footer := r.FooterMessage
	if footer == "" {
		footer = "Thank you"
	}
	b.WriteString(centerLine(footer, width) + "\n\n\n")
	b.WriteString(escCutPartial)
	return b.Bytes()
}

// DrawerKickCommand returns the raw bytes that pulse the cash drawer,
// normally sent down the same printer transport (brief §23).
func DrawerKickCommand() []byte {
	return []byte(escDrawerKick)
}

func formatMoney(m interface{ Format(int) string }, symbol string, decimals int) string {
	return m.Format(decimals) + " " + symbol
}

func methodLabel(method string) string {
	switch method {
	case "CASH":
		return "Cash"
	case "CARD":
		return "Card"
	case "BANK_TRANSFER":
		return "Transfer"
	case "GIFT_CARD":
		return "Gift Card"
	case "MOBILE":
		return "Mobile"
	default:
		return strings.Title(strings.ToLower(method))
	}
}

func twoColumn(left, right string, width int) string {
	if len(left)+len(right)+1 > width {
		left = left[:max0(width-len(right)-1)]
	}
	pad := width - len(left) - len(right)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
}

func centerLine(s string, width int) string {
	if len(s) >= width {
		return s
	}
	padTotal := width - len(s)
	left := padTotal / 2
	return strings.Repeat(" ", left) + s
}

func wrapCenter(s string, width int) string {
	words := strings.Fields(s)
	var b bytes.Buffer
	line := ""
	for _, w := range words {
		if len(line)+len(w)+1 > width {
			b.WriteString(centerLine(line, width) + "\n")
			line = w
		} else if line == "" {
			line = w
		} else {
			line += " " + w
		}
	}
	if line != "" {
		b.WriteString(centerLine(line, width) + "\n")
	}
	return b.String()
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
