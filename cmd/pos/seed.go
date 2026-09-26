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

// seedDemoCatalog adds an illustrative menu so the Sell screen isn't empty
// on first run: Arabic mezze, mains and desserts as served at a Spanish
// Arabic restaurant. These are sample names only, per the architecture
// doc's own rule: never load-bearing for business logic, freely replaced
// from the Products screen. Each item ships with an original SVG
// illustration under assets/products/ (served at /media/products/<file>,
// see main.go) rather than a hot-linked photo, so the catalog renders
// correctly with zero network access, on the very first run, forever
// after — consistent with the whole app's offline-first design.
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

	categories := []string{"Mezze & Starters", "Salads", "Mains & Grill", "Bread", "Desserts", "Drinks"}
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
		barcode, image      string
	}
	products := []demoProduct{
		// Mezze & Starters
		{"MEZ-001", "Hummus", "Mezze & Starters", 495, "2000000000010", "hummus"},
		{"MEZ-002", "Baba Ghanoush", "Mezze & Starters", 525, "2000000000027", "baba-ghanoush"},
		{"MEZ-003", "Muhammara", "Mezze & Starters", 550, "2000000000034", "muhammara"},
		{"MEZ-004", "Labneh with Za'atar", "Mezze & Starters", 475, "2000000000041", "labneh"},
		{"MEZ-005", "Falafel (6 pcs)", "Mezze & Starters", 595, "2000000000058", "falafel"},
		{"MEZ-006", "Samosa (3 pcs)", "Mezze & Starters", 550, "2000000000065", "samosa"},
		{"MEZ-007", "Dolma - Stuffed Vine Leaves", "Mezze & Starters", 625, "2000000000072", "dolma"},
		// Salads
		{"SAL-001", "Tabbouleh", "Salads", 595, "2000000000089", "tabbouleh"},
		{"SAL-002", "Fattoush", "Salads", 625, "2000000000096", "fattoush"},
		// Mains & Grill
		{"MAI-001", "Chicken Shawarma Plate", "Mains & Grill", 1195, "2000000000102", "shawarma"},
		{"MAI-002", "Lamb Kofta Kebab", "Mains & Grill", 1350, "2000000000119", "kofta-kebab"},
		{"MAI-003", "Chicken Kebab Skewers", "Mains & Grill", 1275, "2000000000126", "chicken-kebab"},
		{"MAI-004", "Couscous Royal", "Mains & Grill", 1450, "2000000000133", "couscous-royal"},
		{"MAI-005", "Lamb Tagine", "Mains & Grill", 1595, "2000000000140", "lamb-tagine"},
		{"MAI-006", "Mixed Grill Platter", "Mains & Grill", 1850, "2000000000157", "mixed-grill"},
		// Bread
		{"BRD-001", "Pita Bread", "Bread", 195, "2000000000164", "pita-bread"},
		{"BRD-002", "Za'atar Manakish", "Bread", 395, "2000000000171", "manakish"},
		// Desserts
		{"DES-001", "Baklava (3 pcs)", "Desserts", 450, "2000000000188", "baklava"},
		{"DES-002", "Kunafa", "Desserts", 525, "2000000000195", "kunafa"},
		{"DES-003", "Basbousa", "Desserts", 425, "2000000000201", "basbousa"},
		// Drinks
		{"DRK-001", "Moroccan Mint Tea", "Drinks", 275, "2000000000218", "mint-tea"},
		{"DRK-002", "Turkish Coffee", "Drinks", 295, "2000000000225", "turkish-coffee"},
		{"DRK-003", "Ayran", "Drinks", 250, "2000000000232", "ayran"},
		{"DRK-004", "Karkade - Hibiscus Tea", "Drinks", 275, "2000000000249", "karkade"},
	}
	for _, p := range products {
		image := "/media/products/" + p.image + ".svg"
		res, err := db.ExecContext(ctx, `INSERT INTO products (sku, name, category_id, price, cost, tax_rate_id, track_stock, reorder_level, image_thumbnail, image_medium, active) VALUES (?,?,?,?,?,?,1,8,?,?,1)`,
			p.sku, p.name, catIDs[p.category], p.price, p.price*4/10, taxID, image, image)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		if _, err := db.ExecContext(ctx, `INSERT INTO product_barcodes (product_id, barcode) VALUES (?, ?)`, id, p.barcode); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO inventory_balances (product_id, quantity) VALUES (?, 60)`, id); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO stock_movements (product_id, type, quantity, reference_type) VALUES (?, 'INITIAL', 60, 'seed')`, id); err != nil {
			return err
		}
	}
	return nil
}
