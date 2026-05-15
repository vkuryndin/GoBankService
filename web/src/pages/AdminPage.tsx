import { useEffect, useState } from 'react'
import { RequestStatus } from '../components/RequestStatus'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { EmptyState } from '../components/ui/EmptyState'
import { AdminAccountStatusPanel } from '../features/admin/AdminAccountStatusPanel'
import { AdminSessionsTable } from '../features/admin/AdminSessionsTable'
import { AdminStatisticsPanel } from '../features/admin/AdminStatisticsPanel'
import { AdminUsersTable } from '../features/admin/AdminUsersTable'
import { useAdmin } from '../hooks/useAdmin'
import { useToast } from '../hooks/useToast'
import type { AdminAccountStatusResponse } from '../types/admin'
import type { CurrentUser } from '../types/auth'
import { emptyState, type RequestState } from '../types/common'
import { validatePositiveInteger } from '../utils/validation'

type AdminPageProps = {
  token: string
  currentUser: CurrentUser | null
  sharedAccountId: string
}

function queryState(
  loading: boolean,
  error: unknown,
  success: string,
): RequestState {
  return {
    loading,
    error: error instanceof Error ? error.message : '',
    success: !loading && !error ? success : '',
  }
}

export function AdminPage({ token, currentUser, sharedAccountId }: AdminPageProps) {
  const isAdmin = Boolean(currentUser?.is_admin)
  const {
    users,
    sessions,
    statistics,
    usersQuery,
    sessionsQuery,
    statisticsQuery,
    blockMutation,
    unblockMutation,
  } = useAdmin(
    token,
    isAdmin && token.trim() !== '',
  )
  const { showToast } = useToast()
  const [adminAccountState, setAdminAccountState] = useState<RequestState>(emptyState)
  const [adminAccountId, setAdminAccountId] = useState(sharedAccountId)
  const [adminAccountResult, setAdminAccountResult] =
    useState<AdminAccountStatusResponse | null>(null)

  useEffect(() => {
    if (sharedAccountId) {
      setAdminAccountId(sharedAccountId)
    }
  }, [sharedAccountId])

  const requireAdmin = (): boolean => {
    if (!token) {
      setAdminAccountState({ loading: false, error: 'Сначала нужно войти в систему.', success: '' })
      return false
    }

    if (isAdmin) {
      return true
    }

    setAdminAccountState({ loading: false, error: 'Доступ разрешен только администратору.', success: '' })
    return false
  }


  const loadAdminStatistics = async () => {
    if (!isAdmin || !token) {
      return
    }

    await statisticsQuery.refetch()
  }

  const loadAdminUsers = async () => {
    if (!isAdmin || !token) {
      return
    }

    await usersQuery.refetch()
  }

  const loadAdminSessions = async () => {
    if (!isAdmin || !token) {
      return
    }

    await sessionsQuery.refetch()
  }

  const changeAdminAccountStatus = async (action: 'block' | 'unblock') => {
    if (!requireAdmin()) {
      return
    }

    const validationError = validatePositiveInteger(adminAccountId, 'Account ID')
    if (validationError) {
      setAdminAccountState({ loading: false, error: validationError, success: '' })
      return
    }

    const accountID = Number(adminAccountId)
    const mutation = action === 'block' ? blockMutation : unblockMutation

    setAdminAccountState({ loading: true, error: '', success: '' })
    setAdminAccountResult(null)

    try {
      const data = await mutation.mutateAsync(accountID)
      const successMessage = action === 'block' ? 'Счет заблокирован.' : 'Счет разблокирован.'

      setAdminAccountResult(data)
      setAdminAccountState({ loading: false, error: '', success: successMessage })
      showToast(successMessage, 'success')
    } catch (error) {
      const message = error instanceof Error ? error.message : `Failed to ${action} account`
      setAdminAccountState({ loading: false, error: message, success: '' })
      showToast(message, 'error')
    }
  }

  const usersState = queryState(usersQuery.isFetching, usersQuery.error, users.length > 0 ? 'Список пользователей загружен.' : '')
  const sessionsState = queryState(
    sessionsQuery.isFetching,
    sessionsQuery.error,
    sessions.length > 0 ? 'Список активных сессий загружен.' : '',
  )
  const statisticsState = queryState(
    statisticsQuery.isFetching,
    statisticsQuery.error,
    statistics ? 'Системная статистика загружена.' : '',
  )

  return (
    <Card variant="plain" className="panel">
      <div className="panelHeader">
        <div>
          <h2>Администрирование</h2>
          <p>
            Admin-действия и системная статистика: пользователи, сессии, счета, карты, кредиты, операции и audit events.
          </p>
        </div>

        <div className="actions">
          <Button type="button" onClick={() => void loadAdminUsers()} disabled={usersQuery.isFetching || !token || !isAdmin}>
            {usersQuery.isFetching ? 'Загружаю...' : 'Загрузить пользователей'}
          </Button>
          <Button className="secondary" type="button" onClick={() => void loadAdminSessions()} disabled={sessionsQuery.isFetching || !token || !isAdmin}>
            {sessionsQuery.isFetching ? 'Загружаю...' : 'Активные сессии'}
          </Button>
          <Button className="secondary" type="button" onClick={() => void loadAdminStatistics()} disabled={statisticsQuery.isFetching || !token || !isAdmin}>
            {statisticsQuery.isFetching ? 'Загружаю...' : 'Системная статистика'}
          </Button>
        </div>
      </div>

      {!isAdmin && (
        <EmptyState>
          Этот раздел доступен только администратору. Войдите под admin-пользователем.
        </EmptyState>
      )}

      <div className="adminGrid">
        <section className="subPanel">
          <div className="subPanelHeader">
            <h3>Пользователи</h3>
            <span>{users.length}</span>
          </div>
          <RequestStatus state={usersState} />
          <AdminUsersTable users={users} />
        </section>

        <section className="subPanel">
          <div className="subPanelHeader">
            <h3>Активные сессии</h3>
            <span>{sessions.length}</span>
          </div>
          <RequestStatus state={sessionsState} />
          <AdminSessionsTable sessions={sessions} />
        </section>

        <section className="subPanel adminStatisticsPanel">
          <div className="subPanelHeader">
            <div>
              <h3>Системная статистика</h3>
              <p className="mutedText">Агрегаты по пользователям, счетам, картам, кредитам, операциям и audit events.</p>
            </div>
          </div>
          <RequestStatus state={statisticsState} />
          <AdminStatisticsPanel statistics={statistics} />
        </section>

        <AdminAccountStatusPanel
          accountId={adminAccountId}
          state={adminAccountState}
          result={adminAccountResult}
          disabled={!token || !isAdmin}
          onAccountIdChange={setAdminAccountId}
          onBlock={() => void changeAdminAccountStatus('block')}
          onUnblock={() => void changeAdminAccountStatus('unblock')}
        />
      </div>
    </Card>
  )
}
