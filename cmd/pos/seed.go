package main

import (
	"context"
	"database/sql"
	"fmt"

	"pos/internal/security"
	"pos/internal/settings"
	"pos/internal/storage"
)

// seed populates a freshly migrated, empty database with the minimum data
// the application needs to be usable: permissions, roles, one location and
// register, one admin user, a default tax rate, and default settings. Demo
// catalog data is separate (seedDemoCatalog) and is only ever sample/demo
// content, never something business logic depends on (brief's own rule
// about the Casa Árabe example names).
func seed(ctx context.Context, db *storage.DB, sec *security.Service, set *settings.Service) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM permissions`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		for _, code := range security.AllPermissions {
			if _, err := db.ExecContext(ctx, `INSERT INTO permissions (code) VALUES (?)`, code); err != nil {
				return err
			}
		}
	}

	roleIDs, err := ensureRoles(ctx, db)
	if err != nil {
		return err
	}

	if err := set.Seed(ctx); err != nil {
		return err
	}

	var locationCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM locations`).Scan(&locationCount); err != nil {
		return err
	}
	var registerID int64
	if locationCount == 0 {
		res, err := db.ExecContext(ctx, `INSERT INTO locations (name, address) VALUES (?, ?)`, "Main Store", "")
		if err != nil {
			return err
		}
		locationID, _ := res.LastInsertId()
		res, err = db.ExecContext(ctx, `INSERT INTO registers (location_id, name) VALUES (?, ?)`, locationID, "Register 01")
		if err != nil {
			return err
		}
		registerID, _ = res.LastInsertId()
	} else {
		db.QueryRowContext(ctx, `SELECT id FROM registers LIMIT 1`).Scan(&registerID)
	}

	var userCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return err
	}
	if userCount == 0 {
		if _, err := sec.CreateUser(ctx, security.CreateUserInput{Name: "Administrator", Username: "admin", PIN: "1234", RoleID: roleIDs["Admin"]}); err != nil {
			return err
		}
		if _, err := sec.CreateUser(ctx, security.CreateUserInput{Name: "Cashier One", Username: "cashier", PIN: "1111", RoleID: roleIDs["Cashier"]}); err != nil {
			return err
		}
		fmt.Println("seed: created default users admin/1234 (Admin) and cashier/1111 (Cashier) — change these PINs immediately")
	}

	var taxCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tax_rates`).Scan(&taxCount); err != nil {
		return err
	}
	if taxCount == 0 {
		if _, err := db.ExecContext(ctx, `INSERT INTO tax_rates (name, rate_bps, inclusive) VALUES (?, ?, ?)`, "Standard VAT", 2100, 0); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO tax_rates (name, rate_bps, inclusive) VALUES (?, ?, ?)`, "Reduced VAT", 1000, 0); err != nil {
			return err
		}
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO printers (name, type, connection, config_json, paper_width_mm, is_default_receipt)
		SELECT 'Simulated Receipt Printer', 'RECEIPT', 'SIMULATED', '{}', 80, 1 WHERE NOT EXISTS (SELECT 1 FROM printers)`); err != nil {
		return err
	}

	return seedDemoCatalog(ctx, db)
}

func ensureRoles(ctx context.Context, db *storage.DB) (map[string]int64, error) {
	ids := map[string]int64{}
	roleDefs := map[string][]string{
		"Admin":   security.AllPermissions,
		"Manager": append(append([]string{}, security.CashierPermissions...), security.PermProductWrite, security.PermInventoryAdjust, security.PermSettingsWrite, security.PermDiscountOverride, security.PermPriceOverride, security.PermUsersManage),
		"Cashier": security.CashierPermissions,
	}
	for _, name := range []string{"Admin", "Manager", "Cashier"} {
		var id int64
		err := db.QueryRowContext(ctx, `SELECT id FROM roles WHERE name = ?`, name).Scan(&id)
		if err == sql.ErrNoRows {
			res, err := db.ExecContext(ctx, `INSERT INTO roles (name) VALUES (?)`, name)
			if err != nil {
				return nil, err
			}
			id, _ = res.LastInsertId()
		} else if err != nil {
			return nil, err
		}
		ids[name] = id

		for _, code := range roleDefs[name] {
			var permID int64
			if err := db.QueryRowContext(ctx, `SELECT id FROM permissions WHERE code = ?`, code).Scan(&permID); err != nil {
				continue
			}
			if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO role_permissions (role_id, permission_id) VALUES (?, ?)`, id, permID); err != nil {
				return nil, err
			}
		}
	}
	return ids, nil
}

// seedDemoCatalog adds a small illustrative menu so the Sell screen isn't
// empty on first run. These are sample names only, per the architecture
// doc's own rule: never load-bearing for business logic, freely replaced
// from the Products screen.
func seedDemoCatalog(ctx context.Context, db *storage.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM categories`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var taxID int64
	db.QueryRowContext(ctx, `SELECT id FROM tax_rates ORDER BY id LIMIT 1`).Scan(&taxID)

	categories := []string{"Grill & Meat", "Salads", "Desserts", "Drinks"}
	catIDs := map[string]int64{}
	for i, name := range categories {
		res, err := db.ExecContext(ctx, `INSERT INTO categories (name, sort_order) VALUES (?, ?)`, name, i)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		catIDs[name] = id
	}

	type demoProduct struct {
		sku, name, category string
		price               int64
		barcode             string
	}
	products := []demoProduct{
		{"GRL-001", "Grilled Chicken Skewer", "Grill & Meat", 950, "1000000000011"},
		{"GRL-002", "Lamb Kofta", "Grill & Meat", 1150, "1000000000028"},
		{"SAL-001", "Mixed Salad", "Salads", 550, "1000000000035"},
		{"DES-001", "Baklava (3 pcs)", "Desserts", 450, "1000000000042"},
		{"DRK-001", "Mint Tea", "Drinks", 250, "1000000000059"},
		{"DRK-002", "Sparkling Water", "Drinks", 200, "1000000000066"},
	}
	for _, p := range products {
		res, err := db.ExecContext(ctx, `INSERT INTO products (sku, name, category_id, price, cost, tax_rate_id, track_stock, reorder_level, active) VALUES (?,?,?,?,?,?,1,5,1)`,
			p.sku, p.name, catIDs[p.category], p.price, p.price/2, taxID)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		if _, err := db.ExecContext(ctx, `INSERT INTO product_barcodes (product_id, barcode) VALUES (?, ?)`, id, p.barcode); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO inventory_balances (product_id, quantity) VALUES (?, 50)`, id); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO stock_movements (product_id, type, quantity, reference_type) VALUES (?, 'INITIAL', 50, 'seed')`, id); err != nil {
			return err
		}
	}
	return nil
}
