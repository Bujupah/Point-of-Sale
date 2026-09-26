// Money is always an integer count of minor currency units, mirroring
// internal/domain.Money on the backend. Never treat it as a float.
export type Money = number

export interface User {
  id: number
  name: string
  username: string
  role_id: number
  role_name: string
  active: boolean
}

export interface AuthResult {
  token: string
  user: User
  permissions: string[]
  locked: boolean
  register_id?: number
}

export interface Category {
  id: number
  parent_id?: number
  name: string
  names?: Record<string, string>
  sort_order: number
  active: boolean
}

export interface Variant {
  id: number
  name: string
  price_adjustment: Money
  sku_suffix?: string
  sort_order: number
}

export interface VariantGroup {
  id: number
  name: string
  required: boolean
  sort_order: number
  variants: Variant[]
}

export interface ModifierOption {
  id: number
  name: string
  price_adjustment: Money
  sort_order: number
}

export interface ModifierGroup {
  id: number
  name: string
  min_select: number
  max_select: number
  required: boolean
  options: ModifierOption[]
}

export interface Product {
  id: number
  sku: string
  name: string
  names?: Record<string, string>
  description: string
  category_id?: number
  price: Money
  cost: Money
  original_price?: Money
  tax_rate_id?: number
  tax_rate_bps: number
  tax_inclusive: boolean
  track_stock: boolean
  stock: number
  reorder_level: number
  image_thumbnail?: string
  image_medium?: string
  badge?: string
  is_open_item: boolean
  active: boolean
  favorite: boolean
  barcodes?: string[]
  variant_groups?: VariantGroup[]
  modifier_groups?: ModifierGroup[]
  stock_status?: '' | 'LOW_STOCK' | 'OUT_OF_STOCK'
}

export interface Customer {
  id: number
  customer_number: string
  name: string
  phone: string
  email: string
  notes: string
  loyalty_points: number
  visit_count: number
  lifetime_spend: number
  last_visit_at?: string
  created_at: string
}

export interface CartModifierSelection {
  group: ModifierGroup
  option: ModifierOption
}

export interface CartVariantSelection {
  group: VariantGroup
  variant: Variant
}

export interface CartLine {
  lineId: string // client-generated, for React keys and editing
  product?: Product
  openItemName?: string
  openItemPrice?: Money
  openItemTaxBps?: number
  quantity: number
  unitPriceOverride?: Money
  variantSelections: CartVariantSelection[]
  modifierSelections: CartModifierSelection[]
  discountAmount?: Money
  discountPercentBps?: number
  discountType?: 'FIXED' | 'PERCENT'
  notes?: string
}

export interface SaleItemResult {
  id: number
  product_id?: number
  name: string
  variant?: string
  modifiers?: string[]
  quantity: number
  unit_price: Money
  discount_amount: Money
  tax_amount: Money
  line_total: Money
  notes?: string
}

export interface PaymentResult {
  id: number
  method: string
  amount: Money
  tendered: Money
  change_due: Money
  reference?: string
}

export interface Sale {
  id: number
  receipt_number: string
  register_id: number
  shift_id: number
  cashier_id: number
  cashier_name?: string
  customer_id?: number
  customer_name?: string
  subtotal: Money
  discount_total: Money
  tax_total: Money
  total: Money
  status: string
  note?: string
  created_at: string
  items?: SaleItemResult[]
  payments?: PaymentResult[]
  loyalty_points_earned?: number
  change_due: Money
}

export interface HeldSale {
  id: number
  register_id: number
  cashier_id: number
  customer_id?: number
  note: string
  table_name: string
  ticket_name: string
  subtotal: Money
  total: Money
  item_count: number
  created_at: string
  items?: SaleItemResult[]
}

export interface Shift {
  id: number
  register_id: number
  register_name?: string
  cashier_id: number
  cashier_name?: string
  opening_float: Money
  opening_at: string
  closing_at?: string
  status: 'OPEN' | 'CLOSING' | 'CLOSED'
  expected_cash?: Money
  counted_cash?: Money
  difference?: Money
  notes: string
}

export interface ZReport {
  shift: Shift
  gross_sales: Money
  net_sales: Money
  discount_total: Money
  tax_total: Money
  refunds_total: Money
  payments: { method: string; amount: Money; count: number }[]
  opening_float: Money
  cash_in: Money
  cash_out: Money
  cash_drop: Money
  cash_sales: Money
  cash_refunds: Money
  expected_cash: Money
  counted_cash?: Money
  difference?: Money
  order_count: number
  average_ticket: Money
}

export interface ApiError {
  error: { code: string; message: string }
}
