import { useState } from 'react'
import type { FormEvent } from 'react'
import { RequestStatus } from '../components/RequestStatus'
import type { RegisterResponse } from '../types/auth'
import { emptyState, type RequestState } from '../types/common'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { useToast } from '../hooks/useToast'
import { firstValidationError, validateEmail, validatePassword, validateRequired } from '../utils/validation'
import { useRegistration } from '../hooks/useRegistration'
import { RegisterForm } from '../features/register/RegisterForm'

type RegisterPageProps = {
  onRegistered: (email: string) => void
  onOpenAuth: () => void
}

export function RegisterPage({ onRegistered, onOpenAuth }: RegisterPageProps) {
  const { showToast } = useToast()
  const { registerMutation } = useRegistration()
  const [registerEmail, setRegisterEmail] = useState('')
  const [registerUsername, setRegisterUsername] = useState('')
  const [registerPassword, setRegisterPassword] = useState('')
  const [registerState, setRegisterState] = useState<RequestState>(emptyState)
  const [registeredUser, setRegisteredUser] = useState<RegisterResponse | null>(null)

  const handleRegister = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const validationError = firstValidationError(
      validateEmail(registerEmail),
      validateRequired(registerUsername, 'Username'),
      validatePassword(registerPassword),
    )
    if (validationError) {
      setRegisterState({ loading: false, error: validationError, success: '' })
      return
    }

    setRegisterState({
      loading: true,
      error: '',
      success: '',
    })
    setRegisteredUser(null)

    try {
      const data = await registerMutation.mutateAsync({
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
      showToast('Пользователь зарегистрирован.', 'success')
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Registration failed'
      setRegisterState({
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
          <h2>Регистрация</h2>
          <p>
            Запрос к <code>POST /register</code>. После регистрации нужно отдельно выполнить login.
          </p>
        </div>
      </div>

      <RegisterForm
        email={registerEmail}
        username={registerUsername}
        password={registerPassword}
        loading={registerState.loading}
        onEmailChange={setRegisterEmail}
        onUsernameChange={setRegisterUsername}
        onPasswordChange={setRegisterPassword}
        onSubmit={handleRegister}
      />

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
