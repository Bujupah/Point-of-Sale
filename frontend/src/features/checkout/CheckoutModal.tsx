import { useMemo, useState } from 'react'
import { X, Banknote, CreditCard, Landmark, Gift, MoreHorizontal, CheckCircle2 } from 'lucide-react'
import { useI18n } from '../../i18n'
import { useSettings } from '../../app/SettingsContext'
import { useSell } from '../../app/SellContext'
import { api, ApiError } from '../../services/api'
import { formatMoney, parseMoneyInput, suggestedCashAmounts } from '../../utils/money'
import { computeCartTotals } from '../../utils/pricing'
import { buildItemsPayload } from '../../utils/checkoutPayload'
import { NumericKeypad } from '../../components/NumericKeypad'
import type { Sale } from '../../types'

interface PendingPayment {
  method: string
  amount: number
  tendered?: number
  reference?: string
}

type MethodTab = 'CASH' | 'CARD' | 'BANK_TRANSFER' | 'GIFT_CARD' | 'OTHER' | null

export function CheckoutModal({
  totals,
  onClose,
  onComplete,
}: {
  totals: ReturnType<typeof computeCartTotals>
  onClose: () => void
  onComplete: (sale: Sale, printError?: string) => void
}) {
  const { t } = useI18n()
  const { currencySymbol, currencyDecimals } = useSettings()
  const { lines, customer, orderNote, orderDiscountType, orderDiscountAmount, orderDiscountPercentBps, orderDiscountApproverId, clearCart } = useSell()

  const [payments, setPayments] = useState<PendingPayment[]>([])
  const [activeTab, setActiveTab] = useState<MethodTab>(null)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const paid = payments.reduce((s, p) => s + p.amount, 0)
  const remaining = Math.max(0, totals.total - paid)
  const overpaidChange = payments.reduce((s, p) => s + Math.max(0, (p.tendered ?? p.amount) - p.amount), 0)

  function addPayment(p: PendingPayment) {
    setPayments((prev) => [...prev, p])
    setActiveTab(null)
  }
  function removePayment(idx: number) {
    setPayments((prev) => prev.filter((_, i) => i !== idx))
  }

  async function complete() {
    setSubmitting(true)
    setError('')
    try {
      const res = await api.post<{ sale: Sale; print_error?: string }>('/api/sales', {
        customer_id: customer?.id,
        items: buildItemsPayload(lines),
        order_discount_amount: orderDiscountType === 'FIXED' ? orderDiscountAmount : undefined,
        order_discount_percent_bps: orderDiscountType === 'PERCENT' ? orderDiscountPercentBps : undefined,
        order_discount_type: orderDiscountType,
        manager_approver_id: orderDiscountApproverId,
        note: orderNote,
        payments: payments.map((p) => ({ method: p.method, amount: p.amount, tendered: p.tendered, reference: p.reference })),
      })
      clearCart()
      onComplete(res.sale, res.print_error)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Checkout failed')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="modal-overlay">
      <div className="modal checkout-modal">
        <div className="modal-header">
          <h2>{t('payment_amount_due')}</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose} disabled={submitting}><X size={18} /></button>
        </div>
        <div className="modal-body checkout-body">
          <div className="checkout-amount-due">{formatMoney(remaining, currencyDecimals, currencySymbol)}</div>
          {customer && <div className="text-muted">for {customer.name}</div>}

          {payments.length > 0 && (
            <div className="payment-list">
              {payments.map((p, i) => (
                <div key={i} className="payment-list-row">
                  <span>{methodLabel(p.method, t)}</span>
                  <span>{formatMoney(p.amount, currencyDecimals, currencySymbol)}</span>
                  <button className="btn btn-icon btn-ghost" onClick={() => removePayment(i)}><X size={18} /></button>
                </div>
              ))}
              <div className="payment-list-row text-muted">
                <span>{t('payment_remaining')}</span>
                <span>{formatMoney(remaining, currencyDecimals, currencySymbol)}</span>
                <span />
              </div>
            </div>
          )}

          {remaining <= 0 ? (
            <div className="checkout-ready">
              {overpaidChange > 0 && (
                <div className="change-due-banner">
                  {t('payment_change')}: {formatMoney(overpaidChange, currencyDecimals, currencySymbol)}
                </div>
              )}
              <button className="btn btn-primary btn-lg btn-block" disabled={submitting} onClick={complete}>
                {submitting ? t('common_loading') : <><CheckCircle2 size={20} /> {t('payment_complete')}</>}
              </button>
            </div>
          ) : activeTab === null ? (
            <div className="method-grid">
              <button className="btn btn-lg method-btn" onClick={() => setActiveTab('CASH')}><Banknote size={20} /> {t('payment_cash')}</button>
              <button className="btn btn-lg method-btn" onClick={() => setActiveTab('CARD')}><CreditCard size={20} /> {t('payment_card')}</button>
              <button className="btn btn-lg method-btn" onClick={() => setActiveTab('BANK_TRANSFER')}><Landmark size={20} /> {t('payment_transfer')}</button>
              <button className="btn btn-lg method-btn" onClick={() => setActiveTab('GIFT_CARD')}><Gift size={20} /> {t('payment_gift_card')}</button>
              <button className="btn btn-lg method-btn" onClick={() => setActiveTab('OTHER')}><MoreHorizontal size={20} /> {t('payment_other')}</button>
            </div>
          ) : (
            <MethodPanel
              method={activeTab}
              remaining={remaining}
              decimals={currencyDecimals}
              symbol={currencySymbol}
              onCancel={() => setActiveTab(null)}
              onConfirm={addPayment}
            />
          )}

          {error && <div className="auth-error">{error}</div>}
        </div>
      </div>
    </div>
  )
}

