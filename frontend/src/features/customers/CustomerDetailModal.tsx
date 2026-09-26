import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { useSettings } from '../../app/SettingsContext'
import { api } from '../../services/api'
import { formatMoney } from '../../utils/money'
import type { Customer } from '../../types'
import { CustomerEditModal } from './CustomerEditModal'

interface SaleSummary {
  id: number
  receipt_number: string
  total: number
  status: string
  created_at: string
}

export function CustomerDetailModal({ customer, onClose, onChanged }: { customer: Customer; onClose: () => void; onChanged: () => void }) {
  const { currencySymbol, currencyDecimals } = useSettings()
  const [history, setHistory] = useState<SaleSummary[]>([])
  const [editing, setEditing] = useState(false)
  const [current, setCurrent] = useState(customer)

  useEffect(() => {
    api.get<SaleSummary[]>(`/api/customers/${customer.id}/history`).then((h) => setHistory(h ?? [])).catch(() => {})
  }, [customer.id])

  if (editing) {
    return (
      <CustomerEditModal
        customer={current}
        onClose={() => setEditing(false)}
        onSaved={(c) => {
          setCurrent(c)
          setEditing(false)
          onChanged()
        }}
      />
    )
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth: 460 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>{current.name}</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose}><X size={18} /></button>
        </div>
        <div className="modal-body stack">
          <div className="summary-row"><span>Phone</span><span>{current.phone || '—'}</span></div>
          <div className="summary-row"><span>Email</span><span>{current.email || '—'}</span></div>
          <div className="summary-row"><span>Loyalty points</span><span>{current.loyalty_points}</span></div>
          <div className="summary-row"><span>Lifetime spend</span><span>{formatMoney(current.lifetime_spend, currencyDecimals, currencySymbol)}</span></div>
          <div className="summary-row"><span>Visits</span><span>{current.visit_count}</span></div>

          <h3>Recent orders</h3>
          {history.length === 0 && <div className="text-muted">No orders yet.</div>}
          {history.map((s) => (
            <div key={s.id} className="summary-row">
              <span>{s.receipt_number}</span>
              <span>{formatMoney(s.total, currencyDecimals, currencySymbol)}</span>
            </div>
          ))}
        </div>
        <div className="modal-footer">
          <button className="btn btn-primary" onClick={() => setEditing(true)}>Edit</button>
        </div>
      </div>
    </div>
  )
}
