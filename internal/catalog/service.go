package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"pos/internal/domain"
	"pos/internal/storage"
)

type Service struct {
	db *storage.DB
}

func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

// ---------- Categories ----------

func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, parent_id, name, name_translations, sort_order, active FROM categories WHERE active = 1 ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		var parentID sql.NullInt64
		var namesJSON string
		var active int
		if err := rows.Scan(&c.ID, &parentID, &c.Name, &namesJSON, &c.Sort, &active); err != nil {
			return nil, err
		}
		if parentID.Valid {
			v := parentID.Int64
			c.ParentID = &v
		}
		c.Active = active == 1
		_ = json.Unmarshal([]byte(namesJSON), &c.Names)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) CreateCategory(ctx context.Context, c Category) (*Category, error) {
	namesJSON, _ := json.Marshal(c.Names)
	res, err := s.db.ExecContext(ctx, `INSERT INTO categories (parent_id, name, name_translations, sort_order, active) VALUES (?, ?, ?, ?, 1)`,
		c.ParentID, c.Name, string(namesJSON), c.Sort)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	c.ID = id
	c.Active = true
	return &c, nil
}

func (s *Service) UpdateCategory(ctx context.Context, id int64, c Category) error {
	namesJSON, _ := json.Marshal(c.Names)
	_, err := s.db.ExecContext(ctx, `UPDATE categories SET parent_id=?, name=?, name_translations=?, sort_order=?, active=? WHERE id=?`,
		c.ParentID, c.Name, string(namesJSON), c.Sort, boolToInt(c.Active), id)
	return err
}

// ---------- Modifier group library ----------

func (s *Service) ListModifierGroups(ctx context.Context) ([]ModifierGroup, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, min_select, max_select, required FROM modifier_groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := []ModifierGroup{}
	for rows.Next() {
		var g ModifierGroup
		var required int
		if err := rows.Scan(&g.ID, &g.Name, &g.MinSelect, &g.MaxSelect, &required); err != nil {
			return nil, err
		}
		g.Required = required == 1
		groups = append(groups, g)
	}
	rows.Close()
	for i := range groups {
		opts, err := s.modifierOptions(ctx, groups[i].ID)
		if err != nil {
			return nil, err
		}
		groups[i].Options = opts
	}
	return groups, nil
}

func (s *Service) modifierOptions(ctx context.Context, groupID int64) ([]ModifierOption, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, price_adjustment, sort_order FROM modifier_options WHERE group_id=? ORDER BY sort_order`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ModifierOption{}
	for rows.Next() {
		var o ModifierOption
		if err := rows.Scan(&o.ID, &o.Name, &o.PriceAdjustment, &o.Sort); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Service) CreateModifierGroup(ctx context.Context, g ModifierGroup) (*ModifierGroup, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO modifier_groups (name, min_select, max_select, required) VALUES (?, ?, ?, ?)`,
		g.Name, g.MinSelect, g.MaxSelect, boolToInt(g.Required))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	g.ID = id
	for i, opt := range g.Options {
		if _, err := tx.ExecContext(ctx, `INSERT INTO modifier_options (group_id, name, price_adjustment, sort_order) VALUES (?, ?, ?, ?)`,
			id, opt.Name, opt.PriceAdjustment, i); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &g, nil
}

// ---------- Products ----------

