import { useMemo, useState } from 'react'
import { useI18n } from '../../i18n'
import { useSettings } from '../../app/SettingsContext'
import { useSell } from '../../app/SellContext'
import { useAuth } from '../../app/AuthContext'
import { formatMoney } from '../../utils/money'
import { computeCartTotals } from '../../utils/pricing'
import { CartItemRow } from './CartItemRow'
import { CustomerSelectorModal } from './CustomerSelectorModal'
import { HeldSalesModal } from './HeldSalesModal'
import { DiscountModal } from './DiscountModal'

export function CartPanel({ onCharge }: { onCharge: (totals: ReturnType<typeof computeCartTotals>) => void }) {
  const { t } = useI18n()
  const { currencySymbol, currencyDecimals, settings } = useSettings()
  const { hasPermission } = useAuth()
  const {
    lines,
    customer,
    setCustomer,
    orderNote,
    setOrderNote,
    orderDiscountType,
    orderDiscountAmount,
    orderDiscountPercentBps,
    setOrderDiscount,
    clearCart,
    holdCart,
  } = useSell()

  const [menuOpen, setMenuOpen] = useState(false)
  const [customerModalOpen, setCustomerModalOpen] = useState(false)
  const [heldModalOpen, setHeldModalOpen] = useState(false)
  const [discountModalOpen, setDiscountModalOpen] = useState(false)
  const [noteEditing, setNoteEditing] = useState(false)
  const [noteDraft, setNoteDraft] = useState(orderNote)

  const totals = useMemo(
    () => computeCartTotals(lines, orderDiscountType, orderDiscountAmount, orderDiscountPercentBps),
    [lines, orderDiscountType, orderDiscountAmount, orderDiscountPercentBps],
  )
  const totalsByLine = new Map(totals.lines.map((l) => [l.lineId, l]))
  const maxPercentBps = Number(settings['discount.max_percent_bps'] ?? '2000')

  return (
    <div className="cart-panel">
      <div className="cart-header">
        <div>
          <div className="cart-title">{t('cart_title')}</div>
          <div className="text-muted">{lines.length} {t('cart_items')}</div>
        </div>
        <div className="cart-menu-wrap">
          <button className="btn btn-icon btn-ghost" onClick={() => setMenuOpen((o) => !o)}>⋯</button>
          {menuOpen && (
            <div className="dropdown">
              <button className="dropdown-item" disabled={!hasPermission('sale.hold') || lines.length === 0} onClick={() => { setMenuOpen(false); holdCart(orderNote, '') }}>
                {t('cart_hold_sale')}
              </button>
              <button className="dropdown-item" onClick={() => { setMenuOpen(false); setHeldModalOpen(true) }}>
                {t('cart_view_held')}
              </button>
              <button className="dropdown-item" onClick={() => { setMenuOpen(false); setDiscountModalOpen(true) }} disabled={!hasPermission('discount.apply')}>
                Apply Discount
              </button>
              <button className="dropdown-item" disabled={lines.length === 0} onClick={() => { setMenuOpen(false); clearCart() }}>
                {t('cart_clear')}
              </button>
            </div>
          )}
        </div>
      </div>

      <button className="customer-chip" onClick={() => setCustomerModalOpen(true)}>
        {customer ? (
          <>
            <span className="avatar">{customer.name[0]?.toUpperCase()}</span>
            <span>
              <div>{customer.name}</div>
              <div className="text-muted">{customer.loyalty_points} pts</div>
            </span>
            <button
              className="btn btn-icon btn-ghost"
              onClick={(e) => {
                e.stopPropagation()
                setCustomer(null)
              }}
            >
              ✕
            </button>
          </>
        ) : (
          <span>+ {t('cart_add_customer')}</span>
        )}
      </button>

      {noteEditing ? (
        <input
          className="input order-note-input"
          autoFocus
          value={noteDraft}
          placeholder="Order note (e.g. Table 4, Takeaway)"
          onChange={(e) => setNoteDraft(e.target.value)}
          onBlur={() => {
            setOrderNote(noteDraft)
            setNoteEditing(false)
          }}
          onKeyDown={(e) => e.key === 'Enter' && e.currentTarget.blur()}
        />
      ) : (
        <button className="order-note-btn" onClick={() => setNoteEditing(true)}>
          {orderNote ? `📝 ${orderNote}` : `+ ${t('cart_add_note')}`}
        </button>
      )}

      <div className="cart-lines">
        {lines.length === 0 ? (
          <div className="empty-state">{t('cart_empty')}</div>
        ) : (
          lines.map((line) => <CartItemRow key={line.lineId} line={line} lineTotal={totalsByLine.get(line.lineId)?.lineTotal ?? 0} />)
        )}
      </div>

      <div className="cart-summary">
        <div className="summary-row">
          <span>{t('cart_subtotal')}</span>
          <span>{formatMoney(totals.subtotal, currencyDecimals, currencySymbol)}</span>
        </div>
        {totals.discountTotal > 0 && (
          <div className="summary-row text-danger">
            <span>{t('cart_discount')}</span>
            <span>-{formatMoney(totals.discountTotal, currencyDecimals, currencySymbol)}</span>
          </div>
        )}
        <div className="summary-row">
          <span>{t('cart_tax')}</span>
          <span>{formatMoney(totals.taxTotal, currencyDecimals, currencySymbol)}</span>
        </div>
        <div className="summary-row summary-total">
          <span>{t('cart_total')}</span>
          <span>{formatMoney(totals.total, currencyDecimals, currencySymbol)}</span>
        </div>
        <button className="btn btn-primary btn-lg btn-block charge-btn" disabled={lines.length === 0} onClick={() => onCharge(totals)}>
          {t('cart_charge')} {formatMoney(totals.total, currencyDecimals, currencySymbol)}
        </button>
      </div>

      {customerModalOpen && (
        <CustomerSelectorModal
          onSelect={(c) => {
            setCustomer(c)
            setCustomerModalOpen(false)
          }}
          onClose={() => setCustomerModalOpen(false)}
        />
      )}
      {heldModalOpen && <HeldSalesModal onClose={() => setHeldModalOpen(false)} />}
      {discountModalOpen && (
        <DiscountModal
          maxPercentBps={maxPercentBps}
          currentType={orderDiscountType}
          currentAmount={orderDiscountAmount}
          currentPercentBps={orderDiscountPercentBps}
          onApply={(type, amount, percentBps, approverId) => {
            setOrderDiscount(type, amount, percentBps, approverId)
            setDiscountModalOpen(false)
          }}
          onClose={() => setDiscountModalOpen(false)}
        />
      )}
    </div>
  )
}
