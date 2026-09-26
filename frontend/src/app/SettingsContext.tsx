import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api } from '../services/api'
import { useAuth } from './AuthContext'

export interface StoreSettings {
  [key: string]: string
}

interface SettingsContextValue {
  settings: StoreSettings
  loading: boolean
  currencySymbol: string
  currencyDecimals: number
  reload: () => Promise<void>
  save: (values: Record<string, string>) => Promise<void>
}

const defaults: StoreSettings = {
  'currency.code': 'EUR',
  'currency.decimals': '2',
  'currency.symbol': '€',
  'store.name': 'My Store',
}

const SettingsContext = createContext<SettingsContextValue | null>(null)

export function SettingsProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [settings, setSettings] = useState<StoreSettings>(defaults)
  const [loading, setLoading] = useState(true)

  const reload = useCallback(async () => {
    try {
      const data = await api.get<StoreSettings>('/api/settings')
      setSettings({ ...defaults, ...data })
    } catch {
      // Keep defaults; Settings screen will surface the error if the user
      // tries to save.
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (user) reload()
    else setLoading(false)
  }, [user, reload])

  const save = useCallback(async (values: Record<string, string>) => {
    await api.put('/api/settings', values)
    setSettings((s) => ({ ...s, ...values }))
  }, [])

  const value = useMemo(
    () => ({
      settings,
      loading,
      currencySymbol: settings['currency.symbol'] ?? '€',
      currencyDecimals: Number(settings['currency.decimals'] ?? '2'),
      reload,
      save,
    }),
    [settings, loading, reload, save],
  )

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>
}

export function useSettings() {
  const ctx = useContext(SettingsContext)
  if (!ctx) throw new Error('useSettings must be used within SettingsProvider')
  return ctx
}
