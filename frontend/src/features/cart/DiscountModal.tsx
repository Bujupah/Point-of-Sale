import { useState } from 'react'
import { X } from 'lucide-react'
import { useAuth } from '../../app/AuthContext'
import { useSettings } from '../../app/SettingsContext'
import { api, ApiError } from '../../services/api'

interface Props {
  onApply: (type: 'FIXED' | 'PERCENT', amount: number, percentBps: number, approverId?: number) => void
  onClose: () => void
  currentType?: 'FIXED' | 'PERCENT'
  currentAmount?: number
  currentPercentBps?: number
  maxPercentBps: number
}

export function DiscountModal({ onApply, onClose, currentType, currentAmount, currentPercentBps, maxPercentBps }: Props) {
  const { hasPermission } = useAuth()
  const { currencySymbol, currencyDecimals } = useSettings()
  const [type, setType] = useState<'FIXED' | 'PERCENT'>(currentType ?? 'PERCENT')
  const [percent, setPercent] = useState(currentPercentBps ? currentPercentBps / 100 : 10)
  const [amount, setAmount] = useState(currentAmount ? currentAmount / 10 ** currencyDecimals : 0)
  const [managerUsername, setManagerUsername] = useState('')
  const [managerPin, setManagerPin] = useState('')
  const [error, setError] = useState('')

  const percentBps = Math.round(percent * 100)
  const exceedsLimit = type === 'PERCENT' && percentBps > maxPercentBps
  const needsApproval = exceedsLimit && !hasPermission('discount.override')

  async function apply() {
    setError('')
    const amountMinor = Math.round(amount * 10 ** currencyDecimals)
    if (!exceedsLimit) {
      onApply(type, amountMinor, percentBps)
      return
    }
    try {
      // A manager PIN check reuses the login endpoint purely to verify the
      // credentials and get the approving user's id — it opens a second
      // session as a side effect, which is harmless (idle sessions expire).
      const result = await api.post<{ user: { id: number } }>('/api/auth/login', { username: managerUsername, pin: managerPin })
      onApply(type, amountMinor, percentBps, result.user.id)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Manager approval failed')
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth: 380 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Order Discount</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose}><X size={18} /></button>
        </div>
        <div className="modal-body stack">
          <div className="toolbar">
            <button className={`chip ${type === 'PERCENT' ? 'chip-active' : ''}`} onClick={() => setType('PERCENT')}>%</button>
            <button className={`chip ${type === 'FIXED' ? 'chip-active' : ''}`} onClick={() => setType('FIXED')}>{currencySymbol}</button>
          </div>
          {type === 'PERCENT' ? (
            <label className="field">
              Percent
              <input className="input" type="number" min={0} max={100} value={percent} onChange={(e) => setPercent(Number(e.target.value))} />
            </label>
          ) : (
            <label className="field">
              Amount
              <input className="input" type="number" min={0} value={amount} onChange={(e) => setAmount(Number(e.target.value))} />
            </label>
          )}
          <div className="toolbar">
            {[5, 10, 15, 20].map((p) => (
              <button key={p} className="chip" onClick={() => { setType('PERCENT'); setPercent(p) }}>
                -{p}
              </button>
            ))}
          </div>

          {needsApproval && (
            <div className="manager-approval">
              <p className="text-danger">This discount exceeds the {maxPercentBps / 100}% cashier limit. Manager approval required.</p>
              <label className="field">
                Manager username
                <input className="input" value={managerUsername} onChange={(e) => setManagerUsername(e.target.value)} />
              </label>
              <label className="field">
                Manager PIN
                <input className="input" type="password" value={managerPin} onChange={(e) => setManagerPin(e.target.value)} />
              </label>
            </div>
          )}
          {error && <div className="auth-error">{error}</div>}
        </div>
        <div className="modal-footer">
          <button className="btn" onClick={onClose}>Cancel</button>
          <button className="btn btn-primary" disabled={needsApproval && (!managerUsername || !managerPin)} onClick={apply}>
            Apply
          </button>
        </div>
      </div>
    </div>
  )
}
