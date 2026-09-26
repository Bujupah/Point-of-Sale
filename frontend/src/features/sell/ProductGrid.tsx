import { useEffect, useState } from 'react'
import { api, qs } from '../../services/api'
import { useI18n } from '../../i18n'
import { useSettings } from '../../app/SettingsContext'
import { formatMoney } from '../../utils/money'
import type { Product } from '../../types'

interface Props {
  query: string
  categoryId: number | 'all' | 'favorites'
  onSelect: (product: Product) => void
  reloadToken: number
}

export function ProductGrid({ query, categoryId, onSelect, reloadToken }: Props) {
  const { t } = useI18n()
  const { currencySymbol, currencyDecimals } = useSettings()
  const [products, setProducts] = useState<Product[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    const params: Record<string, string | number> = { limit: 120 }
    if (query) params.q = query
    if (typeof categoryId === 'number') params.category_id = categoryId
    const handle = setTimeout(() => {
      api
        .get<Product[]>(`/api/products${qs(params)}`)
        .then((list) => {
          let result = list ?? []
          if (categoryId === 'favorites') result = result.filter((p) => p.favorite)
          setProducts(result)
        })
        .catch(() => setProducts([]))
        .finally(() => setLoading(false))
    }, 150)
    return () => clearTimeout(handle)
  }, [query, categoryId, reloadToken])

  if (loading && products.length === 0) {
    return (
      <div className="empty-state">
        <div className="spinner" />
      </div>
    )
  }

  if (products.length === 0) {
    return <div className="empty-state">No products match.</div>
  }

  return (
    <div className="product-grid">
      {products.map((p) => {
        const outOfStock = p.stock_status === 'OUT_OF_STOCK'
        return (
          <button
            key={p.id}
            className={`product-card ${outOfStock ? 'product-card-disabled' : ''}`}
            disabled={outOfStock}
            onClick={() => onSelect(p)}
          >
            <div className="product-card-image">
              {p.image_thumbnail ? <img src={p.image_thumbnail} alt="" loading="lazy" /> : <span className="product-card-placeholder">{p.name[0]}</span>}
              <div className="product-card-badges">
                {p.original_price != null && p.original_price > p.price && <span className="badge badge-sale">{t('badge_sale')}</span>}
                {p.stock_status === 'LOW_STOCK' && <span className="badge badge-low">{t('badge_low_stock')}</span>}
                {outOfStock && <span className="badge badge-out">{t('badge_out_of_stock')}</span>}
              </div>
            </div>
            <div className="product-card-name">{p.name}</div>
            <div className="product-card-price">
              {formatMoney(p.price, currencyDecimals, currencySymbol)}
              {p.original_price != null && p.original_price > p.price && (
                <span className="product-card-original-price">{formatMoney(p.original_price, currencyDecimals, currencySymbol)}</span>
              )}
            </div>
          </button>
        )
      })}
    </div>
  )
}
