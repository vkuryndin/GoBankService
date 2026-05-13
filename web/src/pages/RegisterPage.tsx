import { useState } from 'react'
import type { FormEvent } from 'react'
import { authApi } from '../api/authApi'
import { RequestStatus } from '../components/RequestStatus'
import type { RegisterResponse } from '../types/auth'
import { emptyState, type RequestState } from '../types/common'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'

type RegisterPageProps = {
  onRegistered: (email: string) => void
  onOpenAuth: () => void
}

export function RegisterPage({ onRegistered, onOpenAuth }: RegisterPageProps) {
  const [registerEmail, setRegisterEmail] = useState('')
  const [registerUsername, setRegisterUsername] = useState('')
  const [registerPassword, setRegisterPassword] = useState('')
  const [registerState, setRegisterState] = useState<RequestState>(emptyState)
  const [registeredUser, setRegisteredUser] = useState<RegisterResponse | null>(null)

  const handleRegister = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    setRegisterState({
      loading: true,
      error: '',
      success: '',
    })
    setRegisteredUser(null)

    try {
      const data = await authApi.register({
        email: registerEmail,
        username: registerUsername,
        password: registerPassword,
      })

      setRegisteredUser(data)
      setRegisterPassword('')
      onRegistered(data.email)
      setRegisterState({
        loading: false,
        error: '',
        success: 'Пользователь зарегистрирован. Теперь можно войти через Login.',
      })
    } catch (error) {
      setRegisterState({
        loading: false,
        error: error instanceof Error ? error.message : 'Registration failed',
        success: '',
      })
    }
  }

  return (
    <Card variant="plain" className="panel">
      <div className="panelHeader">
        <div>
          <h2>Регистрация</h2>
          <p>
            Запрос к <code>POST /register</code>. После регистрации нужно отдельно выполнить login.
          </p>
        </div>
      </div>

      <form className="form" onSubmit={handleRegister}>
        <label>
          <span>Email</span>
          <input
            value={registerEmail}
            onChange={(event) => setRegisterEmail(event.target.value)}
            placeholder="user@example.com"
            autoComplete="email"
          />
        </label>

        <label>
          <span>Username</span>
          <input
            value={registerUsername}
            onChange={(event) => setRegisterUsername(event.target.value)}
            placeholder="username"
            autoComplete="username"
          />
        </label>

        <label>
          <span>Password</span>
          <input
            value={registerPassword}
            onChange={(event) => setRegisterPassword(event.target.value)}
            placeholder="минимум 8 символов"
            type="password"
            autoComplete="new-password"
          />
        </label>

        <div className="actions">
          <Button type="submit" disabled={registerState.loading}>
            {registerState.loading ? 'Регистрирую...' : 'Зарегистрировать'}
          </Button>
        </div>
      </form>

      <RequestStatus state={registerState} />

      {registeredUser && (
        <div className="result success">
          <strong>Пользователь создан</strong>
          <pre>{JSON.stringify(registeredUser, null, 2)}</pre>
          <Button className="secondary topGap" type="button" onClick={onOpenAuth}>
            Перейти к Login
          </Button>
        </div>
      )}
    </Card>
  )
}
