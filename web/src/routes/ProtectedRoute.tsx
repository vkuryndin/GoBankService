import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { RequestStatus } from '../components/RequestStatus'
import { useAuth } from '../hooks/useAuth'

export function ProtectedRoute() {
  const location = useLocation()
  const { isAuthenticated, authCheckState } = useAuth()

  if (!isAuthenticated) {
    return <Navigate to="/auth" state={{ from: location }} replace />
  }

  if (authCheckState.loading) {
    return (
      <section className="panel">
        <RequestStatus state={authCheckState} />
      </section>
    )
  }

  return <Outlet />
}
