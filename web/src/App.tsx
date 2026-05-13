import { useEffect, useState } from 'react'
import type { CSSProperties } from 'react'
import './App.css'

import { apiRequest } from './api/http'
import { getPageTitle, menuItems } from './config/menu'
import { RequestMessage } from './components/RequestMessage'
import { Sidebar } from './components/Sidebar'
import { Topbar } from './components/Topbar'
import { AdminPage } from './pages/AdminPage'
import { AccountsPage } from './pages/AccountsPage'
import { AuthPage } from './pages/AuthPage'
import { CardsPage } from './pages/CardsPage'
import { CreditsPage } from './pages/CreditsPage'
import { HealthPage } from './pages/HealthPage'
import { PlaceholderPage } from './pages/PlaceholderPage'
import { RegisterPage } from './pages/RegisterPage'
import { TransfersPage } from './pages/TransfersPage'
import type { AuthCheckResponse, CurrentUser } from './types/auth'
import { emptyState, type MenuKey, type RequestState } from './types/common'

const tokenStorageKey = 'bank_service_token'
const collapsedSidebarWidth = 88

function App() {
  const [activeMenu, setActiveMenu] = useState<MenuKey>('health')
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [sidebarWidth, setSidebarWidth] = useState(280)

  const [token, setToken] = useState(() => localStorage.getItem(tokenStorageKey) || '')
  const [currentUser, setCurrentUser] = useState<CurrentUser | null>(null)
  const [authCheckState, setAuthCheckState] = useState<RequestState>(emptyState)
  const [logoutState, setLogoutState] = useState<RequestState>(emptyState)

  const [loginValue, setLoginValue] = useState('test@example.com')
  const [sharedAccountId, setSharedAccountId] = useState('')

  const isAuthenticated = token.trim() !== ''

  useEffect(() => {
    if (token) {
      localStorage.setItem(tokenStorageKey, token)
      void checkCurrentUser(token)
      return
    }

    localStorage.removeItem(tokenStorageKey)
    setCurrentUser(null)
  }, [token])

  const resetUserData = () => {
    setToken('')
    setCurrentUser(null)
    setSharedAccountId('')
    setAuthCheckState(emptyState)
  }

  const checkCurrentUser = async (tokenToCheck = token) => {
    if (!tokenToCheck) {
      setAuthCheckState({
        loading: false,
        error: 'Токен отсутствует.',
        success: '',
      })
      setCurrentUser(null)
      return
    }

    setAuthCheckState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const data = await apiRequest<AuthCheckResponse>('/api/auth/check', {
        token: tokenToCheck,
      })

      setCurrentUser({
        authenticated: data.authenticated,
        user_id: data.user_id,
        email: data.email,
        username: data.username,
        is_admin: data.is_admin,
      })

      setAuthCheckState({
        loading: false,
        error: '',
        success: `Вы вошли как ${data.email}. Роль: ${
          data.is_admin ? 'администратор' : 'пользователь'
        }.`,
      })
    } catch (error) {
      setToken('')
      setCurrentUser(null)
      setAuthCheckState({
        loading: false,
        error:
          error instanceof Error
            ? error.message
            : 'Failed to check current user',
        success: '',
      })
    }
  }

  const handleLogout = async () => {
    if (!token) {
      resetUserData()
      return
    }

    setLogoutState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      await apiRequest<{ message: string }>('/api/logout', {
        method: 'POST',
        token,
      })

      resetUserData()
      setLogoutState({
        loading: false,
        error: '',
        success: 'Вы вышли из системы.',
      })
    } catch (error) {
      setLogoutState({
        loading: false,
        error: error instanceof Error ? error.message : 'Logout failed',
        success: '',
      })
    }
  }

  const appStyle = {
    '--sidebar-width': sidebarCollapsed
      ? `${collapsedSidebarWidth}px`
      : `${sidebarWidth}px`,
  } as CSSProperties

  const activeItem = menuItems.find((item) => item.key === activeMenu)
  const activeTitle = getPageTitle(activeMenu)

  return (
    <main
      className={sidebarCollapsed ? 'app sidebarCollapsed' : 'app'}
      style={appStyle}
    >
      <Sidebar
        activeMenu={activeMenu}
        collapsed={sidebarCollapsed}
        width={sidebarWidth}
        onMenuChange={setActiveMenu}
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
            <RequestMessage state={logoutState} />
          </div>
        )}

        {activeMenu === 'health' && <HealthPage />}

        {activeMenu === 'register' && (
          <RegisterPage
            onRegistered={setLoginValue}
            onOpenAuth={() => setActiveMenu('auth')}
          />
        )}

        {activeMenu === 'auth' && (
          <AuthPage
            token={token}
            currentUser={currentUser}
            authCheckState={authCheckState}
            initialLogin={loginValue}
            onLoginValueChange={setLoginValue}
            onLoginSuccess={(newToken) => {
              setLogoutState(emptyState)
              setToken(newToken)
            }}
            onLoginFailure={resetUserData}
            onCheckCurrentUser={() => void checkCurrentUser()}
          />
        )}

        {activeMenu === 'admin' && (
          <AdminPage
            token={token}
            currentUser={currentUser}
            sharedAccountId={sharedAccountId}
          />
        )}

        {activeMenu === 'accounts' && (
          <AccountsPage
            token={token}
            sharedAccountId={sharedAccountId}
            onSharedAccountIdChange={setSharedAccountId}
          />
        )}

        {activeMenu === 'cards' && (
          <CardsPage token={token} sharedAccountId={sharedAccountId} />
        )}

        {activeMenu === 'transfers' && (
          <TransfersPage
            token={token}
            sharedAccountId={sharedAccountId}
            onSharedAccountIdChange={setSharedAccountId}
          />
        )}

        {activeMenu === 'credits' && (
          <CreditsPage
            token={token}
            sharedAccountId={sharedAccountId}
            onSharedAccountIdChange={setSharedAccountId}
          />
        )}

        {activeItem && !activeItem.implemented && (
          <PlaceholderPage activeMenu={activeMenu} />
        )}
      </section>
    </main>
  )
}

export default App
