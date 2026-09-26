import { useEffect, useState } from 'react'
import { useI18n } from '../../i18n'
import { useSettings } from '../../app/SettingsContext'
import { useShift } from '../../app/ShiftContext'
import { useAuth } from '../../app/AuthContext'
import { api, ApiError } from '../../services/api'
import { formatMoney } from '../../utils/money'
import type { ZReport } from '../../types'
import { OpenShiftPrompt } from './OpenShiftPrompt'
import { CashMovementModal } from './CashMovementModal'
import { CloseShiftModal } from './CloseShiftModal'

export function RegisterScreen() {
  const { t } = useI18n()
  const { currencySymbol, currencyDecimals } = useSettings()
  const { shift, loading } = useShift()
  const { hasPermission } = useAuth()
  const [report, setReport] = useState<ZReport | null>(null)
  const [movementModal, setMovementModal] = useState<'CASH_IN' | 'CASH_OUT' | null>(null)
  const [closeOpen, setCloseOpen] = useState(false)

  async function loadReport() {
    if (!shift) return
    try {
      const r = await api.get<ZReport>(`/api/shifts/${shift.id}/report`)
      setReport(r)
    } catch {
      /* non-fatal */
    }
  }

  useEffect(() => {
    loadReport()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [shift])

  async function openDrawer() {
    try {
      await api.post('/api/hardware/drawer/open')
    } catch (err) {
      alert(err instanceof ApiError ? err.message : 'Could not open drawer')
    }
  }

  if (loading) return <div className="empty-state" style={{ width: '100%' }}><div className="spinner" /></div>
  if (!shift) return <OpenShiftPrompt />

  return (
    <div className="page">
      <div className="page-header">
        <h1>{t('nav_register')}</h1>
        <div className="toolbar">
          {hasPermission('drawer.open') && <button className="btn" onClick={openDrawer}>Open Drawer</button>}
          {hasPermission('cash.in') && <button className="btn" onClick={() => setMovementModal('CASH_IN')}>{t('shift_cash_in')}</button>}
          {hasPermission('cash.out') && <button className="btn" onClick={() => setMovementModal('CASH_OUT')}>{t('shift_cash_out')}</button>}
          <button className="btn btn-primary" onClick={() => setCloseOpen(true)}>{t('shift_close')}</button>
        </div>
      </div>

      <p className="text-muted">{shift.register_name} · {shift.cashier_name} · opened {new Date(shift.opening_at).toLocaleString()}</p>

      {report && (
        <div className="kpi-grid">
          <Kpi label={t('shift_expected_cash')} value={formatMoney(report.expected_cash, currencyDecimals, currencySymbol)} />
          <Kpi label={t('shift_cash_sales')} value={formatMoney(report.cash_sales, currencyDecimals, currencySymbol)} />
          <Kpi label={t('shift_total_sales')} value={formatMoney(report.net_sales, currencyDecimals, currencySymbol)} />
          <Kpi label={t('shift_ticket_count')} value={String(report.order_count)} />
          <Kpi label={t('shift_refunds')} value={formatMoney(report.refunds_total, currencyDecimals, currencySymbol)} />
          <Kpi label={t('shift_cash_in')} value={formatMoney(report.cash_in, currencyDecimals, currencySymbol)} />
          <Kpi label={t('shift_cash_out')} value={formatMoney(report.cash_out, currencyDecimals, currencySymbol)} />
        </div>
      )}

      {movementModal && (
        <CashMovementModal
          type={movementModal}
          onClose={() => setMovementModal(null)}
          onDone={() => {
            setMovementModal(null)
            loadReport()
          }}
        />
      )}
      {closeOpen && report && (
        <CloseShiftModal
          report={report}
          onClose={() => setCloseOpen(false)}
        />
      )}
    </div>
  )
}

function Kpi({ label, value }: { label: string; value: string }) {
  return (
    <div className="card kpi-card">
      <div className="kpi-label">{label}</div>
      <div className="kpi-value">{value}</div>
    </div>
  )
}
