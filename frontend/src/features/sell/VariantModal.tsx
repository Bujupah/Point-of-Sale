import { useMemo, useState } from 'react'
import { X, Minus, Plus } from 'lucide-react'
import { useSettings } from '../../app/SettingsContext'
import { formatMoney } from '../../utils/money'
import type { CartModifierSelection, CartVariantSelection, ModifierGroup, Product, Variant, VariantGroup } from '../../types'

interface Props {
  product: Product
  onConfirm: (variants: CartVariantSelection[], modifiers: CartModifierSelection[], quantity: number, notes: string) => void
  onClose: () => void
}

type ModifierOptionSel = { group: ModifierGroup; options: number[] }

export function VariantModal({ product, onConfirm, onClose }: Props) {
  const { currencySymbol, currencyDecimals } = useSettings()
  const [selectedVariants, setSelectedVariants] = useState<Record<number, Variant>>({})
  const [selectedModifiers, setSelectedModifiers] = useState<Record<number, ModifierOptionSel>>({})
  const [quantity, setQuantity] = useState(1)
  const [notes, setNotes] = useState('')

  const variantGroups = product.variant_groups ?? []
  const modifierGroups = product.modifier_groups ?? []

  const total = useMemo(() => {
    let price = product.price
    for (const g of variantGroups) {
      const v = selectedVariants[g.id]
      if (v) price += v.price_adjustment
    }
    for (const g of modifierGroups) {
      const sel = selectedModifiers[g.id]
      if (sel) {
        for (const optId of sel.options) {
          const opt = g.options.find((o) => o.id === optId)
          if (opt) price += opt.price_adjustment
        }
      }
    }
    return price * quantity
  }, [product, variantGroups, modifierGroups, selectedVariants, selectedModifiers, quantity])

  function selectVariant(group: VariantGroup, variant: Variant) {
    setSelectedVariants((prev) => ({ ...prev, [group.id]: variant }))
  }

  function toggleModifier(group: ModifierGroup, optionId: number) {
    setSelectedModifiers((prev) => {
      const current = prev[group.id]?.options ?? []
      const isSelected = current.includes(optionId)
      let next: number[]
      if (group.max_select === 1) {
        next = isSelected ? [] : [optionId]
      } else if (isSelected) {
        next = current.filter((id) => id !== optionId)
      } else if (current.length >= group.max_select) {
        next = current // at capacity, ignore extra picks
      } else {
        next = [...current, optionId]
      }
      return { ...prev, [group.id]: { group, options: next } }
    })
  }

  const missingRequired = variantGroups.some((g) => g.required && !selectedVariants[g.id])
  const missingRequiredModifiers = modifierGroups.some((g) => g.required && (selectedModifiers[g.id]?.options.length ?? 0) < Math.max(1, g.min_select))

  function confirm() {
    const variants: CartVariantSelection[] = variantGroups
      .filter((g) => selectedVariants[g.id])
      .map((g) => ({ group: g, variant: selectedVariants[g.id] }))
    const modifiers: CartModifierSelection[] = []
    for (const g of modifierGroups) {
      const sel = selectedModifiers[g.id]
      if (!sel) continue
      for (const optId of sel.options) {
        const opt = g.options.find((o) => o.id === optId)
        if (opt) modifiers.push({ group: g, option: opt })
      }
    }
    onConfirm(variants, modifiers, quantity, notes)
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth: 480 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>{product.name}</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose}><X size={18} /></button>
        </div>
        <div className="modal-body variant-modal-body">
          {variantGroups.map((g) => (
            <div key={g.id} className="variant-group">
              <div className="variant-group-title">
                {g.name} {g.required && <span className="text-danger">*</span>}
              </div>
              <div className="variant-options">
                {g.variants.map((v) => (
                  <button
                    key={v.id}
                    className={`chip ${selectedVariants[g.id]?.id === v.id ? 'chip-active' : ''}`}
                    onClick={() => selectVariant(g, v)}
                  >
                    {v.name}
                    {v.price_adjustment !== 0 && ` (+${formatMoney(v.price_adjustment, currencyDecimals, currencySymbol)})`}
                  </button>
                ))}
              </div>
            </div>
          ))}

          {modifierGroups.map((g) => (
            <div key={g.id} className="variant-group">
              <div className="variant-group-title">
                {g.name} {g.required && <span className="text-danger">*</span>}
                <span className="text-muted"> ({g.max_select > 1 ? `up to ${g.max_select}` : 'choose one'})</span>
              </div>
              <div className="variant-options">
                {g.options.map((o) => (
                  <button
                    key={o.id}
                    className={`chip ${selectedModifiers[g.id]?.options.includes(o.id) ? 'chip-active' : ''}`}
                    onClick={() => toggleModifier(g, o.id)}
                  >
                    {o.name}
                    {o.price_adjustment !== 0 && ` (+${formatMoney(o.price_adjustment, currencyDecimals, currencySymbol)})`}
                  </button>
                ))}
              </div>
            </div>
          ))}

          <label className="field">
            Notes
            <textarea className="input" rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} placeholder="e.g. no onions" />
          </label>

          <div className="qty-row">
            <button className="btn btn-icon" onClick={() => setQuantity((q) => Math.max(1, q - 1))}><Minus size={16} /></button>
            <span className="qty-value">{quantity}</span>
            <button className="btn btn-icon" onClick={() => setQuantity((q) => q + 1)}><Plus size={16} /></button>
          </div>
        </div>
        <div className="modal-footer">
          <div className="variant-total">{formatMoney(total, currencyDecimals, currencySymbol)}</div>
          <button className="btn btn-primary btn-lg" disabled={missingRequired || missingRequiredModifiers} onClick={confirm}>
            Add to Cart
          </button>
        </div>
      </div>
    </div>
  )
}
