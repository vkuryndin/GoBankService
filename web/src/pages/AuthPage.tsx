import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
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
  const navigate = useNavigate()
  const {
    loginValue,
    setLoginValue,
    login: loginUser,
    clearSession,
  } = useAuth()
  const { clearSharedAccountId } = useSharedAccount()
  const { showToast } = useToast()
  const [login, setLogin] = useState(loginValue)
  const [password, setPassword] = useState('')
  const [loginState, setLoginState] = useState<RequestState>(emptyState)

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
      navigate('/health', { replace: true })
    } catch (error) {
      clearSession({ reason: 'logout', broadcast: true })
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
    <Card variant="plain" className="panel publicAuthCard">
      <div className="publicAuthHeader">
        <div>
          <p className="eyebrow">Bank Service</p>
          <h2>Вход в систему</h2>
          <p>
            Войдите под своим пользователем или создайте нового пользователя.
          </p>
        </div>
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

      <Button
        className="secondary publicSecondaryAction"
        type="button"
        onClick={() => navigate('/register')}
      >
        Зарегистрировать нового пользователя
      </Button>

      <RequestStatus state={loginState} />
    </Card>
  )
}