function methodLabel(method: string, t: (k: any) => string) {
  switch (method) {
    case 'CASH':
      return t('payment_cash')
    case 'CARD':
      return t('payment_card')
    case 'BANK_TRANSFER':
      return t('payment_transfer')
    case 'GIFT_CARD':
      return t('payment_gift_card')
    default:
      return t('payment_other')
  }
}

function MethodPanel({
  method,
  remaining,
  decimals,
  symbol,
  onCancel,
  onConfirm,
}: {
  method: Exclude<MethodTab, null>
  remaining: number
  decimals: number
  symbol: string
  onCancel: () => void
  onConfirm: (p: PendingPayment) => void
}) {
  if (method === 'CASH') return <CashPanel remaining={remaining} decimals={decimals} symbol={symbol} onCancel={onCancel} onConfirm={onConfirm} />
  if (method === 'CARD') return <CardPanel remaining={remaining} decimals={decimals} symbol={symbol} onCancel={onCancel} onConfirm={onConfirm} />
  if (method === 'GIFT_CARD') return <GiftCardPanel remaining={remaining} decimals={decimals} symbol={symbol} onCancel={onCancel} onConfirm={onConfirm} />
  return <ReferencePanel method={method} remaining={remaining} decimals={decimals} symbol={symbol} onCancel={onCancel} onConfirm={onConfirm} />
}

function CashPanel({ remaining, decimals, symbol, onCancel, onConfirm }: { remaining: number; decimals: number; symbol: string; onCancel: () => void; onConfirm: (p: PendingPayment) => void }) {
  const { t } = useI18n()
  const [input, setInput] = useState(() => (remaining / 10 ** decimals).toFixed(decimals))
  const tendered = parseMoneyInput(input, decimals)
  const amount = Math.min(tendered, remaining)
  const change = Math.max(0, tendered - remaining)
  const insufficient = tendered < remaining

  const suggestions = useMemo(() => suggestedCashAmounts(remaining, decimals), [remaining, decimals])

  return (
    <div className="stack">
      <div className="toolbar">
        {suggestions.map((s) => (
          <button key={s} className="chip" onClick={() => setInput((s / 10 ** decimals).toFixed(decimals))}>
            {s === remaining ? t('payment_exact') : formatMoney(s, decimals, symbol)}
          </button>
        ))}
      </div>
      <input className="input cash-amount-input" value={input} readOnly />
      <NumericKeypad
        onDigit={(d) => setInput((v) => (v === '0' ? d : v.length < 10 ? v + d : v))}
        onBackspace={() => setInput((v) => (v.length > 1 ? v.slice(0, -1) : '0'))}
      />
      <div className="cash-summary">
        <div className="summary-row"><span>{t('payment_received')}</span><span>{formatMoney(tendered, decimals, symbol)}</span></div>
        <div className="summary-row summary-total">
          <span>{insufficient ? t('payment_remaining') : t('payment_change')}</span>
          <span>{insufficient ? formatMoney(remaining - tendered, decimals, symbol) : formatMoney(change, decimals, symbol)}</span>
        </div>
      </div>
      <div className="toolbar">
        <button className="btn btn-block" onClick={onCancel}>{t('common_cancel')}</button>
        <button className="btn btn-primary btn-block" disabled={tendered <= 0} onClick={() => onConfirm({ method: 'CASH', amount, tendered })}>
          {t('common_confirm')}
        </button>
      </div>
    </div>
  )
}

