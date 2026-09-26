import { useState } from 'react'
import { useI18n } from '../../i18n'
import { useSettings } from '../../app/SettingsContext'
import { useShift } from '../../app/ShiftContext'
import { parseMoneyInput } from '../../utils/money'
import { ApiError } from '../../services/api'

export function OpenShiftPrompt() {
  const { t } = useI18n()
  const { currencySymbol, currencyDecimals } = useSettings()
  const { openShift } = useShift()
  const [amount, setAmount] = useState('0')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit() {
    setBusy(true)
    setError('')
    try {
      await openShift(parseMoneyInput(amount, currencyDecimals))
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not open shift')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="empty-state" style={{ width: '100%' }}>
      <div className="card open-shift-card">
        <h2>{t('shift_open')}</h2>
        <p className="text-muted">Enter the opening cash float ({currencySymbol}) to start selling on this register.</p>
        <label className="field">
          {t('shift_opening_float')}
          <input className="input" value={amount} onChange={(e) => setAmount(e.target.value)} inputMode="decimal" autoFocus />
        </label>
        {error && <div className="auth-error">{error}</div>}
        <button className="btn btn-primary btn-lg btn-block" disabled={busy} onClick={submit}>
          {t('shift_open')}
        </button>
      </div>
    </div>
  )
}
