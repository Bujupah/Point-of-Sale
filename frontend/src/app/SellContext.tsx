import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api } from '../services/api'
import type { CartLine, CartModifierSelection, CartVariantSelection, Customer, HeldSale, Product, Sale } from '../types'
import { useAuth } from './AuthContext'
import { useShift } from './ShiftContext'

function newLineId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

interface SellContextValue {
  lines: CartLine[]
  customer: Customer | null
  orderNote: string
  orderDiscountType?: 'FIXED' | 'PERCENT'
  orderDiscountAmount: number
  orderDiscountPercentBps: number
  orderDiscountApproverId?: number
  heldSales: HeldSale[]
  heldCount: number
  reloadHeld: () => Promise<void>

  addProduct: (product: Product, variants: CartVariantSelection[], modifiers: CartModifierSelection[], notes?: string, quantity?: number) => void
  addOpenItem: (name: string, price: number, taxBps: number, notes?: string) => void
  updateQuantity: (lineId: string, quantity: number) => void
  removeLine: (lineId: string) => void
  updateLineNotes: (lineId: string, notes: string) => void
  updateLineDiscount: (lineId: string, type: 'FIXED' | 'PERCENT' | undefined, amount: number, percentBps: number) => void
  setCustomer: (customer: Customer | null) => void
  setOrderNote: (note: string) => void
  setOrderDiscount: (type: 'FIXED' | 'PERCENT' | undefined, amount: number, percentBps: number, approverId?: number) => void
  clearCart: () => void

  holdCart: (note: string, tableName: string) => Promise<void>
  resumeHeld: (id: number) => Promise<void>
  deleteHeld: (id: number) => Promise<void>

  lastCompletedSale: Sale | null
  setLastCompletedSale: (sale: Sale | null) => void
}

const SellContext = createContext<SellContextValue | null>(null)