// CardPanel does not talk to a payment terminal — there is no terminal SDK
// integration in this build (brief's own "future hardware integrations must
// implement an adapter" note). Rather than fake that connection with a
// timed spinner and a scripted "Approved" outcome, this panel reflects what
// actually happens with a standalone/unintegrated terminal: the cashier
// runs the transaction on the physical device and reports its real result
// here. Nothing about the outcome is invented by the software.
function CardPanel({ remaining, decimals, symbol, onCancel, onConfirm }: { remaining: number; decimals: number; symbol: string; onCancel: () => void; onConfirm: (p: PendingPayment) => void }) {
  const { t } = useI18n()
  const [reference, setReference] = useState('')
  const [declined, setDeclined] = useState(false)

  return (
    <div className="stack terminal-state">
      <CreditCard size={32} className="text-muted" />
      <div>Charge {formatMoney(remaining, decimals, symbol)} on the card terminal, then record the result below.</div>
      <label className="field">
        Terminal reference / auth code (optional)
        <input className="input" value={reference} onChange={(e) => setReference(e.target.value)} placeholder="From the terminal receipt" autoFocus />
      </label>
      {declined && <div className="auth-error">Recorded as declined. Try again on the terminal, or choose a different payment method.</div>}
      <div className="toolbar">
        <button className="btn btn-block" onClick={onCancel}>{t('common_cancel')}</button>
        <button className="btn btn-danger btn-block" onClick={() => setDeclined(true)}>Declined</button>
        <button className="btn btn-primary btn-block" onClick={() => onConfirm({ method: 'CARD', amount: remaining, reference })}>
          Approved
        </button>
      </div>
    </div>
  )
}

function GiftCardPanel({ remaining, decimals, symbol, onCancel, onConfirm }: { remaining: number; decimals: number; symbol: string; onCancel: () => void; onConfirm: (p: PendingPayment) => void }) {
  const { t } = useI18n()
  const [code, setCode] = useState('')
  const [balance, setBalance] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [checking, setChecking] = useState(false)

  async function check() {
    setChecking(true)
    setError('')
    try {
      const gc = await api.get<{ balance: number; status: string }>(`/api/gift-cards/${encodeURIComponent(code)}`)
      if (gc.status !== 'ACTIVE') {
        setError('Gift card is not active')
        setBalance(null)
      } else {
        setBalance(gc.balance)
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Gift card not found')
      setBalance(null)
    } finally {
      setChecking(false)
    }
  }

  const amount = balance != null ? Math.min(balance, remaining) : 0

  return (
    <div className="stack">
      <label className="field">
        Gift card code
        <input className="input" value={code} onChange={(e) => setCode(e.target.value.toUpperCase())} autoFocus />
      </label>
      <button className="btn btn-block" disabled={!code || checking} onClick={check}>
        {checking ? t('common_loading') : 'Check Balance'}
      </button>
      {balance != null && <div className="text-success">Balance: {formatMoney(balance, decimals, symbol)}</div>}
      {error && <div className="auth-error">{error}</div>}
      <div className="toolbar">
        <button className="btn btn-block" onClick={onCancel}>{t('common_cancel')}</button>
        <button className="btn btn-primary btn-block" disabled={balance == null || amount <= 0} onClick={() => onConfirm({ method: 'GIFT_CARD', amount, reference: code })}>
          {t('common_confirm')}
        </button>
      </div>
    </div>
  )
}

function ReferencePanel({ method, remaining, decimals, symbol, onCancel, onConfirm }: { method: string; remaining: number; decimals: number; symbol: string; onCancel: () => void; onConfirm: (p: PendingPayment) => void }) {
  const { t } = useI18n()
  const [reference, setReference] = useState('')
  const [amountInput, setAmountInput] = useState((remaining / 10 ** decimals).toFixed(decimals))
  const amount = Math.min(parseMoneyInput(amountInput, decimals), remaining)

  return (
    <div className="stack">
      <label className="field">
        Amount ({symbol})
        <input className="input" value={amountInput} onChange={(e) => setAmountInput(e.target.value)} />
      </label>
      <label className="field">
        Reference / authorization code
        <input className="input" value={reference} onChange={(e) => setReference(e.target.value)} />
      </label>
      <div className="toolbar">
        <button className="btn btn-block" onClick={onCancel}>{t('common_cancel')}</button>
        <button className="btn btn-primary btn-block" disabled={amount <= 0} onClick={() => onConfirm({ method, amount, reference })}>
          {t('common_confirm')}
        </button>
      </div>
    </div>
  )
}
