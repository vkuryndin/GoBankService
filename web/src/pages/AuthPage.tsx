import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { apiRequest } from '../api/http'
import { RequestMessage } from '../components/RequestMessage'
import type { CurrentUser, LoginResponse } from '../types/auth'
import { emptyState, type RequestState } from '../types/common'

type AuthPageProps = {
  token: string
  currentUser: CurrentUser | null
  authCheckState: RequestState
  initialLogin: string
  onLoginValueChange: (login: string) => void
  onLoginSuccess: (token: string) => void
  onLoginFailure: () => void
  onCheckCurrentUser: () => void
}

export function AuthPage({
  token,
  currentUser,
  authCheckState,
  initialLogin,
  onLoginValueChange,
  onLoginSuccess,
  onLoginFailure,
  onCheckCurrentUser,
}: AuthPageProps) {
  const [login, setLogin] = useState(initialLogin || 'test@example.com')
  const [password, setPassword] = useState('password123')
  const [loginState, setLoginState] = useState<RequestState>(emptyState)

  const isAuthenticated = token.trim() !== ''

  const maskedToken = useMemo(() => {
    if (!token) {
      return ''
    }

    if (token.length <= 24) {
      return token
    }

    return `${token.slice(0, 14)}...${token.slice(-10)}`
  }, [token])

  const handleLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    setLoginState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const data = await apiRequest<LoginResponse>('/api/login', {
        method: 'POST',
        body: {
          login,
          password,
        },
      })

      onLoginValueChange(login)
      onLoginSuccess(data.token)
      setLoginState({
        loading: false,
        error: '',
        success: 'Вход выполнен.',
      })
    } catch (error) {
      onLoginFailure()
      setLoginState({
        loading: false,
        error: error instanceof Error ? error.message : 'Login failed',
        success: '',
      })
    }
  }

  return (
    <section className="panel">
      <div className="panelHeader">
        <div>
          <h2>Login и текущий пользователь</h2>
          <p>
            Запросы к <code>POST /login</code> и <code>GET /auth/check</code>.
          </p>
        </div>

        <button
          className="secondary"
          type="button"
          onClick={onCheckCurrentUser}
          disabled={authCheckState.loading || !isAuthenticated}
        >
          {authCheckState.loading ? 'Проверяю...' : 'Кто я сейчас?'}
        </button>
      </div>

      <form className="form" onSubmit={handleLogin}>
        <label>
          <span>Login</span>
          <input
            value={login}
            onChange={(event) => {
              setLogin(event.target.value)
              onLoginValueChange(event.target.value)
            }}
            placeholder="email или username"
            autoComplete="username"
          />
        </label>

        <label>
          <span>Password</span>
          <input
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="password"
            type="password"
            autoComplete="current-password"
          />
        </label>

        <div className="actions">
          <button type="submit" disabled={loginState.loading}>
            {loginState.loading ? 'Вхожу...' : 'Войти'}
          </button>
        </div>
      </form>

      <RequestMessage state={loginState} />
      <RequestMessage state={authCheckState} />

      {isAuthenticated && (
        <div className="tokenBox">
          <span>Текущий токен</span>
          <code>{maskedToken}</code>
        </div>
      )}

      {currentUser && (
        <div className="currentUserBox">
          <strong>Текущая сессия</strong>
          <p>
            Вы вошли как <code>{currentUser.email}</code>.
          </p>
          <p>
            Username: <code>{currentUser.username}</code>
          </p>
          <p>
            Роль:{' '}
            <code>{currentUser.is_admin ? 'администратор' : 'пользователь'}</code>
          </p>
          <p>
            User ID: <code>{currentUser.user_id}</code>
          </p>
        </div>
      )}
    </section>
  )
}
