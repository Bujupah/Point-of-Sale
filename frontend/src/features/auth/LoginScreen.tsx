import { useEffect, useState } from 'react'
import { useI18n } from '../../i18n'
import { api, ApiError } from '../../services/api'
import { useAuth } from '../../app/AuthContext'
import { NumericKeypad } from '../../components/NumericKeypad'

interface RegisterInfo {
  id: number
  name: string
  location_id: number
  location_name: string
}

export function LoginScreen() {
  const { t } = useI18n()
  const { login } = useAuth()
  const [registers, setRegisters] = useState<RegisterInfo[]>([])
  const [registerId, setRegisterId] = useState<number | undefined>(undefined)
  const [username, setUsername] = useState('')
  const [pin, setPin] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    api
      .get<RegisterInfo[]>('/api/registers')
      .then((list) => {
        setRegisters(list ?? [])
        if (list?.length) setRegisterId(list[0].id)
      })
      .catch(() => {})
  }, [])

  async function submit(e?: React.FormEvent) {
    e?.preventDefault()
    setError('')
    setBusy(true)
    try {
      await login(username, pin, registerId)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Login failed')
      setPin('')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-screen">
      <form className="auth-card card" onSubmit={submit}>
        <h1 className="auth-title">{t('login_title')}</h1>

        <label className="field">
          {t('login_username')}
          <input className="input" value={username} onChange={(e) => setUsername(e.target.value)} autoFocus />
        </label>

        <label className="field">
          {t('login_pin')}
          <input className="input pin-display" type="password" value={pin} readOnly inputMode="numeric" />
        </label>

        {registers.length > 0 && (
          <label className="field">
            Register
            <select className="input" value={registerId} onChange={(e) => setRegisterId(Number(e.target.value))}>
              {registers.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.location_name} — {r.name}
                </option>
              ))}
            </select>
          </label>
        )}

        <NumericKeypad
          allowDecimal={false}
          onDigit={(d) => setPin((p) => (p.length < 8 ? p + d : p))}
          onBackspace={() => setPin((p) => p.slice(0, -1))}
          onClear={() => setPin('')}
        />

        {error && <div className="auth-error">{error}</div>}

        <button className="btn btn-primary btn-lg btn-block" type="submit" disabled={busy || !username || !pin}>
          {t('login_submit')}
        </button>

        <p className="text-muted auth-hint">Demo: admin / 1234 or cashier / 1111</p>
      </form>
    </div>
  )
}