func (s *Service) SearchProducts(ctx context.Context, q string, categoryID *int64, activeOnly bool, limit, offset int) ([]Product, error) {
	if limit <= 0 || limit > 200 {
		limit = 60
	}
	var sb strings.Builder
	sb.WriteString(`SELECT p.id, p.sku, p.name, p.name_translations, p.description, p.category_id, p.price, p.cost,
		p.original_price, p.tax_rate_id, COALESCE(t.rate_bps,0), COALESCE(t.inclusive,0),
		p.track_stock, COALESCE(ib.quantity,0), p.reorder_level, p.image_thumbnail, p.image_medium,
		p.badge, p.is_open_item, p.active, p.favorite
		FROM products p
		LEFT JOIN tax_rates t ON t.id = p.tax_rate_id
		LEFT JOIN inventory_balances ib ON ib.product_id = p.id
		LEFT JOIN product_barcodes pb ON pb.product_id = p.id
		WHERE 1=1`)
	var args []any
	if activeOnly {
		sb.WriteString(` AND p.active = 1`)
	}
	if categoryID != nil {
		sb.WriteString(` AND p.category_id = ?`)
		args = append(args, *categoryID)
	}
	if q != "" {
		sb.WriteString(` AND (p.name LIKE ? OR p.name_translations LIKE ? OR p.sku LIKE ? OR p.description LIKE ? OR pb.barcode LIKE ?)`)
		like := "%" + q + "%"
		args = append(args, like, like, like, like, like)
	}
	sb.WriteString(` GROUP BY p.id ORDER BY p.favorite DESC, p.name LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range products {
		products[i].Barcodes, _ = s.barcodesFor(ctx, products[i].ID)
		computeStockStatus(&products[i])
	}
	return products, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProduct(rows rowScanner) (*Product, error) {
	var p Product
	var namesJSON string
	var categoryID sql.NullInt64
	var originalPrice sql.NullInt64
	var taxRateID sql.NullInt64
	var thumb, medium, badge sql.NullString
	var trackStock, isOpenItem, active, favorite int
	if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &namesJSON, &p.Description, &categoryID, &p.Price, &p.Cost,
		&originalPrice, &taxRateID, &p.TaxRateBps, &p.TaxInclusive,
		&trackStock, &p.Stock, &p.ReorderLevel, &thumb, &medium,
		&badge, &isOpenItem, &active, &favorite); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(namesJSON), &p.Names)
	if categoryID.Valid {
		v := categoryID.Int64
		p.CategoryID = &v
	}
	if originalPrice.Valid {
		v := domain.Money(originalPrice.Int64)
		p.OriginalPrice = &v
	}
	if taxRateID.Valid {
		v := taxRateID.Int64
		p.TaxRateID = &v
	}
	p.ImageThumbnail = thumb.String
	p.ImageMedium = medium.String
	p.Badge = badge.String
	p.TrackStock = trackStock == 1
	p.IsOpenItem = isOpenItem == 1
	p.Active = active == 1
	p.Favorite = favorite == 1
	return &p, nil
}

func computeStockStatus(p *Product) {
	if !p.TrackStock {
		return
	}
	if p.Stock <= 0 {
		p.StockStatus = "OUT_OF_STOCK"
	} else if p.Stock <= p.ReorderLevel {
		p.StockStatus = "LOW_STOCK"
	}
}

func (s *Service) barcodesFor(ctx context.Context, productID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT barcode FROM product_barcodes WHERE product_id = ?`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var b string
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

const productSelectCols = `p.id, p.sku, p.name, p.name_translations, p.description, p.category_id, p.price, p.cost,
		p.original_price, p.tax_rate_id, COALESCE(t.rate_bps,0), COALESCE(t.inclusive,0),
		p.track_stock, COALESCE(ib.quantity,0), p.reorder_level, p.image_thumbnail, p.image_medium,
		p.badge, p.is_open_item, p.active, p.favorite`

func (s *Service) GetProduct(ctx context.Context, id int64) (*Product, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+productSelectCols+`
		FROM products p
		LEFT JOIN tax_rates t ON t.id = p.tax_rate_id
		LEFT JOIN inventory_balances ib ON ib.product_id = p.id
		WHERE p.id = ?`, id)
	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if err := s.hydrateProduct(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) GetProductByBarcode(ctx context.Context, barcode string) (*Product, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+productSelectCols+`
		FROM products p
		LEFT JOIN tax_rates t ON t.id = p.tax_rate_id
		LEFT JOIN inventory_balances ib ON ib.product_id = p.id
		JOIN product_barcodes pb ON pb.product_id = p.id
		WHERE pb.barcode = ?`, barcode)
	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError("PRODUCT_NOT_FOUND", "No product matches this barcode", 404)
		}
		return nil, err
	}
	if err := s.hydrateProduct(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) hydrateProduct(ctx context.Context, p *Product) error {
	var err error
	p.Barcodes, err = s.barcodesFor(ctx, p.ID)
	if err != nil {
		return err
	}
	computeStockStatus(p)

	groupRows, err := s.db.QueryContext(ctx, `SELECT id, name, required, sort_order FROM product_variant_groups WHERE product_id = ? ORDER BY sort_order`, p.ID)
	if err != nil {
		return err
	}
	groups := []VariantGroup{}
	for groupRows.Next() {
		var g VariantGroup
		var required int
		if err := groupRows.Scan(&g.ID, &g.Name, &required, &g.Sort); err != nil {
			groupRows.Close()
			return err
		}
		g.Required = required == 1
		groups = append(groups, g)
	}
	groupRows.Close()
	for i := range groups {
		vRows, err := s.db.QueryContext(ctx, `SELECT id, name, price_adjustment, sku_suffix, sort_order FROM product_variants WHERE group_id = ? ORDER BY sort_order`, groups[i].ID)
		if err != nil {
			return err
		}
		for vRows.Next() {
			var v Variant
			if err := vRows.Scan(&v.ID, &v.Name, &v.PriceAdjustment, &v.SKUSuffix, &v.Sort); err != nil {
				vRows.Close()
				return err
			}
			groups[i].Variants = append(groups[i].Variants, v)
		}
		vRows.Close()
	}
	p.VariantGroups = groups

	modRows, err := s.db.QueryContext(ctx, `
		SELECT mg.id, mg.name, mg.min_select, mg.max_select, mg.required
		FROM product_modifier_groups pmg
		JOIN modifier_groups mg ON mg.id = pmg.modifier_group_id
		WHERE pmg.product_id = ? ORDER BY pmg.sort_order`, p.ID)
	if err != nil {
		return err
	}
	mods := []ModifierGroup{}
	for modRows.Next() {
		var g ModifierGroup
		var required int
		if err := modRows.Scan(&g.ID, &g.Name, &g.MinSelect, &g.MaxSelect, &required); err != nil {
			modRows.Close()
			return err
		}
		g.Required = required == 1
		mods = append(mods, g)
	}
	modRows.Close()
	for i := range mods {
		opts, err := s.modifierOptions(ctx, mods[i].ID)
		if err != nil {
			return err
		}
		mods[i].Options = opts
	}
	p.ModifierGroups = mods
	return nil
}

