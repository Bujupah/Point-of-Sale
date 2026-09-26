import { useState } from 'react'
import { X } from 'lucide-react'
import { api, ApiError } from '../../services/api'
import type { Customer } from '../../types'

export function CustomerEditModal({ customer, onClose, onSaved }: { customer?: Customer; onClose: () => void; onSaved: (c: Customer) => void }) {
  const [name, setName] = useState(customer?.name ?? '')
  const [phone, setPhone] = useState(customer?.phone ?? '')
  const [email, setEmail] = useState(customer?.email ?? '')
  const [notes, setNotes] = useState(customer?.notes ?? '')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function save() {
    setBusy(true)
    setError('')
    try {
      const payload = { name, phone, email, notes }
      const saved = customer ? await api.put<Customer>(`/api/customers/${customer.id}`, payload) : await api.post<Customer>('/api/customers', payload)
      onSaved(saved)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Save failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth: 400 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>{customer ? 'Edit Customer' : 'Add Customer'}</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose}><X size={18} /></button>
        </div>
        <div className="modal-body stack">
          <label className="field">Name<input className="input" value={name} onChange={(e) => setName(e.target.value)} autoFocus /></label>
          <label className="field">Phone<input className="input" value={phone} onChange={(e) => setPhone(e.target.value)} /></label>
          <label className="field">Email<input className="input" value={email} onChange={(e) => setEmail(e.target.value)} /></label>
          <label className="field">Notes<textarea className="input" value={notes} onChange={(e) => setNotes(e.target.value)} /></label>
          {error && <div className="auth-error">{error}</div>}
        </div>
        <div className="modal-footer">
          <button className="btn" onClick={onClose}>Cancel</button>
          <button className="btn btn-primary" disabled={busy || !name} onClick={save}>Save</button>
        </div>
      </div>
    </div>
  )
}
