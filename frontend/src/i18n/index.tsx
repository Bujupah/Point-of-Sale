import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import en, { type TranslationKey } from './en'
import es from './es'
import ar from './ar'

export type Language = 'es' | 'en' | 'ar'

const dictionaries: Record<Language, Record<TranslationKey, string>> = { en, es, ar }
export const RTL_LANGUAGES: Language[] = ['ar']

interface I18nContextValue {
  language: Language
  setLanguage: (lang: Language) => void
  t: (key: TranslationKey) => string
  dir: 'ltr' | 'rtl'
}

const I18nContext = createContext<I18nContextValue | null>(null)

const STORAGE_KEY = 'pos.language'

export function I18nProvider({ children, initialLanguage }: { children: ReactNode; initialLanguage?: Language }) {
  const [language, setLanguageState] = useState<Language>(() => {
    if (initialLanguage) return initialLanguage
    try {
      const stored = localStorage.getItem(STORAGE_KEY) as Language | null
      if (stored && dictionaries[stored]) return stored
    } catch {
      /* ignore */
    }
    return 'es'
  })

  const dir: 'ltr' | 'rtl' = RTL_LANGUAGES.includes(language) ? 'rtl' : 'ltr'

  useEffect(() => {
    document.documentElement.dir = dir
    document.documentElement.lang = language
  }, [dir, language])

  const setLanguage = useCallback((lang: Language) => {
    setLanguageState(lang)
    try {
      localStorage.setItem(STORAGE_KEY, lang)
    } catch {
      /* ignore */
    }
  }, [])

  const t = useCallback((key: TranslationKey) => dictionaries[language][key] ?? dictionaries.en[key] ?? key, [language])

  const value = useMemo(() => ({ language, setLanguage, t, dir }), [language, setLanguage, t, dir])

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n() {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n must be used within I18nProvider')
  return ctx
}
