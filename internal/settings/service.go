// Package settings is a small typed wrapper around the generic key/value
// `settings` table. Business configuration (currency, tax defaults,
// discount limits, loyalty rate, receipt header, language) lives here in
// SQLite, never hard-coded and never in browser localStorage (brief §5/§91).
package settings

import (
	"context"
	"encoding/json"
	"strconv"

	"pos/internal/storage"
)

const (
	KeyCurrencyCode        = "currency.code"
	KeyCurrencyDecimals    = "currency.decimals"
	KeyCurrencySymbol      = "currency.symbol"
	KeyLanguage            = "language"
	KeyStoreName           = "store.name"
	KeyStoreAddress        = "store.address"
	KeyStoreTaxID          = "store.tax_id"
	KeyLoyaltyRatePer100   = "loyalty.rate_per_100" // points earned per 100 minor units spent
	KeyDiscountMaxPercent  = "discount.max_percent_bps" // cashier self-serve ceiling before manager override required
	KeySoundEnabled        = "ui.sound_enabled"
	KeyScannerSuffixKey    = "scanner.suffix_key"
	KeyScannerMaxIntervalMs = "scanner.max_interval_ms"
	KeyScannerMinLength    = "scanner.min_length"
)

// Defaults seeded on first run. A merchant changes these from Settings, not
// by editing code.
var Defaults = map[string]string{
	KeyCurrencyCode:         "EUR",
	KeyCurrencyDecimals:     "2",
	KeyCurrencySymbol:       "€",
	KeyLanguage:             "es",
	KeyStoreName:            "My Store",
	KeyStoreAddress:         "",
	KeyStoreTaxID:           "",
	KeyLoyaltyRatePer100:    "1",
	KeyDiscountMaxPercent:   "2000", // 20.00%
	KeySoundEnabled:         "true",
	KeyScannerSuffixKey:     "Enter",
	KeyScannerMaxIntervalMs: "50",
	KeyScannerMinLength:     "6",
}

type Service struct {
	db *storage.DB
}

func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Seed(ctx context.Context) error {
	for k, v := range Defaults {
		if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)`, k, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Get(ctx context.Context, key string) (string, bool) {
	row := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key)
	var v string
	if err := row.Scan(&v); err != nil {
		return "", false
	}
	return v, true
}

func (s *Service) GetInt64(ctx context.Context, key string, fallback int64) int64 {
	v, ok := s.Get(ctx, key)
	if !ok {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func (s *Service) GetBool(ctx context.Context, key string, fallback bool) bool {
	v, ok := s.Get(ctx, key)
	if !ok {
		return fallback
	}
	return v == "true" || v == "1"
}

func (s *Service) All(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

func (s *Service) SetAll(ctx context.Context, values map[string]string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for k, v := range values {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO settings (key, value, updated_at) VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))
			ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`, k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// AsJSON is a convenience for the API layer.
func (s *Service) AsJSON(ctx context.Context) (json.RawMessage, error) {
	all, err := s.All(ctx)
	if err != nil {
		return nil, err
	}
	return json.Marshal(all)
}
