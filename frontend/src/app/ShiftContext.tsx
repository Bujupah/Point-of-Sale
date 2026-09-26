import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api, ApiError } from '../services/api'
import type { Shift } from '../types'
import { useAuth } from './AuthContext'

interface ShiftContextValue {
  shift: Shift | null
  loading: boolean
  reload: () => Promise<void>
  openShift: (openingFloat: number) => Promise<void>
  closeShift: (countedCash: number, notes: string) => Promise<Shift>
}

const ShiftContext = createContext<ShiftContextValue | null>(null)

export function ShiftProvider({ children }: { children: ReactNode }) {
  const { registerId, user } = useAuth()
  const [shift, setShift] = useState<Shift | null>(null)
  const [loading, setLoading] = useState(true)

  const reload = useCallback(async () => {
    if (!registerId) {
      setShift(null)
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const current = await api.get<Shift>('/api/shifts/current')
      setShift(current)
    } catch (e) {
      if (e instanceof ApiError && e.code === 'NO_OPEN_SHIFT') {
        setShift(null)
      }
    } finally {
      setLoading(false)
    }
  }, [registerId])

  useEffect(() => {
    if (user) reload()
    else setLoading(false)
  }, [user, reload])

  const openShift = useCallback(
    async (openingFloat: number) => {
      const created = await api.post<Shift>('/api/shifts/open', { opening_float: openingFloat })
      setShift(created)
    },
    [],
  )

  const closeShift = useCallback(async (countedCash: number, notes: string) => {
    if (!shift) throw new Error('No open shift')
    const closed = await api.post<Shift>(`/api/shifts/${shift.id}/close`, { counted_cash: countedCash, notes })
    setShift(null)
    return closed
  }, [shift])

  const value = useMemo(() => ({ shift, loading, reload, openShift, closeShift }), [shift, loading, reload, openShift, closeShift])

  return <ShiftContext.Provider value={value}>{children}</ShiftContext.Provider>
}

export function useShift() {
  const ctx = useContext(ShiftContext)
  if (!ctx) throw new Error('useShift must be used within ShiftProvider')
  return ctx
}
