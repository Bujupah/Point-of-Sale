import { useState } from 'react'
import { useI18n } from '../../i18n'
import { useSettings } from '../../app/SettingsContext'
import { api, ApiError } from '../../services/api'
import { formatMoney } from '../../utils/money'
import type { Sale } from '../../types'

export function SaleCompleteScreen({ sale, printError, onNewSale }: { sale: Sale; printError?: string; onNewSale: () => void }) {
  const { t } = useI18n()
  const { currencySymbol, currencyDecimals } = useSettings()
  const [preview, setPreview] = useState<string | null>(null)
  const [reprintError, setReprintError] = useState('')

  async function showReceipt() {
    try {
      const res = await api.get<{ preview: string }>(`/api/sales/${sale.id}/receipt-preview`)
      setPreview(res.preview)
    } catch (err) {
      setReprintError(err instanceof ApiError ? err.message : 'Could not load receipt')
    }
  }

  async function reprint() {
    setReprintError('')
    try {
      await api.post(`/api/sales/${sale.id}/print`)
    } catch (err) {
      setReprintError(err instanceof ApiError ? err.message : 'Printer unavailable')
    }
  }

  return (
    <div className="modal-overlay">
      <div className="modal" style={{ maxWidth: 420 }}>
        <div className="modal-body sale-complete">
          <div className="sale-complete-icon">✅</div>
          <h2>{t('sale_success')}</h2>
          <div className="text-muted">{sale.receipt_number}</div>

          <div className="sale-complete-total">{formatMoney(sale.total, currencyDecimals, currencySymbol)}</div>

          {(sale.payments ?? []).map((p) => (
            <div key={p.id} className="summary-row">
              <span>{p.method}</span>
              <span>{formatMoney(p.amount, currencyDecimals, currencySymbol)}</span>
            </div>
          ))}
          {sale.change_due > 0 && (
            <div className="summary-row summary-total">
              <span>{t('payment_change')}</span>
              <span>{formatMoney(sale.change_due, currencyDecimals, currencySymbol)}</span>
            </div>
          )}

          {printError && <div className="auth-error">Sale completed. Receipt could not be printed: {printError}</div>}
          {reprintError && <div className="auth-error">{reprintError}</div>}

          {preview && <pre className="receipt-preview">{preview}</pre>}

          <div className="stack">
            <button className="btn btn-primary btn-lg btn-block" onClick={onNewSale} autoFocus>
              {t('sale_new')}
            </button>
            <div className="toolbar">
              <button className="btn btn-block" onClick={reprint}>
                {t('sale_print_receipt')}
              </button>
              <button className="btn btn-block" onClick={showReceipt}>
                {t('sale_view_receipt')}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
