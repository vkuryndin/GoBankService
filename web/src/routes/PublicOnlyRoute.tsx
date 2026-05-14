import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'

export function PublicOnlyRoute() {
  const { hasToken, isAuthenticated, isAuthChecking } = useAuth()

  if (hasToken && isAuthChecking) {
    return (
      <div className="panel">
        <div className="loadingNotice">Проверяю текущую сессию...</div>
      </div>
    )
  }

  if (isAuthenticated) {
    return <Navigate to="/health" replace />
  }

  return <Outlet />
}
