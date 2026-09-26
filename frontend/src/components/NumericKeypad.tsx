// Shared numeric keypad (brief §25's cash keypad, also reused for PIN entry
// and any other numeric input) — one component, several call sites.
import { Delete } from 'lucide-react'

interface Props {
  onDigit: (digit: string) => void
  onBackspace: () => void
  onClear?: () => void
  allowDecimal?: boolean
}

const layout = ['1', '2', '3', '4', '5', '6', '7', '8', '9']

export function NumericKeypad({ onDigit, onBackspace, onClear, allowDecimal = true }: Props) {
  return (
    <div className="numeric-keypad">
      {layout.map((d) => (
        <button key={d} type="button" className="btn keypad-key" onClick={() => onDigit(d)}>
          {d}
        </button>
      ))}
      {allowDecimal ? (
        <button type="button" className="btn keypad-key" onClick={() => onDigit('.')}>
          .
        </button>
      ) : (
        <button type="button" className="btn keypad-key" onClick={onClear}>
          C
        </button>
      )}
      <button type="button" className="btn keypad-key" onClick={() => onDigit('0')}>
        0
      </button>
      <button type="button" className="btn keypad-key" onClick={onBackspace} aria-label="Backspace">
        <Delete size={20} />
      </button>
    </div>
  )
}
