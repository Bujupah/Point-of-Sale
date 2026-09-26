// Client-side mirror of internal/sales/calc.go, used only to show the
// cashier a live running total and to size the payment request. The backend
// recomputes everything authoritatively from the current catalog inside the
// checkout transaction (brief §63/§65) — this is a preview, never the
// source of truth, and a mismatch (e.g. a price changed mid-sale) simply
// surfaces as a clear error from POST /api/sales for the cashier to retry.
import type { CartLine, Money } from '../types'

export interface LineTotals {
  lineId: string
  unitPrice: Money
  lineSubtotal: Money
  lineDiscount: Money
  netBase: Money
  taxAmount: Money
  lineTotal: Money
}

export interface CartTotals {
  lines: LineTotals[]
  subtotal: Money
  discountTotal: Money
  taxTotal: Money
  total: Money
}

// Mirrors domain.Money.MulPercentBps from the Go backend. That Go version
// adds half the denominator before an INTEGER division to round half-up
// without floating point; naively porting that "+ den/2" trick to
// JavaScript (which divides in floating point) rounds every exact-zero
// result up by one unit (0 + 5000) / 10000 = 0.5 -> Math.round -> 1. JS's
// Math.round already rounds half-up on its own, so the fix is to divide
// directly with no manual half-unit addition.
function mulPercentBps(amount: Money, bps: number): Money {
  return Math.round((amount * bps) / 10000)
}

export function unitPriceOf(line: CartLine): Money {
  if (line.product) {
    let price = line.unitPriceOverride ?? line.product.price
    for (const v of line.variantSelections) price += v.variant.price_adjustment
    for (const m of line.modifierSelections) price += m.option.price_adjustment
    return price
  }
  return line.openItemPrice ?? 0
}

export function computeCartTotals(
  lines: CartLine[],
  orderDiscountType: 'FIXED' | 'PERCENT' | undefined,
  orderDiscountAmount: Money,
  orderDiscountPercentBps: number,
): CartTotals {
  const resolved = lines.map((line) => {
    const unitPrice = unitPriceOf(line)
    const lineSubtotal = unitPrice * line.quantity
    let lineDiscount = 0
    if (line.discountType === 'FIXED') lineDiscount = line.discountAmount ?? 0
    else if (line.discountType === 'PERCENT') lineDiscount = mulPercentBps(lineSubtotal, line.discountPercentBps ?? 0)
    lineDiscount = Math.min(Math.max(lineDiscount, 0), lineSubtotal)
    const netBase = lineSubtotal - lineDiscount
    const taxBps = line.product?.tax_rate_bps ?? line.openItemTaxBps ?? 0
    const taxInclusive = line.product?.tax_inclusive ?? false
    return { lineId: line.lineId, unitPrice, lineSubtotal, lineDiscount, netBase, taxBps, taxInclusive }
  })

  const subtotal = resolved.reduce((s, l) => s + l.lineSubtotal, 0)
  const lineDiscountSum = resolved.reduce((s, l) => s + l.lineDiscount, 0)
  const netBaseSum = resolved.reduce((s, l) => s + l.netBase, 0)

  let orderDiscount = 0
  if (orderDiscountType === 'FIXED') orderDiscount = orderDiscountAmount
  else if (orderDiscountType === 'PERCENT') orderDiscount = mulPercentBps(netBaseSum, orderDiscountPercentBps)
  orderDiscount = Math.min(Math.max(orderDiscount, 0), netBaseSum)

  let allocated = 0
  const out: LineTotals[] = resolved.map((l, i) => {
    let cut = 0
    if (netBaseSum > 0) {
      if (i === resolved.length - 1) cut = orderDiscount - allocated
      else {
        cut = Math.floor((orderDiscount * l.netBase) / netBaseSum)
        allocated += cut
      }
    }
    const adjustedBase = l.netBase - cut
    let taxAmount: number
    let lineTotal: number
    if (l.taxInclusive && l.taxBps > 0) {
      taxAmount = adjustedBase - Math.floor((adjustedBase * 10000) / (10000 + l.taxBps))
      lineTotal = adjustedBase
    } else {
      taxAmount = mulPercentBps(adjustedBase, l.taxBps)
      lineTotal = adjustedBase + taxAmount
    }
    return { lineId: l.lineId, unitPrice: l.unitPrice, lineSubtotal: l.lineSubtotal, lineDiscount: l.lineDiscount, netBase: l.netBase, taxAmount, lineTotal }
  })

  return {
    lines: out,
    subtotal,
    discountTotal: lineDiscountSum + orderDiscount,
    taxTotal: out.reduce((s, l) => s + l.taxAmount, 0),
    total: out.reduce((s, l) => s + l.lineTotal, 0),
  }
}
