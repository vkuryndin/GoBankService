import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { RequestStatus } from '../components/RequestStatus'
import { useAuth } from '../hooks/useAuth'

export function ProtectedRoute() {
  const location = useLocation()
  const { hasToken, isAuthenticated, isAuthChecking, authCheckState } = useAuth()

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

  return <Outlet />
}
