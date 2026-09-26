import { useState } from 'react'
import { X, CheckCircle2 } from 'lucide-react'
import { useI18n } from '../../i18n'
import { useSettings } from '../../app/SettingsContext'
import { useShift } from '../../app/ShiftContext'
import { formatMoney, parseMoneyInput } from '../../utils/money'
import { ApiError } from '../../services/api'
import type { ZReport } from '../../types'

export function CloseShiftModal({ report, onClose }: { report: ZReport; onClose: () => void }) {
  const { t } = useI18n()
  const { currencySymbol, currencyDecimals } = useSettings()
  const { closeShift } = useShift()
  const [counted, setCounted] = useState('')
  const [notes, setNotes] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [closed, setClosed] = useState<{ expected: number; counted: number; difference: number } | null>(null)

  const countedMinor = parseMoneyInput(counted || '0', currencyDecimals)
  const difference = countedMinor - report.expected_cash

  async function submit() {
    setBusy(true)
    setError('')
    try {
      const result = await closeShift(countedMinor, notes)
      setClosed({ expected: result.expected_cash ?? 0, counted: result.counted_cash ?? 0, difference: result.difference ?? 0 })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not close shift')
    } finally {
      setBusy(false)
    }
  }

  if (closed) {
    return (
      <div className="modal-overlay">
        <div className="modal" style={{ maxWidth: 380 }}>
          <div className="modal-body stack" style={{ textAlign: 'center' }}>
            <div className="sale-complete-icon"><CheckCircle2 size={48} className="text-success" /></div>
            <h2>{t('shift_z_report')}</h2>
            <div className="summary-row"><span>{t('shift_expected_cash')}</span><span>{formatMoney(closed.expected, currencyDecimals, currencySymbol)}</span></div>
            <div className="summary-row"><span>{t('shift_counted_cash')}</span><span>{formatMoney(closed.counted, currencyDecimals, currencySymbol)}</span></div>
            <div className={`summary-row summary-total ${closed.difference !== 0 ? 'text-danger' : 'text-success'}`}>
              <span>{t('shift_difference')}</span><span>{formatMoney(closed.difference, currencyDecimals, currencySymbol)}</span>
            </div>
            <a className="btn btn-block" href={`/api/shifts/${report.shift.id}/report.csv`} target="_blank" rel="noreferrer">
              Download CSV
            </a>
            <button className="btn btn-primary btn-block" onClick={onClose}>{t('common_close')}</button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth: 400 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>{t('shift_close')}</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose}><X size={18} /></button>
        </div>
        <div className="modal-body stack">
          <div className="summary-row"><span>{t('shift_expected_cash')}</span><span>{formatMoney(report.expected_cash, currencyDecimals, currencySymbol)}</span></div>
          <label className="field">
            {t('shift_counted_cash')}
            <input className="input" value={counted} onChange={(e) => setCounted(e.target.value)} inputMode="decimal" autoFocus />
          </label>
          {counted && (
            <div className={`summary-row ${difference !== 0 ? 'text-danger' : 'text-success'}`}>
              <span>{t('shift_difference')}</span>
              <span>{formatMoney(difference, currencyDecimals, currencySymbol)}</span>
            </div>
          )}
          <label className="field">
            {t('common_notes')}
            <textarea className="input" value={notes} onChange={(e) => setNotes(e.target.value)} />
          </label>
          {error && <div className="auth-error">{error}</div>}
        </div>
        <div className="modal-footer">
          <button className="btn" onClick={onClose}>{t('common_cancel')}</button>
          <button className="btn btn-primary" disabled={busy || !counted} onClick={submit}>{t('shift_close')}</button>
        </div>
      </div>
    </div>
  )
}
