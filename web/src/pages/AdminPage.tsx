import { useEffect, useState } from 'react'
import { apiRequest } from '../api/http'
import { RequestMessage } from '../components/RequestMessage'
import type { AdminAccountStatusResponse, AdminSession, AdminUser } from '../types/admin'
import type { CurrentUser } from '../types/auth'
import { emptyState, type RequestState } from '../types/common'
import { formatDate } from '../utils/format'

type AdminPageProps = {
  token: string
  currentUser: CurrentUser | null
  sharedAccountId: string
}

export function AdminPage({ token, currentUser, sharedAccountId }: AdminPageProps) {
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
      const data = await apiRequest<AdminUser[]>('/api/admin/users', { token })

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
      const data = await apiRequest<AdminSession[]>('/api/admin/logged-in-users', {
        token,
      })

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

    const accountID = Number(adminAccountId)
    if (!Number.isInteger(accountID) || accountID <= 0) {
      setAdminAccountState({
        loading: false,
        error: 'Укажи корректный account_id.',
        success: '',
      })
      return
    }

    setAdminAccountState({
      loading: true,
      error: '',
      success: '',
    })
    setAdminAccountResult(null)

    try {
      const data = await apiRequest<AdminAccountStatusResponse>(
        `/api/admin/accounts/${accountID}/${action}`,
        {
          method: 'POST',
          token,
        },
      )

      setAdminAccountResult(data)
      setAdminAccountState({
        loading: false,
        error: '',
        success: action === 'block' ? 'Счет заблокирован.' : 'Счет разблокирован.',
      })
    } catch (error) {
      setAdminAccountState({
        loading: false,
        error:
          error instanceof Error ? error.message : `Failed to ${action} account`,
        success: '',
      })
    }
  }

  return (
    <section className="panel">
      <div className="panelHeader">
        <div>
          <h2>Администрирование</h2>
          <p>
            Все admin-действия API: пользователи, активные сессии, блокировка и разблокировка счетов.
          </p>
        </div>

        <div className="actions">
          <button
            type="button"
            onClick={loadAdminUsers}
            disabled={adminUsersState.loading || !token}
          >
            {adminUsersState.loading ? 'Загружаю...' : 'Загрузить пользователей'}
          </button>
          <button
            className="secondary"
            type="button"
            onClick={loadAdminSessions}
            disabled={adminSessionsState.loading || !token}
          >
            {adminSessionsState.loading ? 'Загружаю...' : 'Активные сессии'}
          </button>
        </div>
      </div>

      {!currentUser?.is_admin && (
        <div className="empty">
          Этот раздел доступен только администратору. Войдите под admin-пользователем.
        </div>
      )}

      <div className="adminGrid">
        <section className="subPanel">
          <div className="subPanelHeader">
            <h3>Пользователи</h3>
            <span>{adminUsers.length}</span>
          </div>

          <RequestMessage state={adminUsersState} />

          {adminUsers.length === 0 && (
            <div className="empty">Нажми “Загрузить пользователей”.</div>
          )}

          {adminUsers.length > 0 && (
            <div className="tableWrap compactTable">
              <table>
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Email</th>
                    <th>Username</th>
                    <th>Role</th>
                    <th>Accounts</th>
                    <th>Blocked</th>
                    <th>Created</th>
                  </tr>
                </thead>
                <tbody>
                  {adminUsers.map((user) => (
                    <tr
                      key={user.id}
                      className={currentUser?.user_id === user.id ? 'currentRow' : ''}
                    >
                      <td>{user.id}</td>
                      <td>{user.email}</td>
                      <td>{user.username}</td>
                      <td>
                        <span className={user.is_admin ? 'badge successBadge' : 'badge mutedBadge'}>
                          {user.is_admin ? 'admin' : 'user'}
                        </span>
                      </td>
                      <td>{user.accounts_count}</td>
                      <td>{user.blocked_accounts_count}</td>
                      <td>{formatDate(user.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        <section className="subPanel">
          <div className="subPanelHeader">
            <h3>Активные сессии</h3>
            <span>{adminSessions.length}</span>
          </div>

          <RequestMessage state={adminSessionsState} />

          {adminSessions.length === 0 && (
            <div className="empty">Нажми “Активные сессии”.</div>
          )}

          {adminSessions.length > 0 && (
            <div className="tableWrap compactTable">
              <table>
                <thead>
                  <tr>
                    <th>Session</th>
                    <th>User</th>
                    <th>Email</th>
                    <th>Username</th>
                    <th>Created</th>
                    <th>Expires</th>
                  </tr>
                </thead>
                <tbody>
                  {adminSessions.map((session) => (
                    <tr
                      key={session.session_id}
                      className={
                        currentUser?.user_id === session.user_id ? 'currentRow' : ''
                      }
                    >
                      <td>{session.session_id}</td>
                      <td>{session.user_id}</td>
                      <td>{session.email}</td>
                      <td>{session.username}</td>
                      <td>{formatDate(session.created_at)}</td>
                      <td>{formatDate(session.expires_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
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
            <label>
              <span>Account ID</span>
              <input
                value={adminAccountId}
                onChange={(event) => setAdminAccountId(event.target.value)}
                placeholder="например, 16"
              />
            </label>

            <div className="actions">
              <button
                className="danger"
                type="button"
                onClick={() => void changeAdminAccountStatus('block')}
                disabled={adminAccountState.loading || !token}
              >
                {adminAccountState.loading ? 'Выполняю...' : 'Заблокировать'}
              </button>
              <button
                className="secondary"
                type="button"
                onClick={() => void changeAdminAccountStatus('unblock')}
                disabled={adminAccountState.loading || !token}
              >
                {adminAccountState.loading ? 'Выполняю...' : 'Разблокировать'}
              </button>
            </div>
          </div>

          <RequestMessage state={adminAccountState} />

          {adminAccountResult && (
            <div className="result success">
              <strong>{adminAccountResult.message}</strong>
              <pre>{JSON.stringify(adminAccountResult, null, 2)}</pre>
            </div>
          )}
        </section>
      </div>
    </section>
  )
}