type ProductInput struct {
	SKU            string        `json:"sku"`
	Name           string        `json:"name"`
	Names          map[string]string `json:"names,omitempty"`
	Description    string        `json:"description"`
	CategoryID     *int64        `json:"category_id,omitempty"`
	Price          domain.Money  `json:"price"`
	Cost           domain.Money  `json:"cost"`
	OriginalPrice  *domain.Money `json:"original_price,omitempty"`
	TaxRateID      *int64        `json:"tax_rate_id,omitempty"`
	TrackStock     bool          `json:"track_stock"`
	InitialStock   int64         `json:"initial_stock"`
	ReorderLevel   int64         `json:"reorder_level"`
	ImageThumbnail string        `json:"image_thumbnail,omitempty"`
	ImageMedium    string        `json:"image_medium,omitempty"`
	Badge          string        `json:"badge,omitempty"`
	IsOpenItem     bool          `json:"is_open_item"`
	Active         bool          `json:"active"`
	Favorite       bool          `json:"favorite"`
	Barcodes       []string      `json:"barcodes,omitempty"`
	VariantGroups  []VariantGroup    `json:"variant_groups,omitempty"`
	ModifierGroupIDs []int64     `json:"modifier_group_ids,omitempty"`
}

func (s *Service) CreateProduct(ctx context.Context, in ProductInput, userID int64) (*Product, error) {
	if in.SKU == "" || in.Name == "" {
		return nil, domain.NewError("INVALID_INPUT", "SKU and name are required", 400)
	}
	namesJSON, _ := json.Marshal(in.Names)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `INSERT INTO products
		(sku, name, name_translations, description, category_id, price, cost, original_price, tax_rate_id,
		 track_stock, reorder_level, image_thumbnail, image_medium, badge, is_open_item, active, favorite)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		in.SKU, in.Name, string(namesJSON), in.Description, in.CategoryID, in.Price, in.Cost, in.OriginalPrice, in.TaxRateID,
		boolToInt(in.TrackStock), in.ReorderLevel, nullIfEmpty(in.ImageThumbnail), nullIfEmpty(in.ImageMedium), in.Badge,
		boolToInt(in.IsOpenItem), boolToInt(in.Active), boolToInt(in.Favorite))
	if err != nil {
		if isUniqueConstraint(err) {
			return nil, domain.NewError("SKU_EXISTS", "A product with this SKU already exists", 409)
		}
		return nil, err
	}
	id, _ := res.LastInsertId()

	if err := writeProductChildren(ctx, tx, id, in); err != nil {
		return nil, err
	}

	if in.TrackStock {
		if _, err := tx.ExecContext(ctx, `INSERT INTO inventory_balances (product_id, quantity) VALUES (?, ?)`, id, in.InitialStock); err != nil {
			return nil, err
		}
		if in.InitialStock != 0 {
			if _, err := tx.ExecContext(ctx, `INSERT INTO stock_movements (product_id, type, quantity, reference_type, created_by) VALUES (?, 'INITIAL', ?, 'product_create', ?)`,
				id, in.InitialStock, userID); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetProduct(ctx, id)
}

func (s *Service) UpdateProduct(ctx context.Context, id int64, in ProductInput) (*Product, error) {
	namesJSON, _ := json.Marshal(in.Names)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `UPDATE products SET
		sku=?, name=?, name_translations=?, description=?, category_id=?, price=?, cost=?, original_price=?, tax_rate_id=?,
		track_stock=?, reorder_level=?, image_thumbnail=?, image_medium=?, badge=?, is_open_item=?, active=?, favorite=?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id=?`,
		in.SKU, in.Name, string(namesJSON), in.Description, in.CategoryID, in.Price, in.Cost, in.OriginalPrice, in.TaxRateID,
		boolToInt(in.TrackStock), in.ReorderLevel, nullIfEmpty(in.ImageThumbnail), nullIfEmpty(in.ImageMedium), in.Badge,
		boolToInt(in.IsOpenItem), boolToInt(in.Active), boolToInt(in.Favorite), id)
	if err != nil {
		if isUniqueConstraint(err) {
			return nil, domain.NewError("SKU_EXISTS", "A product with this SKU already exists", 409)
		}
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, domain.ErrNotFound
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM product_barcodes WHERE product_id = ?`, id); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM product_variants WHERE group_id IN (SELECT id FROM product_variant_groups WHERE product_id = ?)`, id); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM product_variant_groups WHERE product_id = ?`, id); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM product_modifier_groups WHERE product_id = ?`, id); err != nil {
		return nil, err
	}
	if err := writeProductChildren(ctx, tx, id, in); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetProduct(ctx, id)
}

func writeProductChildren(ctx context.Context, tx *sql.Tx, productID int64, in ProductInput) error {
	for _, bc := range in.Barcodes {
		if bc == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO product_barcodes (product_id, barcode) VALUES (?, ?)`, productID, bc); err != nil {
			if isUniqueConstraint(err) {
				return domain.NewError("BARCODE_EXISTS", fmt.Sprintf("Barcode %s is already assigned to another product", bc), 409)
			}
			return err
		}
	}
	for gi, g := range in.VariantGroups {
		res, err := tx.ExecContext(ctx, `INSERT INTO product_variant_groups (product_id, name, required, sort_order) VALUES (?, ?, ?, ?)`,
			productID, g.Name, boolToInt(g.Required), gi)
		if err != nil {
			return err
		}
		groupID, _ := res.LastInsertId()
		for vi, v := range g.Variants {
			if _, err := tx.ExecContext(ctx, `INSERT INTO product_variants (group_id, name, price_adjustment, sku_suffix, sort_order) VALUES (?, ?, ?, ?, ?)`,
				groupID, v.Name, v.PriceAdjustment, v.SKUSuffix, vi); err != nil {
				return err
			}
		}
	}
	for i, modGroupID := range in.ModifierGroupIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO product_modifier_groups (product_id, modifier_group_id, sort_order) VALUES (?, ?, ?)`,
			productID, modGroupID, i); err != nil {
			return err
		}
	}
	return nil
}

func isUniqueConstraint(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
