import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { ShoppingCart, Receipt, Users, Wallet, Package, BarChart3, Settings, PanelLeftClose, PanelLeftOpen } from 'lucide-react'
import { useI18n } from '../i18n'
import { useSell } from './SellContext'

const items = [
  { to: '/sell', icon: ShoppingCart, key: 'nav_sell' as const },
  { to: '/orders', icon: Receipt, key: 'nav_orders' as const },
  { to: '/customers', icon: Users, key: 'nav_customers' as const },
  { to: '/register', icon: Wallet, key: 'nav_register' as const },
  { to: '/products', icon: Package, key: 'nav_products' as const },
  { to: '/reports', icon: BarChart3, key: 'nav_reports' as const },
]

export function Sidebar() {
  const { t } = useI18n()
  const [collapsed, setCollapsed] = useState(false)
  const { heldCount } = useSell()
  const ToggleIcon = collapsed ? PanelLeftOpen : PanelLeftClose

  return (
    <nav className={`sidebar ${collapsed ? 'sidebar-collapsed' : ''}`} aria-label="Main navigation">
      <button className="sidebar-toggle" onClick={() => setCollapsed((c) => !c)} title="Toggle sidebar" aria-label="Toggle sidebar">
        <ToggleIcon size={18} />
      </button>
      <div className="sidebar-items">
        {items.map((item) => (
          <NavLink key={item.to} to={item.to} className={({ isActive }) => `sidebar-item ${isActive ? 'active' : ''}`}>
            <span className="sidebar-icon">
              <item.icon size={19} />
            </span>
            {!collapsed && <span className="sidebar-label">{t(item.key)}</span>}
            {item.to === '/sell' && heldCount > 0 && <span className="sidebar-badge">{heldCount}</span>}
          </NavLink>
        ))}
      </div>
      <div className="sidebar-bottom">
        <NavLink to="/settings" className={({ isActive }) => `sidebar-item ${isActive ? 'active' : ''}`}>
          <span className="sidebar-icon">
            <Settings size={19} />
          </span>
          {!collapsed && <span className="sidebar-label">{t('nav_settings')}</span>}
        </NavLink>
      </div>
    </nav>
  )
}
