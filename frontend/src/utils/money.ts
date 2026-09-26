import type { Money } from '../types'

// Formats minor units as a decimal string using the currency's own
// precision (2 or 3 decimals). Never do float math on Money — this function
// only formats for display.
export function formatMoney(amount: Money, decimals: number, symbol: string): string {
  const neg = amount < 0
  const v = Math.abs(amount)
  const div = 10 ** decimals
  const whole = Math.floor(v / div)
  const frac = v % div
  const fracStr = String(frac).padStart(decimals, '0')
  const num = decimals > 0 ? `${whole}.${fracStr}` : `${whole}`
  return `${neg ? '-' : ''}${num} ${symbol}`
}

// Suggests quick cash-tendered amounts (brief §25): the exact total, the
// next round number just above it, and the common bill denominations that
// cover it — scaled to the currency's own minor-unit precision so this
// works for 2- and 3-decimal currencies alike.
export function suggestedCashAmounts(total: Money, decimals: number): Money[] {
  if (total <= 0) return []
  const unit = 10 ** decimals
  const suggestions = new Set<number>([total])
  const roundStep = unit * 5
  const rounded = Math.ceil(total / roundStep) * roundStep
  if (rounded > total) suggestions.add(rounded)
  for (const bill of [10, 20, 50, 100]) {
    const amount = bill * unit
    if (amount >= total) suggestions.add(amount)
  }
  return Array.from(suggestions)
    .sort((a, b) => a - b)
    .slice(0, 4)
}

export function parseMoneyInput(input: string, decimals: number): Money {
  const cleaned = input.replace(/[^0-9.-]/g, '')
  const f = parseFloat(cleaned)
  if (Number.isNaN(f)) return 0
  return Math.round(f * 10 ** decimals)
}
