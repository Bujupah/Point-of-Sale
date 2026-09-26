import { useState } from 'react'
import { useSettings } from '../../app/SettingsContext'
import { formatMoney } from '../../utils/money'
import { unitPriceOf } from '../../utils/pricing'
import type { CartLine } from '../../types'
import { useSell } from '../../app/SellContext'

export function CartItemRow({ line, lineTotal }: { line: CartLine; lineTotal: number }) {
  const { updateQuantity, removeLine, updateLineNotes } = useSell()
  const { currencySymbol, currencyDecimals } = useSettings()
  const [editingNote, setEditingNote] = useState(false)
  const [noteDraft, setNoteDraft] = useState(line.notes ?? '')

  const name = line.product?.name ?? line.openItemName ?? 'Item'
  const variantText = [...line.variantSelections.map((v) => v.variant.name), ...line.modifierSelections.map((m) => m.option.name)].join(', ')
  const unitPrice = unitPriceOf(line)

  return (
    <div className="cart-row">
      <div className="cart-row-main">
        <div className="cart-row-name">{name}</div>
        {variantText && <div className="cart-row-variant text-muted">{variantText}</div>}
        {editingNote ? (
          <input
            className="input cart-row-note-input"
            autoFocus
            value={noteDraft}
            onChange={(e) => setNoteDraft(e.target.value)}
            onBlur={() => {
              updateLineNotes(line.lineId, noteDraft)
              setEditingNote(false)
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') e.currentTarget.blur()
            }}
          />
        ) : (
          <button className="cart-row-note-btn text-muted" onClick={() => setEditingNote(true)}>
            {line.notes ? `📝 ${line.notes}` : '+ note'}
          </button>
        )}
      </div>
      <div className="cart-row-qty">
        <button className="btn btn-icon" onClick={() => updateQuantity(line.lineId, line.quantity - 1)}>−</button>
        <span>{line.quantity}</span>
        <button className="btn btn-icon" onClick={() => updateQuantity(line.lineId, line.quantity + 1)}>+</button>
      </div>
      <div className="cart-row-price">
        <div>{formatMoney(lineTotal, currencyDecimals, currencySymbol)}</div>
        <div className="text-muted cart-row-unit">{formatMoney(unitPrice, currencyDecimals, currencySymbol)} ea</div>
      </div>
      <button className="btn btn-icon btn-ghost cart-row-delete" onClick={() => removeLine(line.lineId)} title="Remove">
        🗑
      </button>
    </div>
  )
}
