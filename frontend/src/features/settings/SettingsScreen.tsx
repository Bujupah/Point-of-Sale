import { useState } from 'react'
import { useI18n } from '../../i18n'
import { useAuth } from '../../app/AuthContext'
import { StoreSettingsTab } from './StoreSettingsTab'
import { UsersTab } from './UsersTab'
import { PrintersTab } from './PrintersTab'

type Tab = 'store' | 'users' | 'printers'

export function SettingsScreen() {
  const { t } = useI18n()
  const { hasPermission } = useAuth()
  const [tab, setTab] = useState<Tab>('store')

  return (
    <div className="page">
      <div className="page-header">
        <h1>{t('nav_settings')}</h1>
      </div>
      <div className="category-bar">
        <button className={`chip ${tab === 'store' ? 'chip-active' : ''}`} onClick={() => setTab('store')}>{t('settings_store')}</button>
        {hasPermission('users.manage') && (
          <button className={`chip ${tab === 'users' ? 'chip-active' : ''}`} onClick={() => setTab('users')}>{t('settings_users')}</button>
        )}
        <button className={`chip ${tab === 'printers' ? 'chip-active' : ''}`} onClick={() => setTab('printers')}>{t('settings_printers')}</button>
      </div>
      <div style={{ marginTop: 16 }}>
        {tab === 'store' && <StoreSettingsTab />}
        {tab === 'users' && hasPermission('users.manage') && <UsersTab />}
        {tab === 'printers' && <PrintersTab />}
      </div>
    </div>
  )
}
