// Package sales owns the sale-commit transaction — the single most
// important piece of the whole application. A sale is never visible as
// successful before its transaction commits (brief §30/§76).
package sales

import "pos/internal/domain"

type ModifierSelection struct {
	ModifierOptionID int64 `json:"modifier_option_id"`
}

type VariantSelection struct {
	VariantID int64 `json:"variant_id"`
}

// ItemInput is one cart line as sent by the client. Pricing is always
// resolved server-side from the current catalog (plus variant/modifier
// adjustments); the client never dictates unit_price except for an explicit,
// permission-gated open item or price override.
type ItemInput struct {
	ProductID          *int64              `json:"product_id,omitempty"`
	OpenItemName       string              `json:"open_item_name,omitempty"`
	OpenItemPrice      *domain.Money       `json:"open_item_price,omitempty"`
	OpenItemTaxBps     int64               `json:"open_item_tax_bps,omitempty"`
	Quantity           int64               `json:"quantity"`
	UnitPriceOverride  *domain.Money       `json:"unit_price_override,omitempty"`
	VariantSelections  []VariantSelection  `json:"variant_selections,omitempty"`
	ModifierSelections []ModifierSelection `json:"modifier_selections,omitempty"`
	DiscountAmount     domain.Money        `json:"discount_amount,omitempty"`     // used when DiscountType == FIXED
	DiscountPercentBps int64               `json:"discount_percent_bps,omitempty"` // used when DiscountType == PERCENT, basis points (500 = 5%)
	DiscountType       string              `json:"discount_type,omitempty"`       // "", FIXED, PERCENT
	Notes              string              `json:"notes,omitempty"`
}

type PaymentInput struct {
	Method    string       `json:"method"` // CASH, CARD, BANK_TRANSFER, GIFT_CARD, OTHER, MOBILE, VOUCHER, CHEQUE, CUSTOM
	Amount    domain.Money `json:"amount"`
	Tendered  domain.Money `json:"tendered,omitempty"`
	Reference string       `json:"reference,omitempty"`
}

type CheckoutInput struct {
	CustomerID          *int64         `json:"customer_id,omitempty"`
	Items               []ItemInput    `json:"items"`
	OrderDiscountAmount domain.Money   `json:"order_discount_amount,omitempty"`
	OrderDiscountPercentBps int64      `json:"order_discount_percent_bps,omitempty"`
	OrderDiscountType   string         `json:"order_discount_type,omitempty"` // "", FIXED, PERCENT
	OrderDiscountReason string         `json:"order_discount_reason,omitempty"`
	ManagerApproverID   *int64         `json:"manager_approver_id,omitempty"`
	Note                string         `json:"note,omitempty"`
	Payments            []PaymentInput `json:"payments"`
}

type SaleItemResult struct {
	ID              int64        `json:"id"`
	ProductID       *int64       `json:"product_id,omitempty"`
	Name            string       `json:"name"`
	Variant         string       `json:"variant,omitempty"`
	Modifiers       []string     `json:"modifiers,omitempty"`
	Quantity        int64        `json:"quantity"`
	UnitPrice       domain.Money `json:"unit_price"`
	DiscountAmount  domain.Money `json:"discount_amount"`
	TaxAmount       domain.Money `json:"tax_amount"`
	LineTotal       domain.Money `json:"line_total"`
	Notes           string       `json:"notes,omitempty"`
}

type PaymentResult struct {
	ID        int64        `json:"id"`
	Method    string       `json:"method"`
	Amount    domain.Money `json:"amount"`
	Tendered  domain.Money `json:"tendered"`
	ChangeDue domain.Money `json:"change_due"`
	Reference string       `json:"reference,omitempty"`
}

type Sale struct {
	ID             int64            `json:"id"`
	ReceiptNumber  string           `json:"receipt_number"`
	RegisterID     int64            `json:"register_id"`
	ShiftID        int64            `json:"shift_id"`
	CashierID      int64            `json:"cashier_id"`
	CashierName    string           `json:"cashier_name,omitempty"`
	CustomerID     *int64           `json:"customer_id,omitempty"`
	CustomerName   string           `json:"customer_name,omitempty"`
	Subtotal       domain.Money     `json:"subtotal"`
	DiscountTotal  domain.Money     `json:"discount_total"`
	TaxTotal       domain.Money     `json:"tax_total"`
	Total          domain.Money     `json:"total"`
	Status         string           `json:"status"`
	Note           string           `json:"note,omitempty"`
	CreatedAt      string           `json:"created_at"`
	Items          []SaleItemResult `json:"items,omitempty"`
	Payments       []PaymentResult  `json:"payments,omitempty"`
	LoyaltyEarned  int64            `json:"loyalty_points_earned,omitempty"`
	ChangeDue      domain.Money     `json:"change_due"`
}
