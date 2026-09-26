import { useState } from 'react'
import { CheckCircle2, XCircle, Loader2 } from 'lucide-react'
import { api, ApiError } from '../../services/api'

type Status = { kind: 'idle' } | { kind: 'busy'; message: string } | { kind: 'success'; message: string } | { kind: 'error'; message: string }

export function PrintersTab() {
  const [status, setStatus] = useState<Status>({ kind: 'idle' })

  async function test() {
    setStatus({ kind: 'busy', message: 'Testing...' })
    try {
      await api.post('/api/hardware/printer/test')
      setStatus({ kind: 'success', message: 'Test page sent' })
    } catch (err) {
      setStatus({ kind: 'error', message: err instanceof ApiError ? err.message : 'Printer test failed' })
    }
  }

  async function drawer() {
    setStatus({ kind: 'busy', message: 'Opening drawer...' })
    try {
      await api.post('/api/hardware/drawer/open')
      setStatus({ kind: 'success', message: 'Drawer opened' })
    } catch (err) {
      setStatus({ kind: 'error', message: err instanceof ApiError ? err.message : 'Drawer open failed' })
    }
  }

  return (
    <div className="card" style={{ padding: 24, maxWidth: 480 }}>
      <div className="stack">
        <p className="text-muted">
          The default receipt printer is configured in the database (see the <code>printers</code> table / installer
          configuration). This panel lets you verify it's reachable before a shift starts.
        </p>
        <div className="toolbar">
          <button className="btn" onClick={test}>Test Print</button>
          <button className="btn" onClick={drawer}>Open Drawer</button>
        </div>
        {status.kind !== 'idle' && (
          <div className={`status-message ${status.kind === 'error' ? 'text-danger' : status.kind === 'success' ? 'text-success' : 'text-muted'}`}>
            {status.kind === 'busy' && <Loader2 size={16} className="spin" />}
            {status.kind === 'success' && <CheckCircle2 size={16} />}
            {status.kind === 'error' && <XCircle size={16} />}
            {status.message}
          </div>
        )}
      </div>
    </div>
  )
}
