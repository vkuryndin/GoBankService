import { useState } from 'react'
import type { CSSProperties } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { getMenuKeyByPath, getPageTitle } from '../config/menu'
import { RequestStatus } from '../components/RequestStatus'
import { Sidebar } from '../components/Sidebar'
import { Topbar } from '../components/Topbar'
import { ConfirmDialog } from '../components/ui/ConfirmDialog'
import { useAuth } from '../hooks/useAuth'
import { useSharedAccount } from '../hooks/useSharedAccount'

const collapsedSidebarWidth = 88

export function AppLayout() {
  const location = useLocation()
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [logoutConfirmOpen, setLogoutConfirmOpen] = useState(false)
  const [sidebarWidth, setSidebarWidth] = useState(280)
  const { currentUser, isAuthenticated, authCheckState, logoutState, logout } = useAuth()
  const { clearSharedAccountId } = useSharedAccount()

  const activeMenu = getMenuKeyByPath(location.pathname)
  const activeTitle = getPageTitle(activeMenu)

  const appStyle = {
    '--sidebar-width': sidebarCollapsed
      ? `${collapsedSidebarWidth}px`
      : `${sidebarWidth}px`,
  } as CSSProperties

  const handleLogout = () => {
    setLogoutConfirmOpen(true)
  }

  const confirmLogout = async () => {
    clearSharedAccountId()
    await logout()
    setLogoutConfirmOpen(false)
  }

  return (
    <main
      className={sidebarCollapsed ? 'app sidebarCollapsed' : 'app'}
      style={appStyle}
    >
      <Sidebar
        activeMenu={activeMenu}
        collapsed={sidebarCollapsed}
        width={sidebarWidth}
        onCollapsedChange={setSidebarCollapsed}
        onWidthChange={setSidebarWidth}
      />

      <section className="content">
        <Topbar
          title={activeTitle}
          currentUser={currentUser}
          isAuthenticated={isAuthenticated}
          authCheckState={authCheckState}
          logoutLoading={logoutState.loading}
          onLogout={handleLogout}
        />

        {(logoutState.error || logoutState.success) && (
          <div className="panel slimPanel">
            <RequestStatus state={logoutState} />
          </div>
        )}

        <Outlet />

        <ConfirmDialog
          open={logoutConfirmOpen}
          title="Выйти из системы"
          message="Завершить текущую сессию?"
          confirmText="Выйти"
          danger
          loading={logoutState.loading}
          onConfirm={() => void confirmLogout()}
          onCancel={() => setLogoutConfirmOpen(false)}
        />
      </section>
    </main>
  )
}
