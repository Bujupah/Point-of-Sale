import { useState } from 'react'
import { api, ApiError } from '../../services/api'

export function PrintersTab() {
  const [status, setStatus] = useState('')

  async function test() {
    setStatus('Testing...')
    try {
      await api.post('/api/hardware/printer/test')
      setStatus('Test page sent ✓')
    } catch (err) {
      setStatus(err instanceof ApiError ? `Error: ${err.message}` : 'Printer test failed')
    }
  }

  async function drawer() {
    setStatus('Opening drawer...')
    try {
      await api.post('/api/hardware/drawer/open')
      setStatus('Drawer opened ✓')
    } catch (err) {
      setStatus(err instanceof ApiError ? `Error: ${err.message}` : 'Drawer open failed')
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
        {status && <div className="text-muted">{status}</div>}
      </div>
    </div>
  )
}
