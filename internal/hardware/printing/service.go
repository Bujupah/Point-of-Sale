package printing

import (
	"context"
	"database/sql"
	"errors"

	"pos/internal/sales"
	"pos/internal/settings"
	"pos/internal/storage"
)

type Service struct {
	db       *storage.DB
	settings *settings.Service
}

func NewService(db *storage.DB, set *settings.Service) *Service {
	return &Service{db: db, settings: set}
}

func (s *Service) defaultReceiptPrinter(ctx context.Context) (ReceiptPrinter, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, connection, config_json, paper_width_mm FROM printers WHERE is_default_receipt = 1 LIMIT 1`)
	var cfg Config
	var configJSON string
	if err := row.Scan(&cfg.ID, &cfg.Name, &cfg.Connection, &configJSON, &cfg.PaperWidthMM); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No printer configured yet: fall back to a named simulated
			// printer so Sell/checkout never blocks on hardware setup.
			return &SimulatedPrinter{Name: "default", CharsPerLine: 48}, nil
		}
		return nil, err
	}
	cfg.Target = extractTarget(configJSON)
	return New(cfg), nil
}

func extractTarget(configJSON string) string {
	// config_json is a small {"target": "..."} document; avoid pulling in a
	// struct for one field.
	const key = `"target"`
	idx := indexOf(configJSON, key)
	if idx < 0 {
		return ""
	}
	rest := configJSON[idx+len(key):]
	start := indexOf(rest, `"`) + 1
	if start <= 0 {
		return ""
	}
	rest = rest[start:]
	end := indexOf(rest, `"`)
	if end < 0 {
		return rest
	}
	return rest[:end]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func (s *Service) buildReceipt(ctx context.Context, sale *sales.Sale) (Receipt, error) {
	all, err := s.settings.All(ctx)
	if err != nil {
		return Receipt{}, err
	}
	decimals := 2
	if v, ok := all[settings.KeyCurrencyDecimals]; ok {
		if v == "3" {
			decimals = 3
		}
	}
	r := Receipt{
		StoreName:        all[settings.KeyStoreName],
		StoreAddress:     all[settings.KeyStoreAddress],
		StoreTaxID:       all[settings.KeyStoreTaxID],
		ReceiptNumber:    sale.ReceiptNumber,
		CashierName:      sale.CashierName,
		CreatedAt:        sale.CreatedAt,
		CustomerName:     sale.CustomerName,
		Note:             sale.Note,
		Subtotal:         sale.Subtotal,
		Discount:         sale.DiscountTotal,
		Tax:              sale.TaxTotal,
		Total:            sale.Total,
		ChangeDue:        sale.ChangeDue,
		CurrencySymbol:   all[settings.KeyCurrencySymbol],
		CurrencyDecimals: decimals,
	}
	for _, it := range sale.Items {
		r.Lines = append(r.Lines, ReceiptLine{
			Name: it.Name, Variant: it.Variant, Modifiers: it.Modifiers,
			Quantity: it.Quantity, UnitPrice: it.UnitPrice, LineTotal: it.LineTotal,
		})
	}
	for _, p := range sale.Payments {
		r.Payments = append(r.Payments, ReceiptPayment{Method: p.Method, Amount: p.Amount, Tendered: p.Tendered, ChangeDue: p.ChangeDue})
	}
	return r, nil
}

// PrintSale prints a completed sale's receipt. Called only after the sale
// transaction has committed; any error here is reported to the caller as a
// printer problem, never as a reason to undo the sale (brief §80).
func (s *Service) PrintSale(ctx context.Context, sale *sales.Sale) error {
	printer, err := s.defaultReceiptPrinter(ctx)
	if err != nil {
		return err
	}
	receipt, err := s.buildReceipt(ctx, sale)
	if err != nil {
		return err
	}
	return printer.PrintReceipt(ctx, receipt)
}

// PreviewSale renders the ASCII receipt preview without touching hardware.
func (s *Service) PreviewSale(ctx context.Context, sale *sales.Sale) (string, error) {
	receipt, err := s.buildReceipt(ctx, sale)
	if err != nil {
		return "", err
	}
	return AsciiPreview(receipt, 48), nil
}

func (s *Service) TestPrint(ctx context.Context) error {
	printer, err := s.defaultReceiptPrinter(ctx)
	if err != nil {
		return err
	}
	return printer.PrintTest(ctx)
}

func (s *Service) OpenDrawer(ctx context.Context) error {
	printer, err := s.defaultReceiptPrinter(ctx)
	if err != nil {
		return err
	}
	return printer.OpenDrawer(ctx)
}
