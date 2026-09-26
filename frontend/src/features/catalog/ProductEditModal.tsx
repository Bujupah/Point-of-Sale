import { useEffect, useState } from 'react'
import { useSettings } from '../../app/SettingsContext'
import { api, ApiError } from '../../services/api'
import { formatMoney, parseMoneyInput } from '../../utils/money'
import type { Category, ModifierGroup, Product, VariantGroup } from '../../types'

interface TaxRate {
  id: number
  name: string
  rate_bps: number
}

export function ProductEditModal({ product, onClose, onSaved }: { product?: Product; onClose: () => void; onSaved: () => void }) {
  const { currencySymbol, currencyDecimals } = useSettings()
  const [categories, setCategories] = useState<Category[]>([])
  const [taxRates, setTaxRates] = useState<TaxRate[]>([])
  const [modifierLibrary, setModifierLibrary] = useState<ModifierGroup[]>([])

  const [sku, setSku] = useState(product?.sku ?? '')
  const [name, setName] = useState(product?.name ?? '')
  const [description, setDescription] = useState(product?.description ?? '')
  const [categoryId, setCategoryId] = useState<number | ''>(product?.category_id ?? '')
  const [price, setPrice] = useState(product ? (product.price / 10 ** currencyDecimals).toFixed(currencyDecimals) : '')
  const [cost, setCost] = useState(product ? (product.cost / 10 ** currencyDecimals).toFixed(currencyDecimals) : '0')
  const [taxRateId, setTaxRateId] = useState<number | ''>(product?.tax_rate_id ?? '')
  const [trackStock, setTrackStock] = useState(product?.track_stock ?? true)
  const [reorderLevel, setReorderLevel] = useState(String(product?.reorder_level ?? 5))
  const [initialStock, setInitialStock] = useState('0')
  const [barcodes, setBarcodes] = useState<string[]>(product?.barcodes ?? [])
  const [newBarcode, setNewBarcode] = useState('')
  const [active, setActive] = useState(product?.active ?? true)
  const [favorite, setFavorite] = useState(product?.favorite ?? false)
  const [modifierGroupIds, setModifierGroupIds] = useState<number[]>(product?.modifier_groups?.map((g) => g.id) ?? [])
  const [variantGroups, setVariantGroups] = useState<VariantGroup[]>(product?.variant_groups ?? [])

  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    api.get<Category[]>('/api/categories').then((r) => setCategories(r ?? [])).catch(() => {})
    api.get<TaxRate[]>('/api/tax-rates').then((r) => setTaxRates(r ?? [])).catch(() => {})
    api.get<ModifierGroup[]>('/api/modifier-groups').then((r) => setModifierLibrary(r ?? [])).catch(() => {})
  }, [])

  function addVariantGroup() {
    setVariantGroups((prev) => [...prev, { id: -Date.now(), name: 'New Option Group', required: true, sort_order: prev.length, variants: [] }])
  }
  function addVariant(groupIdx: number) {
    setVariantGroups((prev) =>
      prev.map((g, i) => (i === groupIdx ? { ...g, variants: [...g.variants, { id: -Date.now(), name: 'Option', price_adjustment: 0, sort_order: g.variants.length }] } : g)),
    )
  }

  async function save() {
    setBusy(true)
    setError('')
    try {
      const payload = {
        sku,
        name,
        description,
        category_id: categoryId === '' ? undefined : categoryId,
        price: parseMoneyInput(price, currencyDecimals),
        cost: parseMoneyInput(cost, currencyDecimals),
        tax_rate_id: taxRateId === '' ? undefined : taxRateId,
        track_stock: trackStock,
        initial_stock: Number(initialStock) || 0,
        reorder_level: Number(reorderLevel) || 0,
        active,
        favorite,
        barcodes,
        modifier_group_ids: modifierGroupIds,
        variant_groups: variantGroups.map((g) => ({
          name: g.name,
          required: g.required,
          sort_order: g.sort_order,
          variants: g.variants.map((v) => ({ name: v.name, price_adjustment: v.price_adjustment, sort_order: v.sort_order })),
        })),
      }
      if (product) await api.put(`/api/products/${product.id}`, payload)
      else await api.post('/api/products', payload)
      onSaved()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Save failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth: 560 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>{product ? 'Edit Product' : 'New Product'}</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose}>✕</button>
        </div>
        <div className="modal-body stack">
          <div className="form-grid-2">
            <label className="field">SKU<input className="input" value={sku} onChange={(e) => setSku(e.target.value)} /></label>
            <label className="field">Name<input className="input" value={name} onChange={(e) => setName(e.target.value)} /></label>
          </div>
          <label className="field">Description<textarea className="input" value={description} onChange={(e) => setDescription(e.target.value)} /></label>
          <div className="form-grid-2">
            <label className="field">
              Category
              <select className="input" value={categoryId} onChange={(e) => setCategoryId(e.target.value ? Number(e.target.value) : '')}>
                <option value="">—</option>
                {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </label>
            <label className="field">
              Tax rate
              <select className="input" value={taxRateId} onChange={(e) => setTaxRateId(e.target.value ? Number(e.target.value) : '')}>
                <option value="">None</option>
                {taxRates.map((t) => <option key={t.id} value={t.id}>{t.name} ({(t.rate_bps / 100).toFixed(2)}%)</option>)}
              </select>
            </label>
          </div>
          <div className="form-grid-2">
            <label className="field">Price ({currencySymbol})<input className="input" value={price} onChange={(e) => setPrice(e.target.value)} /></label>
            <label className="field">Cost ({currencySymbol})<input className="input" value={cost} onChange={(e) => setCost(e.target.value)} /></label>
          </div>

          <label className="field-inline">
            <input type="checkbox" checked={trackStock} onChange={(e) => setTrackStock(e.target.checked)} /> Track stock
          </label>
          {trackStock && (
            <div className="form-grid-2">
              <label className="field">Reorder level<input className="input" value={reorderLevel} onChange={(e) => setReorderLevel(e.target.value)} /></label>
              {!product && <label className="field">Initial stock<input className="input" value={initialStock} onChange={(e) => setInitialStock(e.target.value)} /></label>}
            </div>
          )}

          <div className="field">
            Barcodes
            <div className="toolbar">
              <input className="input" value={newBarcode} onChange={(e) => setNewBarcode(e.target.value)} placeholder="Scan or type..." />
              <button className="btn" onClick={() => { if (newBarcode) { setBarcodes((b) => [...b, newBarcode]); setNewBarcode('') } }}>Add</button>
            </div>
            <div className="toolbar" style={{ marginTop: 8 }}>
              {barcodes.map((b) => (
                <span key={b} className="chip chip-active" onClick={() => setBarcodes((bc) => bc.filter((x) => x !== b))}>{b} ✕</span>
              ))}
            </div>
          </div>

          <div className="field">
            Modifier groups (reusable)
            <div className="toolbar">
              {modifierLibrary.map((g) => (
                <button
                  key={g.id}
                  className={`chip ${modifierGroupIds.includes(g.id) ? 'chip-active' : ''}`}
                  onClick={() => setModifierGroupIds((ids) => (ids.includes(g.id) ? ids.filter((i) => i !== g.id) : [...ids, g.id]))}
                >
                  {g.name}
                </button>
              ))}
            </div>
          </div>

          <div className="field">
            <div className="toolbar" style={{ justifyContent: 'space-between' }}>
              <span>Variant groups (this product only)</span>
              <button className="btn" onClick={addVariantGroup}>+ Add group</button>
            </div>
            {variantGroups.map((g, gi) => (
              <div key={g.id} className="card variant-editor-group">
                <input className="input" value={g.name} onChange={(e) => setVariantGroups((prev) => prev.map((x, i) => (i === gi ? { ...x, name: e.target.value } : x)))} />
                {g.variants.map((v, vi) => (
                  <div key={v.id} className="toolbar">
                    <input
                      className="input"
                      value={v.name}
                      onChange={(e) =>
                        setVariantGroups((prev) => prev.map((x, i) => (i === gi ? { ...x, variants: x.variants.map((vv, j) => (j === vi ? { ...vv, name: e.target.value } : vv)) } : x)))
                      }
                    />
                    <input
                      className="input"
                      style={{ width: 100 }}
                      value={v.price_adjustment}
                      onChange={(e) =>
                        setVariantGroups((prev) =>
                          prev.map((x, i) => (i === gi ? { ...x, variants: x.variants.map((vv, j) => (j === vi ? { ...vv, price_adjustment: Number(e.target.value) || 0 } : vv)) } : x)),
                        )
                      }
                    />
                  </div>
                ))}
                <button className="btn" onClick={() => addVariant(gi)}>+ Add option</button>
              </div>
            ))}
          </div>

          <div className="toolbar">
            <label className="field-inline"><input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} /> Active</label>
            <label className="field-inline"><input type="checkbox" checked={favorite} onChange={(e) => setFavorite(e.target.checked)} /> Favorite</label>
          </div>

          {price && <div className="text-muted">Preview: {formatMoney(parseMoneyInput(price, currencyDecimals), currencyDecimals, currencySymbol)}</div>}
          {error && <div className="auth-error">{error}</div>}
        </div>
        <div className="modal-footer">
          <button className="btn" onClick={onClose}>Cancel</button>
          <button className="btn btn-primary" disabled={busy || !sku || !name || !price} onClick={save}>Save</button>
        </div>
      </div>
    </div>
  )
}
