import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { useI18n } from '../i18n'
import { useSell } from './SellContext'

const items = [
  { to: '/sell', icon: '🛒', key: 'nav_sell' as const },
  { to: '/orders', icon: '🧾', key: 'nav_orders' as const },
  { to: '/customers', icon: '👤', key: 'nav_customers' as const },
  { to: '/register', icon: '💰', key: 'nav_register' as const },
  { to: '/products', icon: '📦', key: 'nav_products' as const },
  { to: '/reports', icon: '📊', key: 'nav_reports' as const },
]

export function Sidebar() {
  const { t } = useI18n()
  const [collapsed, setCollapsed] = useState(false)
  const { heldCount } = useSell()

  return (
    <nav className={`sidebar ${collapsed ? 'sidebar-collapsed' : ''}`} aria-label="Main navigation">
      <button className="sidebar-toggle" onClick={() => setCollapsed((c) => !c)} title="Toggle sidebar" aria-label="Toggle sidebar">
        {collapsed ? '»' : '«'}
      </button>
      <div className="sidebar-items">
        {items.map((item) => (
          <NavLink key={item.to} to={item.to} className={({ isActive }) => `sidebar-item ${isActive ? 'active' : ''}`}>
            <span className="sidebar-icon">{item.icon}</span>
            {!collapsed && <span className="sidebar-label">{t(item.key)}</span>}
            {item.to === '/sell' && heldCount > 0 && <span className="sidebar-badge">{heldCount}</span>}
          </NavLink>
        ))}
      </div>
      <div className="sidebar-bottom">
        <NavLink to="/settings" className={({ isActive }) => `sidebar-item ${isActive ? 'active' : ''}`}>
          <span className="sidebar-icon">⚙️</span>
          {!collapsed && <span className="sidebar-label">{t('nav_settings')}</span>}
        </NavLink>
      </div>
    </nav>
  )
}
