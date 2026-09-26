import { useEffect, useState } from 'react'
import { api, qs } from '../../services/api'
import type { Customer } from '../../types'

export function CustomerSelectorModal({ onSelect, onClose }: { onSelect: (c: Customer) => void; onClose: () => void }) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Customer[]>([])
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [newPhone, setNewPhone] = useState('')

  useEffect(() => {
    const handle = setTimeout(() => {
      api.get<Customer[]>(`/api/customers${qs({ q: query, limit: 20 })}`).then((r) => setResults(r ?? [])).catch(() => setResults([]))
    }, 150)
    return () => clearTimeout(handle)
  }, [query])

  async function createCustomer() {
    const c = await api.post<Customer>('/api/customers', { name: newName, phone: newPhone })
    onSelect(c)
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ maxWidth: 420 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Add Customer</h2>
          <button className="btn btn-icon btn-ghost" onClick={onClose}>✕</button>
        </div>
        <div className="modal-body">
          {creating ? (
            <div className="stack">
              <label className="field">
                Name
                <input className="input" value={newName} onChange={(e) => setNewName(e.target.value)} autoFocus />
              </label>
              <label className="field">
                Phone
                <input className="input" value={newPhone} onChange={(e) => setNewPhone(e.target.value)} />
              </label>
              <div className="toolbar">
                <button className="btn" onClick={() => setCreating(false)}>Back</button>
                <button className="btn btn-primary" disabled={!newName} onClick={createCustomer}>Create & Select</button>
              </div>
            </div>
          ) : (
            <>
              <input className="input" placeholder="Search name, phone, email..." value={query} onChange={(e) => setQuery(e.target.value)} autoFocus />
              <div className="customer-results">
                {results.map((c) => (
                  <button key={c.id} className="customer-result-row" onClick={() => onSelect(c)}>
                    <span className="avatar">{c.name[0]?.toUpperCase()}</span>
                    <span>
                      <div>{c.name}</div>
                      <div className="text-muted">{c.phone || c.email}</div>
                    </span>
                    <span className="text-muted">{c.loyalty_points} pts</span>
                  </button>
                ))}
                {results.length === 0 && <div className="empty-state">No customers found.</div>}
              </div>
              <button className="btn btn-block" onClick={() => { setCreating(true); setNewName(query) }}>+ New Customer</button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
