const shortcuts: [string, string][] = [
  ['F1', 'Search'],
  ['F2', 'Focus scanner/search'],
  ['F3', 'Discount'],
  ['F4', 'Customer'],
  ['F5', 'Hold sale'],
  ['F6', 'Held sales'],
  ['F7', 'Delete line'],
  ['F8', 'Payment'],
  ['F9', 'Cash'],
  ['F10', 'Card'],
  ['F11', 'Drawer'],
  ['F12', 'Complete sale'],
  ['Esc', 'Back / close'],
  ['Enter', 'Confirm'],
  ['+ / -', 'Increase / decrease quantity'],
]

export function ShortcutHelpOverlay({ onClose }: { onClose: () => void }) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth: 420 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Keyboard Shortcuts</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose}>✕</button>
        </div>
        <div className="modal-body">
          <table className="table">
            <tbody>
              {shortcuts.map(([key, desc]) => (
                <tr key={key}>
                  <td style={{ width: 90 }}><kbd>{key}</kbd></td>
                  <td>{desc}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
