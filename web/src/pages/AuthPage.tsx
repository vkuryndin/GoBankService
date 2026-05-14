import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { RequestStatus } from '../components/RequestStatus'
import { emptyState, type RequestState } from '../types/common'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { useAuth } from '../hooks/useAuth'
import { useSharedAccount } from '../hooks/useSharedAccount'
import { useToast } from '../hooks/useToast'
import { firstValidationError, validatePassword, validateRequired } from '../utils/validation'
import { AuthLoginForm } from '../features/auth/AuthLoginForm'

export function AuthPage() {
  const {
    token,
    currentUser,
    authCheckState,
    loginValue,
    setLoginValue,
    login: loginUser,
    clearSession,
    checkCurrentUser,
  } = useAuth()
  const { clearSharedAccountId } = useSharedAccount()
  const { showToast } = useToast()
  const [login, setLogin] = useState(loginValue)
  const [password, setPassword] = useState('')
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

    const validationError = firstValidationError(
      validateRequired(login, 'Login'),
      validatePassword(password),
    )
    if (validationError) {
      setLoginState({ loading: false, error: validationError, success: '' })
      return
    }

    setLoginState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      await loginUser({ login, password })

      setLoginValue(login)
      setLoginState({
        loading: false,
        error: '',
        success: 'Вход выполнен.',
      })
      showToast('Вход выполнен.', 'success')
    } catch (error) {
      clearSession()
      clearSharedAccountId()
      const message = error instanceof Error ? error.message : 'Login failed'
      setLoginState({
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
          <h2>Login и текущий пользователь</h2>
          <p>
            Запросы к <code>POST /login</code> и <code>GET /auth/check</code>.
          </p>
        </div>

        <Button
          className="secondary"
          type="button"
          onClick={() => void checkCurrentUser()}
          disabled={authCheckState.loading || !isAuthenticated}
        >
          {authCheckState.loading ? 'Проверяю...' : 'Кто я сейчас?'}
        </Button>
      </div>

      <AuthLoginForm
        login={login}
        password={password}
        loading={loginState.loading}
        onLoginChange={(value) => {
          setLogin(value)
          setLoginValue(value)
        }}
        onPasswordChange={setPassword}
        onSubmit={handleLogin}
      />

      <RequestStatus state={loginState} />
      <RequestStatus state={authCheckState} />

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
    </Card>
  )
}
