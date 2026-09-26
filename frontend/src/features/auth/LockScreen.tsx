import { useState } from 'react'
import { useI18n } from '../../i18n'
import { useAuth } from '../../app/AuthContext'
import { ApiError } from '../../services/api'
import { NumericKeypad } from '../../components/NumericKeypad'
import { useShift } from '../../app/ShiftContext'

export function LockScreen() {
  const { t } = useI18n()
  const { user, unlock, switchCashier, logout } = useAuth()
  const { shift } = useShift()
  const [pin, setPin] = useState('')
  const [switching, setSwitching] = useState(false)
  const [switchUsername, setSwitchUsername] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit() {
    setBusy(true)
    setError('')
    try {
      if (switching) await switchCashier(switchUsername, pin)
      else await unlock(pin)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed')
      setPin('')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-screen">
      <div className="auth-card card">
        <div className="lock-avatar">{(switching ? switchUsername[0] : user?.name?.[0])?.toUpperCase() ?? '?'}</div>
        <h1 className="auth-title">{t('lock_title')}</h1>
        <p className="text-muted">{switching ? 'Switching cashier' : user?.name}</p>
        {shift && <p className="text-muted">{shift.register_name}</p>}

        {switching && (
          <label className="field">
            {t('login_username')}
            <input className="input" value={switchUsername} onChange={(e) => setSwitchUsername(e.target.value)} autoFocus />
          </label>
        )}

        <input className="input pin-display" type="password" value={pin} readOnly />

        <NumericKeypad
          allowDecimal={false}
          onDigit={(d) => setPin((p) => (p.length < 8 ? p + d : p))}
          onBackspace={() => setPin((p) => p.slice(0, -1))}
          onClear={() => setPin('')}
        />

        {error && <div className="auth-error">{error}</div>}

        <button className="btn btn-primary btn-lg btn-block" onClick={submit} disabled={busy || !pin || (switching && !switchUsername)}>
          {t('lock_unlock')}
        </button>
        <button className="btn btn-block" onClick={() => { setSwitching((s) => !s); setPin(''); setError('') }}>
          {switching ? 'Cancel' : t('topbar_switch_cashier')}
        </button>
        <button className="btn btn-ghost btn-block" onClick={logout}>
          {t('topbar_logout')}
        </button>
      </div>
    </div>
  )
}
