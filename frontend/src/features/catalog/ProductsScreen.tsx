import { useEffect, useState } from 'react'
import { useI18n } from '../../i18n'
import { useSettings } from '../../app/SettingsContext'
import { api, qs } from '../../services/api'
import { formatMoney } from '../../utils/money'
import type { Product } from '../../types'
import { ProductEditModal } from './ProductEditModal'

export function ProductsScreen() {
  const { t } = useI18n()
  const { currencySymbol, currencyDecimals } = useSettings()
  const [products, setProducts] = useState<Product[]>([])
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState<Product | 'new' | null>(null)

  async function load() {
    setLoading(true)
    try {
      const list = await api.get<Product[]>(`/api/products${qs({ q: search, limit: 200 })}`)
      setProducts(list ?? [])
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
        <h1>{t('nav_products')}</h1>
        <div className="toolbar">
          <input className="input" style={{ width: 260 }} placeholder={t('common_search')} value={search} onChange={(e) => setSearch(e.target.value)} />
          <button className="btn btn-primary" onClick={() => setEditing('new')}>{t('products_add')}</button>
        </div>
      </div>

      {loading ? (
        <div className="spinner" />
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>{t('products_sku')}</th>
              <th>{t('common_name')}</th>
              <th>{t('products_price')}</th>
              <th>{t('products_stock')}</th>
              <th>{t('common_status')}</th>
            </tr>
          </thead>
          <tbody>
            {products.map((p) => (
              <tr key={p.id} onClick={() => setEditing(p)} style={{ cursor: 'pointer' }}>
                <td>{p.sku}</td>
                <td>{p.name}</td>
                <td>{formatMoney(p.price, currencyDecimals, currencySymbol)}</td>
                <td>{p.track_stock ? p.stock : '—'}{p.stock_status && <span className={`badge ${p.stock_status === 'OUT_OF_STOCK' ? 'badge-out' : 'badge-low'}`} style={{ marginInlineStart: 8 }}>{t(p.stock_status === 'OUT_OF_STOCK' ? 'badge_out_of_stock' : 'badge_low_stock')}</span>}</td>
                <td>{p.active ? 'Active' : 'Inactive'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {products.length === 0 && !loading && <div className="empty-state">No products found.</div>}

      {editing && (
        <ProductEditModal
          product={editing === 'new' ? undefined : editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null)
            load()
          }}
        />
      )}
    </div>
  )
}
