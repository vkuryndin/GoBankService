import { Navigate, Route, Routes, useNavigate } from 'react-router-dom'
import { AppLayout } from '../layout/AppLayout'
import { AdminPage } from '../pages/AdminPage'
import { AccountsPage } from '../pages/AccountsPage'
import { AnalyticsPage } from '../pages/AnalyticsPage'
import { AuthPage } from '../pages/AuthPage'
import { CardsPage } from '../pages/CardsPage'
import { CreditsPage } from '../pages/CreditsPage'
import { HealthPage } from '../pages/HealthPage'
import { NotificationsPage } from '../pages/NotificationsPage'
import { RatesPage } from '../pages/RatesPage'
import { RegisterPage } from '../pages/RegisterPage'
import { TransfersPage } from '../pages/TransfersPage'
import { useAuth } from '../hooks/useAuth'
import { useSharedAccount } from '../hooks/useSharedAccount'
import { AdminRoute } from './AdminRoute'
import { ProtectedRoute } from './ProtectedRoute'

function RegisterRoute() {
  const navigate = useNavigate()
  const { setLoginValue } = useAuth()

  return (
    <RegisterPage
      onRegistered={setLoginValue}
      onOpenAuth={() => navigate('/auth')}
    />
  )
}

function AdminPageRoute() {
  const { token, currentUser } = useAuth()
  const { sharedAccountId } = useSharedAccount()

  return (
    <AdminPage
      token={token}
      currentUser={currentUser}
      sharedAccountId={sharedAccountId}
    />
  )
}

function AccountsRoute() {
  const { token } = useAuth()
  const { sharedAccountId, setSharedAccountId } = useSharedAccount()

  return (
    <AccountsPage
      token={token}
      sharedAccountId={sharedAccountId}
      onSharedAccountIdChange={setSharedAccountId}
    />
  )
}

function CardsRoute() {
  const { token } = useAuth()
  const { sharedAccountId } = useSharedAccount()

  return <CardsPage token={token} sharedAccountId={sharedAccountId} />
}

function TransfersRoute() {
  const { token } = useAuth()
  const { sharedAccountId, setSharedAccountId } = useSharedAccount()

  return (
    <TransfersPage
      token={token}
      sharedAccountId={sharedAccountId}
      onSharedAccountIdChange={setSharedAccountId}
    />
  )
}

function CreditsRoute() {
  const { token } = useAuth()
  const { sharedAccountId, setSharedAccountId } = useSharedAccount()

  return (
    <CreditsPage
      token={token}
      sharedAccountId={sharedAccountId}
      onSharedAccountIdChange={setSharedAccountId}
    />
  )
}

function AnalyticsRoute() {
  const { token } = useAuth()

  return <AnalyticsPage token={token} />
}

function NotificationsRoute() {
  const { token } = useAuth()

  return <NotificationsPage token={token} />
}

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<Navigate to="/health" replace />} />
        <Route path="health" element={<HealthPage />} />
        <Route path="register" element={<RegisterRoute />} />
        <Route path="auth" element={<AuthPage />} />

        <Route element={<ProtectedRoute />}>
          <Route path="accounts" element={<AccountsRoute />} />
          <Route path="cards" element={<CardsRoute />} />
          <Route path="transfers" element={<TransfersRoute />} />
          <Route path="credits" element={<CreditsRoute />} />
          <Route path="analytics" element={<AnalyticsRoute />} />
          <Route path="rates" element={<RatesPage />} />
          <Route path="notifications" element={<NotificationsRoute />} />
        </Route>

        <Route element={<AdminRoute />}>
          <Route path="admin" element={<AdminPageRoute />} />
        </Route>

        <Route path="*" element={<Navigate to="/health" replace />} />
      </Route>
    </Routes>
  )
}
