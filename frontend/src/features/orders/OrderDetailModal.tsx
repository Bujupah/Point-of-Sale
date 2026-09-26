import { useState } from 'react'
import { useSettings } from '../../app/SettingsContext'
import { useAuth } from '../../app/AuthContext'
import { api, ApiError } from '../../services/api'
import { formatMoney } from '../../utils/money'
import type { Sale } from '../../types'
import { RefundModal } from './RefundModal'

export function OrderDetailModal({ sale, onClose, onChanged }: { sale: Sale; onClose: () => void; onChanged: () => void }) {
  const { currencySymbol, currencyDecimals } = useSettings()
  const { hasPermission } = useAuth()
  const [refundOpen, setRefundOpen] = useState(false)
  const [error, setError] = useState('')

  async function reprint() {
    setError('')
    try {
      await api.post(`/api/sales/${sale.id}/print`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Printer unavailable')
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth: 480 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>{sale.receipt_number}</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose}>✕</button>
        </div>
        <div className="modal-body stack">
          <div className="summary-row"><span>Status</span><span>{sale.status}</span></div>
          <div className="summary-row"><span>Cashier</span><span>{sale.cashier_name}</span></div>
          {sale.customer_name && <div className="summary-row"><span>Customer</span><span>{sale.customer_name}</span></div>}
          <div className="summary-row"><span>Date</span><span>{new Date(sale.created_at).toLocaleString()}</span></div>

          <table className="table">
            <thead>
              <tr><th>Item</th><th>Qty</th><th>Total</th></tr>
            </thead>
            <tbody>
              {(sale.items ?? []).map((it) => (
                <tr key={it.id}>
                  <td>{it.name}{it.variant ? ` (${it.variant})` : ''}</td>
                  <td>{it.quantity}</td>
                  <td>{formatMoney(it.line_total, currencyDecimals, currencySymbol)}</td>
                </tr>
              ))}
            </tbody>
          </table>

          <div className="summary-row"><span>Subtotal</span><span>{formatMoney(sale.subtotal, currencyDecimals, currencySymbol)}</span></div>
          {sale.discount_total > 0 && <div className="summary-row"><span>Discount</span><span>-{formatMoney(sale.discount_total, currencyDecimals, currencySymbol)}</span></div>}
          <div className="summary-row"><span>Tax</span><span>{formatMoney(sale.tax_total, currencyDecimals, currencySymbol)}</span></div>
          <div className="summary-row summary-total"><span>Total</span><span>{formatMoney(sale.total, currencyDecimals, currencySymbol)}</span></div>

          {(sale.payments ?? []).map((p) => (
            <div key={p.id} className="summary-row text-muted"><span>{p.method}</span><span>{formatMoney(p.amount, currencyDecimals, currencySymbol)}</span></div>
          ))}

          {error && <div className="auth-error">{error}</div>}

          <div className="toolbar">
            <button className="btn btn-block" onClick={reprint}>Reprint</button>
            {hasPermission('refund.create') && sale.status !== 'REFUNDED' && (
              <button className="btn btn-danger btn-block" onClick={() => setRefundOpen(true)}>Refund</button>
            )}
          </div>
        </div>
      </div>
      {refundOpen && (
        <RefundModal
          sale={sale}
          onClose={() => setRefundOpen(false)}
          onDone={() => {
            setRefundOpen(false)
            onChanged()
          }}
        />
      )}
    </div>
  )
}
