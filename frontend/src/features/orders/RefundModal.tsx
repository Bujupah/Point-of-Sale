import { useState } from 'react'
import { X } from 'lucide-react'
import { useSettings } from '../../app/SettingsContext'
import { api, ApiError } from '../../services/api'
import { formatMoney } from '../../utils/money'
import type { Sale } from '../../types'

export function RefundModal({ sale, onClose, onDone }: { sale: Sale; onClose: () => void; onDone: () => void }) {
  const { currencySymbol, currencyDecimals } = useSettings()
  const [selected, setSelected] = useState<Record<number, number>>({}) // sale_item_id -> quantity
  const [reason, setReason] = useState('')
  const [method, setMethod] = useState<'CASH' | 'CARD' | 'OTHER'>('CASH')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const items = sale.items ?? []

  function total() {
    let sum = 0
    for (const it of items) {
      const qty = selected[it.id] ?? 0
      if (qty > 0) sum += Math.round((it.line_total * qty) / it.quantity)
    }
    return sum
  }

  async function submit() {
    setBusy(true)
    setError('')
    try {
      const refundItems = items.filter((it) => (selected[it.id] ?? 0) > 0).map((it) => ({ sale_item_id: it.id, quantity: selected[it.id] }))
      if (refundItems.length === 0) {
        setError('Select at least one item to refund')
        setBusy(false)
        return
      }
      await api.post(`/api/sales/${sale.id}/refund`, {
        items: refundItems,
        reason,
        payments: [{ method, amount: total() }],
      })
      onDone()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Refund failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth: 460 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Refund {sale.receipt_number}</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose}><X size={18} /></button>
        </div>
        <div className="modal-body stack">
          {items.map((it) => (
            <div key={it.id} className="summary-row">
              <span>{it.name} ({it.quantity} × {formatMoney(it.unit_price, currencyDecimals, currencySymbol)})</span>
              <input
                className="input"
                style={{ width: 70 }}
                type="number"
                min={0}
                max={it.quantity}
                value={selected[it.id] ?? 0}
                onChange={(e) => setSelected((s) => ({ ...s, [it.id]: Math.min(it.quantity, Math.max(0, Number(e.target.value))) }))}
              />
            </div>
          ))}
          <label className="field">
            Reason
            <input className="input" value={reason} onChange={(e) => setReason(e.target.value)} />
          </label>
          <label className="field">
            Refund method
            <select className="input" value={method} onChange={(e) => setMethod(e.target.value as any)}>
              <option value="CASH">Cash</option>
              <option value="CARD">Card</option>
              <option value="OTHER">Other</option>
            </select>
          </label>
          <div className="summary-row summary-total"><span>Refund total</span><span>{formatMoney(total(), currencyDecimals, currencySymbol)}</span></div>
          {error && <div className="auth-error">{error}</div>}
        </div>
        <div className="modal-footer">
          <button className="btn" onClick={onClose}>Cancel</button>
          <button className="btn btn-danger" disabled={busy || total() <= 0} onClick={submit}>Confirm Refund</button>
        </div>
      </div>
    </div>
  )
}
