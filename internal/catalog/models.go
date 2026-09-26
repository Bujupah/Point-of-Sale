// Package catalog owns categories, products, barcodes, variants, and
// modifiers — the sellable menu/catalog the Sell screen renders from.
package catalog

import "pos/internal/domain"

type Category struct {
	ID       int64             `json:"id"`
	ParentID *int64            `json:"parent_id,omitempty"`
	Name     string            `json:"name"`
	Names    map[string]string `json:"names,omitempty"`
	Sort     int               `json:"sort_order"`
	Active   bool              `json:"active"`
}

type Product struct {
	ID             int64             `json:"id"`
	SKU            string            `json:"sku"`
	Name           string            `json:"name"`
	Names          map[string]string `json:"names,omitempty"`
	Description    string            `json:"description"`
	CategoryID     *int64            `json:"category_id,omitempty"`
	Price          domain.Money      `json:"price"`
	Cost           domain.Money      `json:"cost"`
	OriginalPrice  *domain.Money     `json:"original_price,omitempty"`
	TaxRateID      *int64            `json:"tax_rate_id,omitempty"`
	TaxRateBps     int64             `json:"tax_rate_bps"`
	TaxInclusive   bool              `json:"tax_inclusive"`
	TrackStock     bool              `json:"track_stock"`
	Stock          int64             `json:"stock"`
	ReorderLevel   int64             `json:"reorder_level"`
	ImageThumbnail string            `json:"image_thumbnail,omitempty"`
	ImageMedium    string            `json:"image_medium,omitempty"`
	Badge          string            `json:"badge,omitempty"`
	IsOpenItem     bool              `json:"is_open_item"`
	Active         bool              `json:"active"`
	Favorite       bool              `json:"favorite"`
	Barcodes       []string          `json:"barcodes,omitempty"`
	VariantGroups  []VariantGroup    `json:"variant_groups,omitempty"`
	ModifierGroups []ModifierGroup   `json:"modifier_groups,omitempty"`
	StockStatus    string            `json:"stock_status,omitempty"` // computed: "", LOW_STOCK, OUT_OF_STOCK
}

type VariantGroup struct {
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
	Required bool      `json:"required"`
	Sort     int       `json:"sort_order"`
	Variants []Variant `json:"variants"`
}

type Variant struct {
	ID              int64        `json:"id"`
	Name            string       `json:"name"`
	PriceAdjustment domain.Money `json:"price_adjustment"`
	SKUSuffix       string       `json:"sku_suffix,omitempty"`
	Sort            int          `json:"sort_order"`
}

type ModifierGroup struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	MinSelect int               `json:"min_select"`
	MaxSelect int               `json:"max_select"`
	Required  bool              `json:"required"`
	Options   []ModifierOption  `json:"options"`
}

type ModifierOption struct {
	ID              int64        `json:"id"`
	Name            string       `json:"name"`
	PriceAdjustment domain.Money `json:"price_adjustment"`
	Sort            int          `json:"sort_order"`
}
