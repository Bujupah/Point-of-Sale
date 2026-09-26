import type { CartLine } from '../types'

// Builds the item payload for POST /api/sales / /api/held-sales. Price is
// never sent for a catalog product — the backend always resolves it fresh
// from the current product + variant + modifier records (brief §63/§65);
// only open items (which have no catalog row) carry an explicit price.
export function buildItemsPayload(lines: CartLine[]) {
  return lines.map((l) => ({
    product_id: l.product?.id,
    open_item_name: l.openItemName,
    open_item_price: l.openItemPrice,
    open_item_tax_bps: l.openItemTaxBps,
    quantity: l.quantity,
    variant_selections: l.variantSelections.map((v) => ({ variant_id: v.variant.id })),
    modifier_selections: l.modifierSelections.map((m) => ({ modifier_option_id: m.option.id })),
    discount_amount: l.discountType === 'FIXED' ? l.discountAmount ?? 0 : undefined,
    discount_percent_bps: l.discountType === 'PERCENT' ? l.discountPercentBps ?? 0 : undefined,
    discount_type: l.discountType,
    notes: l.notes,
  }))
}
