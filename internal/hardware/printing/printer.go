// Package printing implements the hardware abstraction for receipt printing
// and the cash drawer kick that normally rides the same cable (brief §21-23,
// §32, §81). Sale completion and printing are separate concerns: a printer
// failure is reported to the caller but never rolls back an already
// committed sale (brief §80) — that rule is enforced by the API layer, which
// prints only after the sale transaction has committed.
package printing

import (
	"context"
	"pos/internal/domain"
)

// ReceiptLine is one printed line item.
type ReceiptLine struct {
	Name      string
	Variant   string
	Modifiers []string
	Quantity  int64
	UnitPrice domain.Money
	LineTotal domain.Money
}

type ReceiptPayment struct {
	Method    string
	Amount    domain.Money
	Tendered  domain.Money
	ChangeDue domain.Money
}

// Receipt is the fully-resolved, printer-agnostic content of a receipt. It
// is built from a committed sale's snapshot data, never recomputed from
// current catalog prices (brief §65).
type Receipt struct {
	StoreName       string
	StoreAddress    string
	StoreTaxID      string
	ReceiptNumber   string
	CashierName     string
	CreatedAt       string
	CustomerName    string
	Note            string
	Lines           []ReceiptLine
	Subtotal        domain.Money
	Discount        domain.Money
	Tax             domain.Money
	Total           domain.Money
	Payments        []ReceiptPayment
	ChangeDue       domain.Money
	CurrencySymbol  string
	CurrencyDecimals int
	FooterMessage   string
}

// ReceiptPrinter is the hardware boundary every concrete transport
// (simulated, Windows RAW spool, COM, LPT) implements identically, per
// brief §32.
type ReceiptPrinter interface {
	PrintReceipt(ctx context.Context, r Receipt) error
	PrintTest(ctx context.Context) error
	Cut(ctx context.Context) error
	OpenDrawer(ctx context.Context) error
}

// Config describes how to reach one physical printer, as stored in the
// `printers` table.
type Config struct {
	ID            int64
	Name          string
	Connection    string // SIMULATED, WINDOWS_SPOOL, COM, LPT
	Target        string // Windows printer name, or a device path like COM1/LPT1
	PaperWidthMM  int
}

// New builds the concrete ReceiptPrinter for a Config. Connection kinds that
// don't exist on the current OS (e.g. WINDOWS_SPOOL on Linux) fail closed
// with a clear error rather than silently no-opping, so a misconfiguration
// is visible in the printer test button rather than a swallowed failure at
// checkout time.
func New(cfg Config) ReceiptPrinter {
	width := cfg.PaperWidthMM
	if width == 0 {
		width = 80
	}
	switch cfg.Connection {
	case "COM", "LPT":
		return &SerialPrinter{Path: cfg.Target, CharsPerLine: charsPerLine(width)}
	case "WINDOWS_SPOOL":
		return newWindowsSpoolPrinter(cfg.Target, charsPerLine(width))
	default:
		return &SimulatedPrinter{Name: cfg.Name, CharsPerLine: charsPerLine(width)}
	}
}

func charsPerLine(paperWidthMM int) int {
	if paperWidthMM <= 58 {
		return 32
	}
	return 48
}
