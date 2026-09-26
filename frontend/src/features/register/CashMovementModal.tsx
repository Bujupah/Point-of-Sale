import { useState } from 'react'
import { useI18n } from '../../i18n'
import { useSettings } from '../../app/SettingsContext'
import { api, ApiError } from '../../services/api'
import { parseMoneyInput } from '../../utils/money'

export function CashMovementModal({ type, onClose, onDone }: { type: 'CASH_IN' | 'CASH_OUT'; onClose: () => void; onDone: () => void }) {
  const { t } = useI18n()
  const { currencyDecimals } = useSettings()
  const [amount, setAmount] = useState('')
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit() {
    setBusy(true)
    setError('')
    try {
      await api.post('/api/cash/movements', { type, amount: parseMoneyInput(amount, currencyDecimals), reason })
      onDone()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth: 360 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>{type === 'CASH_IN' ? t('shift_cash_in') : t('shift_cash_out')}</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose}>✕</button>
        </div>
        <div className="modal-body stack">
          <label className="field">
            Amount
            <input className="input" value={amount} onChange={(e) => setAmount(e.target.value)} inputMode="decimal" autoFocus />
          </label>
          <label className="field">
            {t('common_reason')}
            <input className="input" value={reason} onChange={(e) => setReason(e.target.value)} />
          </label>
          {error && <div className="auth-error">{error}</div>}
        </div>
        <div className="modal-footer">
          <button className="btn" onClick={onClose}>{t('common_cancel')}</button>
          <button className="btn btn-primary" disabled={busy || !amount || !reason} onClick={submit}>{t('common_confirm')}</button>
        </div>
      </div>
    </div>
  )
}
