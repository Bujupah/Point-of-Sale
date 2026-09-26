import { useEffect, useState } from 'react'
import { api, ApiError } from '../../services/api'
import type { User } from '../../types'

interface Role {
  id: number
  name: string
}

export function UsersTab() {
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [creating, setCreating] = useState(false)
  const [name, setName] = useState('')
  const [username, setUsername] = useState('')
  const [pin, setPin] = useState('')
  const [roleId, setRoleId] = useState<number | ''>('')
  const [error, setError] = useState('')

  function load() {
    api.get<User[]>('/api/users').then((r) => setUsers(r ?? [])).catch(() => {})
    api.get<Role[]>('/api/roles').then((res) => {
      const r = res ?? []
      setRoles(r)
      if (r.length && !roleId) setRoleId(r[r.length - 1].id)
    }).catch(() => {})
  }

  useEffect(load, [])

  async function createUser() {
    setError('')
    try {
      await api.post('/api/users', { name, username, pin, role_id: roleId })
      setCreating(false)
      setName('')
      setUsername('')
      setPin('')
      load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed')
    }
  }

  async function toggleActive(u: User) {
    await api.put(`/api/users/${u.id}`, { name: u.name, role_id: u.role_id, active: !u.active })
    load()
  }

  return (
    <div className="stack" style={{ maxWidth: 600 }}>
      <table className="table">
        <thead><tr><th>Name</th><th>Username</th><th>Role</th><th>Status</th><th></th></tr></thead>
        <tbody>
          {users.map((u) => (
            <tr key={u.id}>
              <td>{u.name}</td>
              <td>{u.username}</td>
              <td>{u.role_name}</td>
              <td>{u.active ? 'Active' : 'Disabled'}</td>
              <td><button className="btn" onClick={() => toggleActive(u)}>{u.active ? 'Disable' : 'Enable'}</button></td>
            </tr>
          ))}
        </tbody>
      </table>

      {creating ? (
        <div className="card" style={{ padding: 16 }}>
          <div className="stack">
            <label className="field">Name<input className="input" value={name} onChange={(e) => setName(e.target.value)} /></label>
            <label className="field">Username<input className="input" value={username} onChange={(e) => setUsername(e.target.value)} /></label>
            <label className="field">PIN<input className="input" type="password" value={pin} onChange={(e) => setPin(e.target.value)} /></label>
            <label className="field">Role
              <select className="input" value={roleId} onChange={(e) => setRoleId(Number(e.target.value))}>
                {roles.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
              </select>
            </label>
            {error && <div className="auth-error">{error}</div>}
            <div className="toolbar">
              <button className="btn" onClick={() => setCreating(false)}>Cancel</button>
              <button className="btn btn-primary" onClick={createUser} disabled={!name || !username || pin.length < 4}>Create</button>
            </div>
          </div>
        </div>
      ) : (
        <button className="btn btn-primary" onClick={() => setCreating(true)}>+ Add User</button>
      )}
    </div>
  )
}
