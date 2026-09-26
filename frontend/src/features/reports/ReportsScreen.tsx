import { useEffect, useState } from 'react'
import { useI18n } from '../../i18n'
import { useSettings } from '../../app/SettingsContext'
import { api, qs } from '../../services/api'
import { formatMoney } from '../../utils/money'

interface Summary {
  revenue: number
  order_count: number
  average_ticket: number
  discount_total: number
  tax_total: number
  refunds_total: number
  cash_amount: number
  card_amount: number
  other_amount: number
  cash_percent_bps: number
  card_percent_bps: number
}

interface TopProduct {
  name: string
  quantity_sold: number
  sales_amount: number
  percent_of_revenue_bps: number
}

type RangeKey = 'today' | 'yesterday' | '7days' | 'custom'

function rangeToDates(key: RangeKey, from: string, to: string): { date_from?: string; date_to?: string } {
  const now = new Date()
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).toISOString()
  const endOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate(), 23, 59, 59, 999).toISOString()
  if (key === 'today') return { date_from: startOfDay(now), date_to: endOfDay(now) }
  if (key === 'yesterday') {
    const y = new Date(now)
    y.setDate(y.getDate() - 1)
    return { date_from: startOfDay(y), date_to: endOfDay(y) }
  }
  if (key === '7days') {
    const start = new Date(now)
    start.setDate(start.getDate() - 6)
    return { date_from: startOfDay(start), date_to: endOfDay(now) }
  }
  return { date_from: from || undefined, date_to: to || undefined }
}

export function ReportsScreen() {
  const { t } = useI18n()
  const { currencySymbol, currencyDecimals } = useSettings()
  const [rangeKey, setRangeKey] = useState<RangeKey>('today')
  const [customFrom, setCustomFrom] = useState('')
  const [customTo, setCustomTo] = useState('')
  const [summary, setSummary] = useState<Summary | null>(null)
  const [topProducts, setTopProducts] = useState<TopProduct[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const dates = rangeToDates(rangeKey, customFrom, customTo)
    setLoading(true)
    Promise.all([
      api.get<Summary>(`/api/reports/summary${qs(dates)}`),
      api.get<TopProduct[]>(`/api/reports/products${qs({ ...dates, limit: 10 })}`),
    ])
      .then(([s, p]) => {
        setSummary(s)
        setTopProducts(p ?? [])
      })
      .finally(() => setLoading(false))
  }, [rangeKey, customFrom, customTo])

  const dates = rangeToDates(rangeKey, customFrom, customTo)

  return (
    <div className="page">
      <div className="page-header">
        <h1>{t('nav_reports')}</h1>
        <div className="toolbar">
          <button className={`chip ${rangeKey === 'today' ? 'chip-active' : ''}`} onClick={() => setRangeKey('today')}>{t('reports_today')}</button>
          <button className={`chip ${rangeKey === 'yesterday' ? 'chip-active' : ''}`} onClick={() => setRangeKey('yesterday')}>{t('reports_yesterday')}</button>
          <button className={`chip ${rangeKey === '7days' ? 'chip-active' : ''}`} onClick={() => setRangeKey('7days')}>{t('reports_7days')}</button>
          <button className={`chip ${rangeKey === 'custom' ? 'chip-active' : ''}`} onClick={() => setRangeKey('custom')}>{t('reports_custom')}</button>
          {rangeKey === 'custom' && (
            <>
              <input className="input" type="date" value={customFrom} onChange={(e) => setCustomFrom(e.target.value)} />
              <input className="input" type="date" value={customTo} onChange={(e) => setCustomTo(e.target.value)} />
            </>
          )}
          <a className="btn" href={`/api/reports/summary.csv${qs(dates)}`} target="_blank" rel="noreferrer">CSV</a>
        </div>
      </div>

      {loading || !summary ? (
        <div className="spinner" />
      ) : (
        <>
          <div className="kpi-grid">
            <Kpi label={t('reports_revenue')} value={formatMoney(summary.revenue, currencyDecimals, currencySymbol)} />
            <Kpi label={t('reports_orders')} value={String(summary.order_count)} />
            <Kpi label={t('reports_average_ticket')} value={formatMoney(summary.average_ticket, currencyDecimals, currencySymbol)} />
            <Kpi label="Cash %" value={`${(summary.cash_percent_bps / 100).toFixed(1)}%`} />
            <Kpi label="Card %" value={`${(summary.card_percent_bps / 100).toFixed(1)}%`} />
            <Kpi label={t('shift_refunds')} value={formatMoney(summary.refunds_total, currencyDecimals, currencySymbol)} />
            <Kpi label={t('cart_discount')} value={formatMoney(summary.discount_total, currencyDecimals, currencySymbol)} />
            <Kpi label={t('cart_tax')} value={formatMoney(summary.tax_total, currencyDecimals, currencySymbol)} />
          </div>

          <h3>{t('reports_top_products')}</h3>
          <table className="table">
            <thead>
              <tr><th>{t('common_name')}</th><th>Qty</th><th>{t('common_total')}</th><th>%</th></tr>
            </thead>
            <tbody>
              {topProducts.map((p) => (
                <tr key={p.name}>
                  <td>{p.name}</td>
                  <td>{p.quantity_sold}</td>
                  <td>{formatMoney(p.sales_amount, currencyDecimals, currencySymbol)}</td>
                  <td>{(p.percent_of_revenue_bps / 100).toFixed(1)}%</td>
                </tr>
              ))}
            </tbody>
          </table>
          {topProducts.length === 0 && <div className="empty-state">No sales in this range.</div>}
        </>
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
