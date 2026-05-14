import { useEffect, useState } from 'react'
import { adminApi } from '../api/adminApi'
import { RequestStatus } from '../components/RequestStatus'
import type { AdminAccountStatusResponse, AdminSession, AdminUser } from '../types/admin'
import type { CurrentUser } from '../types/auth'
import { emptyState, type RequestState } from '../types/common'
import { formatDate } from '../utils/format'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { StatusBadge } from '../components/ui/StatusBadge'
import { DataTable } from '../components/ui/DataTable'
import { EmptyState } from '../components/ui/EmptyState'
import { Field } from '../components/ui/Field'
import { Input } from '../components/ui/Input'
import { useToast } from '../hooks/useToast'
import { validatePositiveInteger } from '../utils/validation'

type AdminPageProps = {
  token: string
  currentUser: CurrentUser | null
  sharedAccountId: string
}

export function AdminPage({ token, currentUser, sharedAccountId }: AdminPageProps) {
  const { showToast } = useToast()
  const [adminUsersState, setAdminUsersState] = useState<RequestState>(emptyState)
  const [adminSessionsState, setAdminSessionsState] = useState<RequestState>(emptyState)
  const [adminAccountState, setAdminAccountState] = useState<RequestState>(emptyState)

  const [adminUsers, setAdminUsers] = useState<AdminUser[]>([])
  const [adminSessions, setAdminSessions] = useState<AdminSession[]>([])
  const [adminAccountId, setAdminAccountId] = useState(sharedAccountId)
  const [adminAccountResult, setAdminAccountResult] =
    useState<AdminAccountStatusResponse | null>(null)

  useEffect(() => {
    if (sharedAccountId) {
      setAdminAccountId(sharedAccountId)
    }
  }, [sharedAccountId])

  const requireAdmin = (setState: (state: RequestState) => void): boolean => {
    if (!token) {
      setState({
        loading: false,
        error: 'Сначала нужно войти в систему.',
        success: '',
      })
      return false
    }

    if (currentUser?.is_admin) {
      return true
    }

    setState({
      loading: false,
      error: 'Доступ разрешен только администратору.',
      success: '',
    })

    return false
  }

  const loadAdminUsers = async () => {
    if (!requireAdmin(setAdminUsersState)) {
      return
    }

    setAdminUsersState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const data = await adminApi.listUsers(token)

      setAdminUsers(Array.isArray(data) ? data : [])
      setAdminUsersState({
        loading: false,
        error: '',
        success: 'Список пользователей загружен.',
      })
    } catch (error) {
      setAdminUsersState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load users',
        success: '',
      })
    }
  }

  const loadAdminSessions = async () => {
    if (!requireAdmin(setAdminSessionsState)) {
      return
    }

    setAdminSessionsState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const data = await adminApi.listSessions(token)

      setAdminSessions(Array.isArray(data) ? data : [])
      setAdminSessionsState({
        loading: false,
        error: '',
        success: 'Список активных сессий загружен.',
      })
    } catch (error) {
      setAdminSessionsState({
        loading: false,
        error:
          error instanceof Error
            ? error.message
            : 'Failed to load logged in users',
        success: '',
      })
    }
  }

  const changeAdminAccountStatus = async (action: 'block' | 'unblock') => {
    if (!requireAdmin(setAdminAccountState)) {
      return
    }

    const validationError = validatePositiveInteger(adminAccountId, 'Account ID')
    if (validationError) {
      setAdminAccountState({
        loading: false,
        error: validationError,
        success: '',
      })
      return
    }

    const accountID = Number(adminAccountId)

    setAdminAccountState({
      loading: true,
      error: '',
      success: '',
    })
    setAdminAccountResult(null)

    try {
      const data = action === 'block'
        ? await adminApi.blockAccount(token, accountID)
        : await adminApi.unblockAccount(token, accountID)

      setAdminAccountResult(data)
      const successMessage = action === 'block' ? 'Счет заблокирован.' : 'Счет разблокирован.'
      setAdminAccountState({
        loading: false,
        error: '',
        success: successMessage,
      })
      showToast(successMessage, 'success')
    } catch (error) {
      const message = error instanceof Error ? error.message : `Failed to ${action} account`
      setAdminAccountState({
        loading: false,
        error: message,
        success: '',
      })
      showToast(message, 'error')
    }
  }

  return (
    <Card variant="plain" className="panel">
      <div className="panelHeader">
        <div>
          <h2>Администрирование</h2>
          <p>
            Все admin-действия API: пользователи, активные сессии, блокировка и разблокировка счетов.
          </p>
        </div>

        <div className="actions">
          <Button
            type="button"
            onClick={loadAdminUsers}
            disabled={adminUsersState.loading || !token}
          >
            {adminUsersState.loading ? 'Загружаю...' : 'Загрузить пользователей'}
          </Button>
          <Button
            className="secondary"
            type="button"
            onClick={loadAdminSessions}
            disabled={adminSessionsState.loading || !token}
          >
            {adminSessionsState.loading ? 'Загружаю...' : 'Активные сессии'}
          </Button>
        </div>
      </div>

      {!currentUser?.is_admin && (
        <EmptyState>
          Этот раздел доступен только администратору. Войдите под admin-пользователем.
        </EmptyState>
      )}

      <div className="adminGrid">
        <section className="subPanel">
          <div className="subPanelHeader">
            <h3>Пользователи</h3>
            <span>{adminUsers.length}</span>
          </div>

          <RequestStatus state={adminUsersState} />

          {adminUsers.length === 0 && (
            <EmptyState>Нажми “Загрузить пользователей”.</EmptyState>
          )}

          {adminUsers.length > 0 && (
            <DataTable
              className="compactTable"
              rows={adminUsers}
              getRowKey={(user) => user.id}
              columns={[
                { key: 'id', title: 'ID', render: (user) => user.id },
                { key: 'email', title: 'Email', render: (user) => user.email },
                { key: 'username', title: 'Username', render: (user) => user.username },
                {
                  key: 'role',
                  title: 'Role',
                  render: (user) => (
                    <StatusBadge tone={user.is_admin ? 'success' : 'muted'}>
                      {user.is_admin ? 'admin' : 'user'}
                    </StatusBadge>
                  ),
                },
                { key: 'accounts', title: 'Accounts', render: (user) => user.accounts_count },
                { key: 'blocked', title: 'Blocked', render: (user) => user.blocked_accounts_count },
                { key: 'created', title: 'Created', render: (user) => formatDate(user.created_at) },
              ]}
            />
          )}
        </section>

        <section className="subPanel">
          <div className="subPanelHeader">
            <h3>Активные сессии</h3>
            <span>{adminSessions.length}</span>
          </div>

          <RequestStatus state={adminSessionsState} />

          {adminSessions.length === 0 && (
            <EmptyState>Нажми “Активные сессии”.</EmptyState>
          )}

          {adminSessions.length > 0 && (
            <DataTable
              className="compactTable"
              rows={adminSessions}
              getRowKey={(session) => session.session_id}
              columns={[
                { key: 'session', title: 'Session', render: (session) => session.session_id },
                { key: 'user', title: 'User', render: (session) => session.user_id },
                { key: 'email', title: 'Email', render: (session) => session.email },
                { key: 'username', title: 'Username', render: (session) => session.username },
                { key: 'created', title: 'Created', render: (session) => formatDate(session.created_at) },
                { key: 'expires', title: 'Expires', render: (session) => formatDate(session.expires_at) },
              ]}
            />
          )}
        </section>

        <section className="subPanel adminAccountPanel">
          <div className="subPanelHeader">
            <h3>Блокировка счета</h3>
          </div>

          <p className="mutedText">
            Используй ID счета. Можно взять ID из раздела Accounts или из test.http.
          </p>

          <div className="form adminAccountForm">
            <Field label="Account ID">
              <Input
                value={adminAccountId}
                onChange={(event) => setAdminAccountId(event.target.value)}
                placeholder="например, 16"
              />
            </Field>

            <div className="actions">
              <Button
                className="danger"
                type="button"
                onClick={() => void changeAdminAccountStatus('block')}
                disabled={adminAccountState.loading || !token}
              >
                {adminAccountState.loading ? 'Выполняю...' : 'Заблокировать'}
              </Button>
              <Button
                className="secondary"
                type="button"
                onClick={() => void changeAdminAccountStatus('unblock')}
                disabled={adminAccountState.loading || !token}
              >
                {adminAccountState.loading ? 'Выполняю...' : 'Разблокировать'}
              </Button>
            </div>
          </div>

          <RequestStatus state={adminAccountState} />

          {adminAccountResult && (
            <div className="result success">
              <strong>{adminAccountResult.message}</strong>
              <pre>{JSON.stringify(adminAccountResult, null, 2)}</pre>
            </div>
          )}
        </section>
      </div>
    </Card>
  )
}
