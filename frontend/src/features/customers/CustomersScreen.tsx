import { useEffect, useState } from 'react'
import { useI18n } from '../../i18n'
import { useSettings } from '../../app/SettingsContext'
import { api, qs } from '../../services/api'
import { formatMoney } from '../../utils/money'
import type { Customer } from '../../types'
import { CustomerDetailModal } from './CustomerDetailModal'
import { CustomerEditModal } from './CustomerEditModal'

export function CustomersScreen() {
  const { t } = useI18n()
  const { currencySymbol, currencyDecimals } = useSettings()
  const [customers, setCustomers] = useState<Customer[]>([])
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<Customer | null>(null)
  const [creating, setCreating] = useState(false)

  async function load() {
    setLoading(true)
    try {
      const list = await api.get<Customer[]>(`/api/customers${qs({ q: search, limit: 100 })}`)
      setCustomers(list ?? [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const handle = setTimeout(load, 200)
    return () => clearTimeout(handle)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search])

  return (
    <div className="page">
      <div className="page-header">
        <h1>{t('nav_customers')}</h1>
        <div className="toolbar">
          <input className="input" style={{ width: 260 }} placeholder={t('common_search')} value={search} onChange={(e) => setSearch(e.target.value)} />
          <button className="btn btn-primary" onClick={() => setCreating(true)}>{t('customers_add')}</button>
        </div>
      </div>

      {loading ? (
        <div className="spinner" />
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>{t('common_name')}</th>
              <th>{t('common_phone')}</th>
              <th>{t('customers_loyalty_points')}</th>
              <th>{t('customers_visit_count')}</th>
              <th>{t('customers_lifetime_spend')}</th>
            </tr>
          </thead>
          <tbody>
            {customers.map((c) => (
              <tr key={c.id} onClick={() => setSelected(c)} style={{ cursor: 'pointer' }}>
                <td>{c.name}</td>
                <td>{c.phone}</td>
                <td>{c.loyalty_points}</td>
                <td>{c.visit_count}</td>
                <td>{formatMoney(c.lifetime_spend, currencyDecimals, currencySymbol)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {customers.length === 0 && !loading && <div className="empty-state">No customers yet.</div>}

      {selected && <CustomerDetailModal customer={selected} onClose={() => setSelected(null)} onChanged={load} />}
      {creating && (
        <CustomerEditModal
          onClose={() => setCreating(false)}
          onSaved={() => {
            setCreating(false)
            load()
          }}
        />
      )}
    </div>
  )
}