export function SellProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const { shift } = useShift()
  const [lines, setLines] = useState<CartLine[]>([])
  const [customer, setCustomer] = useState<Customer | null>(null)
  const [orderNote, setOrderNote] = useState('')
  const [orderDiscountType, setOrderDiscountType] = useState<'FIXED' | 'PERCENT' | undefined>(undefined)
  const [orderDiscountAmount, setOrderDiscountAmount] = useState(0)
  const [orderDiscountPercentBps, setOrderDiscountPercentBps] = useState(0)
  const [orderDiscountApproverId, setOrderDiscountApproverId] = useState<number | undefined>(undefined)
  const [heldSales, setHeldSales] = useState<HeldSale[]>([])
  const [lastCompletedSale, setLastCompletedSale] = useState<Sale | null>(null)

  const reloadHeld = useCallback(async () => {
    if (!shift) {
      setHeldSales([])
      return
    }
    try {
      const list = await api.get<HeldSale[]>('/api/held-sales')
      setHeldSales(list ?? [])
    } catch {
      /* non-fatal; sidebar badge just stays at last known count */
    }
  }, [shift])

  useEffect(() => {
    if (user && shift) reloadHeld()
  }, [user, shift, reloadHeld])

  const clearCart = useCallback(() => {
    setLines([])
    setCustomer(null)
    setOrderNote('')
    setOrderDiscountType(undefined)
    setOrderDiscountAmount(0)
    setOrderDiscountPercentBps(0)
    setOrderDiscountApproverId(undefined)
  }, [])

  const addProduct = useCallback<SellContextValue['addProduct']>((product, variants, modifiers, notes, quantity = 1) => {
    setLines((prev) => {
      const variantKey = variants.map((v) => v.variant.id).sort().join(',')
      const modifierKey = modifiers.map((m) => m.option.id).sort().join(',')
      const existing = prev.find(
        (l) =>
          l.product?.id === product.id &&
          !l.notes &&
          !notes &&
          l.variantSelections.map((v) => v.variant.id).sort().join(',') === variantKey &&
          l.modifierSelections.map((m) => m.option.id).sort().join(',') === modifierKey,
      )
      if (existing) {
        return prev.map((l) => (l.lineId === existing.lineId ? { ...l, quantity: l.quantity + quantity } : l))
      }
      const line: CartLine = {
        lineId: newLineId(),
        product,
        quantity,
        variantSelections: variants,
        modifierSelections: modifiers,
        notes,
      }
      return [...prev, line]
    })
  }, [])

  const addOpenItem = useCallback<SellContextValue['addOpenItem']>((name, price, taxBps, notes) => {
    setLines((prev) => [
      ...prev,
      { lineId: newLineId(), openItemName: name, openItemPrice: price, openItemTaxBps: taxBps, quantity: 1, variantSelections: [], modifierSelections: [], notes },
    ])
  }, [])

  const updateQuantity = useCallback((lineId: string, quantity: number) => {
    setLines((prev) => (quantity <= 0 ? prev.filter((l) => l.lineId !== lineId) : prev.map((l) => (l.lineId === lineId ? { ...l, quantity } : l))))
  }, [])

  const removeLine = useCallback((lineId: string) => {
    setLines((prev) => prev.filter((l) => l.lineId !== lineId))
  }, [])

  const updateLineNotes = useCallback((lineId: string, notes: string) => {
    setLines((prev) => prev.map((l) => (l.lineId === lineId ? { ...l, notes } : l)))
  }, [])

  const updateLineDiscount = useCallback((lineId: string, type: 'FIXED' | 'PERCENT' | undefined, amount: number, percentBps: number) => {
    setLines((prev) => prev.map((l) => (l.lineId === lineId ? { ...l, discountType: type, discountAmount: amount, discountPercentBps: percentBps } : l)))
  }, [])

  const setOrderDiscount = useCallback((type: 'FIXED' | 'PERCENT' | undefined, amount: number, percentBps: number, approverId?: number) => {
    setOrderDiscountType(type)
    setOrderDiscountAmount(amount)
    setOrderDiscountPercentBps(percentBps)
    setOrderDiscountApproverId(approverId)
  }, [])

  const holdCart = useCallback(
    async (note: string, tableName: string) => {
      const items = lines.map((l) => ({
        product_id: l.product?.id,
        open_item_name: l.openItemName,
        open_item_price: l.openItemPrice,
        open_item_tax_bps: l.openItemTaxBps,
        quantity: l.quantity,
        variant_selections: l.variantSelections.map((v) => ({ variant_id: v.variant.id })),
        modifier_selections: l.modifierSelections.map((m) => ({ modifier_option_id: m.option.id })),
        notes: l.notes,
      }))
      await api.post('/api/held-sales', { items, note, table_name: tableName, customer_id: customer?.id })
      clearCart()
      await reloadHeld()
    },
    [lines, customer, clearCart, reloadHeld],
  )

  // Resuming a held sale re-adds each line through the normal
  // addProduct/addOpenItem path rather than trusting the held snapshot's
  // price: held_sale_items only stores variant/modifier NAMES (not ids, see
  // migrations/004), so an exact re-selection isn't possible. Re-resolving
  // against the live product keeps checkout on the normal, permission-safe
  // path (no price-override needed) at the cost of picking up any price
  // change since the sale was parked — acceptable for a "held a few minutes
  // ago" ticket, and the original variant/modifier text is preserved in the
  // line's notes so nothing is silently dropped.
  const resumeHeld = useCallback(
    async (id: number) => {
      const held = await api.post<HeldSale>(`/api/held-sales/${id}/resume`)
      for (const item of held.items ?? []) {
        const combinedNotes = [item.variant, item.notes].filter(Boolean).join(' — ')
        if (item.product_id) {
          try {
            const product = await api.get<Product>(`/api/products/${item.product_id}`)
            addProduct(product, [], [], combinedNotes || undefined, item.quantity)
          } catch {
            addOpenItem(item.name, item.unit_price, 0, combinedNotes || undefined)
          }
        } else {
          addOpenItem(item.name, item.unit_price, 0, combinedNotes || undefined)
        }
      }
      if (held.customer_id) {
        try {
          setCustomer(await api.get<Customer>(`/api/customers/${held.customer_id}`))
        } catch {
          /* customer may have been removed; cart still resumes */
        }
      }
      if (held.note) setOrderNote(held.note)
      await reloadHeld()
    },
    [addProduct, addOpenItem, reloadHeld],
  )

  const deleteHeld = useCallback(
    async (id: number) => {
      await api.del(`/api/held-sales/${id}`)
      await reloadHeld()
    },
    [reloadHeld],
  )

  const value = useMemo<SellContextValue>(
    () => ({
      lines,
      customer,
      orderNote,
      orderDiscountType,
      orderDiscountAmount,
      orderDiscountPercentBps,
      orderDiscountApproverId,
      heldSales,
      heldCount: heldSales.length,
      reloadHeld,
      addProduct,
      addOpenItem,
      updateQuantity,
      removeLine,
      updateLineNotes,
      updateLineDiscount,
      setCustomer,
      setOrderNote,
      setOrderDiscount,
      clearCart,
      holdCart,
      resumeHeld,
      deleteHeld,
      lastCompletedSale,
      setLastCompletedSale,
    }),
    [
      lines,
      customer,
      orderNote,
      orderDiscountType,
      orderDiscountAmount,
      orderDiscountPercentBps,
      orderDiscountApproverId,
      heldSales,
      reloadHeld,
      addProduct,
      addOpenItem,
      updateQuantity,
      removeLine,
      updateLineNotes,
      updateLineDiscount,
      setOrderDiscount,
      clearCart,
      holdCart,
      resumeHeld,
      deleteHeld,
      lastCompletedSale,
    ],
  )

  return <SellContext.Provider value={value}>{children}</SellContext.Provider>
}

export function useSell() {
  const ctx = useContext(SellContext)
  if (!ctx) throw new Error('useSell must be used within SellProvider')
  return ctx
}
