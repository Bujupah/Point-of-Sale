import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api, getToken, setToken } from '../services/api'
import type { AuthResult, User } from '../types'

interface AuthState {
  loading: boolean
  user: User | null
  permissions: string[]
  locked: boolean
  registerId: number | null
}

interface AuthContextValue extends AuthState {
  login: (username: string, pin: string, registerId?: number) => Promise<void>
  lock: () => Promise<void>
  unlock: (pin: string) => Promise<void>
  switchCashier: (username: string, pin: string) => Promise<void>
  logout: () => Promise<void>
  hasPermission: (code: string) => boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

function applyAuthResult(result: AuthResult) {
  setToken(result.token)
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ loading: true, user: null, permissions: [], locked: false, registerId: null })

  const refreshMe = useCallback(async () => {
    if (!getToken()) {
      setState({ loading: false, user: null, permissions: [], locked: false, registerId: null })
      return
    }
    try {
      const me = await api.get<{ user: User; permissions: string[]; locked: boolean; register_id?: number }>('/api/auth/me')
      setState({ loading: false, user: me.user, permissions: me.permissions, locked: me.locked, registerId: me.register_id ?? null })
    } catch {
      setToken(null)
      setState({ loading: false, user: null, permissions: [], locked: false, registerId: null })
    }
  }, [])

  useEffect(() => {
    refreshMe()
  }, [refreshMe])

  const login = useCallback(async (username: string, pin: string, registerId?: number) => {
    const result = await api.post<AuthResult>('/api/auth/login', { username, pin, register_id: registerId })
    applyAuthResult(result)
    setState({ loading: false, user: result.user, permissions: result.permissions, locked: result.locked, registerId: result.register_id ?? null })
  }, [])

  const lock = useCallback(async () => {
    await api.post('/api/auth/lock')
    setState((s) => ({ ...s, locked: true }))
  }, [])

  const unlock = useCallback(async (pin: string) => {
    const result = await api.post<AuthResult>('/api/auth/unlock', { pin })
    setState((s) => ({ ...s, locked: false, user: result.user, permissions: result.permissions }))
  }, [])

  const switchCashier = useCallback(async (username: string, pin: string) => {
    const result = await api.post<AuthResult>('/api/auth/switch', { username, pin })
    applyAuthResult(result)
    setState({ loading: false, user: result.user, permissions: result.permissions, locked: result.locked, registerId: result.register_id ?? null })
  }, [])

  const logout = useCallback(async () => {
    try {
      await api.post('/api/auth/logout')
    } finally {
      setToken(null)
      setState({ loading: false, user: null, permissions: [], locked: false, registerId: null })
    }
  }, [])

  const hasPermission = useCallback((code: string) => state.permissions.includes(code), [state.permissions])

  const value = useMemo(
    () => ({ ...state, login, lock, unlock, switchCashier, logout, hasPermission }),
    [state, login, lock, unlock, switchCashier, logout, hasPermission],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
