import { useSettings } from '../../app/SettingsContext'
import { X } from 'lucide-react'
import { useSell } from '../../app/SellContext'
import { formatMoney } from '../../utils/money'

export function HeldSalesModal({ onClose }: { onClose: () => void }) {
  const { heldSales, resumeHeld, deleteHeld } = useSell()
  const { currencySymbol, currencyDecimals } = useSettings()

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth: 560 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Held Sales</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose}><X size={18} /></button>
        </div>
        <div className="modal-body">
          {heldSales.length === 0 && <div className="empty-state">No held sales.</div>}
          <div className="stack">
            {heldSales.map((h) => (
              <div key={h.id} className="held-sale-row card">
                <div>
                  <div className="held-sale-title">
                    {h.ticket_name || h.table_name || `Ticket #${h.id}`}
                  </div>
                  <div className="text-muted">
                    {h.item_count} items · {formatMoney(h.total, currencyDecimals, currencySymbol)} · {new Date(h.created_at).toLocaleTimeString()}
                  </div>
                  {h.note && <div className="text-muted">{h.note}</div>}
                </div>
                <div className="toolbar">
                  <button
                    className="btn btn-primary"
                    onClick={async () => {
                      await resumeHeld(h.id)
                      onClose()
                    }}
                  >
                    Resume
                  </button>
                  <button className="btn btn-danger" onClick={() => deleteHeld(h.id)}>
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
