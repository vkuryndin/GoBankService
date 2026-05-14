import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { RequestStatus } from '../components/RequestStatus'
import { useAuth } from '../hooks/useAuth'

export function ProtectedRoute() {
  const location = useLocation()
  const { isAuthenticated, isAuthChecking, authCheckState } = useAuth()

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
