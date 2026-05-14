import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { RequestStatus } from '../components/RequestStatus'
import { useAuth } from '../hooks/useAuth'

export function AdminRoute() {
  const location = useLocation()
  const { hasToken, isAuthenticated, isAuthChecking, currentUser, authCheckState } = useAuth()

  if (!hasToken) {
    return <Navigate to="/auth" state={{ from: location }} replace />
  }

  if (isAuthChecking) {
    return (
      <section className="panel">
        <RequestStatus state={authCheckState} />
      </section>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/auth" state={{ from: location }} replace />
  }

  if (!currentUser?.is_admin) {
    return (
      <section className="panel">
        <h2>Нет доступа</h2>
        <div className="empty">
          Этот раздел доступен только администратору. Войдите под admin-пользователем.
        </div>
      </section>
    )
  }

  return <Outlet />
}
