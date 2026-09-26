import { useEffect, useRef, useState } from 'react'
import { useI18n, type Language } from '../i18n'
import { useAuth } from './AuthContext'
import { useSettings } from './SettingsContext'
import { useShift } from './ShiftContext'
import { setSoundEnabled } from '../services/audio'
import { ShortcutHelpOverlay } from './ShortcutHelp'

export function TopBar({ onLock }: { onLock: () => void }) {
  const { t, language, setLanguage } = useI18n()
  const { user, logout } = useAuth()
  const { settings } = useSettings()
  const { shift } = useShift()
  const [menuOpen, setMenuOpen] = useState(false)
  const [helpOpen, setHelpOpen] = useState(false)
  const [soundOn, setSoundOn] = useState(true)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [])

  const status = !shift ? { label: 'No Shift', cls: 'status-dot-warn' } : { label: t('topbar_ready'), cls: 'status-dot-ok' }

  return (
    <header className="topbar">
      <div className="topbar-left">
        <div className="store-name">{settings['store.name']}</div>
      </div>

      <div className="topbar-center">
        <span className={`status-dot ${status.cls}`} />
        <span className="register-label">{t('topbar_register')} {shift?.register_name ?? ''}</span>
        <span className="text-muted">{status.label}</span>
      </div>

      <div className="topbar-right">
        <label className="sound-toggle" title="Toggle sound">
          <input
            type="checkbox"
            checked={soundOn}
            onChange={(e) => {
              setSoundOn(e.target.checked)
              setSoundEnabled(e.target.checked)
            }}
          />
          {soundOn ? '🔊' : '🔇'}
        </label>

        <select
          className="lang-select"
          value={language}
          onChange={(e) => setLanguage(e.target.value as Language)}
          aria-label="Language"
        >
          <option value="es">Español</option>
          <option value="en">English</option>
          <option value="ar">العربية</option>
        </select>

        <button className="btn btn-icon btn-ghost" title="Keyboard shortcuts" onClick={() => setHelpOpen(true)}>
          ⌨
        </button>

        <div className="cashier-menu" ref={menuRef}>
          <button className="btn btn-ghost" onClick={() => setMenuOpen((o) => !o)}>
            <span className="avatar">{user?.name?.[0]?.toUpperCase()}</span>
            {user?.name}
          </button>
          {menuOpen && (
            <div className="dropdown">
              <div className="dropdown-item text-muted">{t('topbar_my_session')}</div>
              <button className="dropdown-item" onClick={onLock}>
                {t('topbar_lock_register')}
              </button>
              <button className="dropdown-item" onClick={logout}>
                {t('topbar_logout')}
              </button>
            </div>
          )}
        </div>
      </div>
      {helpOpen && <ShortcutHelpOverlay onClose={() => setHelpOpen(false)} />}
    </header>
  )
}
