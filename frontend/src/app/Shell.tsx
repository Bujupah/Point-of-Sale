import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'
import { useAuth } from './AuthContext'

export function Shell() {
  const { lock } = useAuth()
  return (
    <div className="shell">
      <Sidebar />
      <div className="main-area">
        <TopBar onLock={() => lock()} />
        <div className="content-area">
          <Outlet />
        </div>
      </div>
    </div>
  )
}
