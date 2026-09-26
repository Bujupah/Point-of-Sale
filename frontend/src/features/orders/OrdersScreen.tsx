import { useEffect, useState } from 'react'
import { useI18n } from '../../i18n'
import { useSettings } from '../../app/SettingsContext'
import { api, qs } from '../../services/api'
import { formatMoney } from '../../utils/money'
import type { Sale } from '../../types'
import { OrderDetailModal } from './OrderDetailModal'

export function OrdersScreen() {
  const { t } = useI18n()
  const { currencySymbol, currencyDecimals } = useSettings()
  const [sales, setSales] = useState<Sale[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<Sale | null>(null)

  async function load() {
    setLoading(true)
    try {
      const list = await api.get<Sale[]>(`/api/sales${qs({ receipt_number: search, limit: 100 })}`)
      setSales(list ?? [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const handle = setTimeout(load, 200)
    return () => clearTimeout(handle)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search])

  async function openDetail(id: number) {
    const full = await api.get<Sale>(`/api/sales/${id}`)
    setSelected(full)
  }

  return (
    <div className="page">
      <div className="page-header">
        <h1>{t('nav_orders')}</h1>
        <input className="input" style={{ width: 260 }} placeholder="Search receipt #..." value={search} onChange={(e) => setSearch(e.target.value)} />
      </div>

      {loading ? (
        <div className="spinner" />
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>{t('orders_receipt')}</th>
              <th>{t('common_date')}</th>
              <th>{t('orders_cashier')}</th>
              <th>{t('orders_customer')}</th>
              <th>{t('common_status')}</th>
              <th>{t('common_total')}</th>
            </tr>
          </thead>
          <tbody>
            {sales.map((s) => (
              <tr key={s.id} onClick={() => openDetail(s.id)} style={{ cursor: 'pointer' }}>
                <td>{s.receipt_number}</td>
                <td>{new Date(s.created_at).toLocaleString()}</td>
                <td>{s.cashier_name}</td>
                <td>{s.customer_name || '—'}</td>
                <td>{s.status}</td>
                <td>{formatMoney(s.total, currencyDecimals, currencySymbol)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {sales.length === 0 && !loading && <div className="empty-state">No orders yet.</div>}

      {selected && (
        <OrderDetailModal
          sale={selected}
          onClose={() => setSelected(null)}
          onChanged={() => {
            setSelected(null)
            load()
          }}
        />
      )}
    </div>
  )
}
