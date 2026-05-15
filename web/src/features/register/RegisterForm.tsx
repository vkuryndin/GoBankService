import type { FormEvent } from 'react'
import { Button } from '../../components/ui/Button'
import { Field } from '../../components/ui/Field'
import { Input } from '../../components/ui/Input'

type RegisterFormProps = {
  email: string
  username: string
  password: string
  passwordConfirmation: string
  loading: boolean
  onEmailChange: (value: string) => void
  onUsernameChange: (value: string) => void
  onPasswordChange: (value: string) => void
  onPasswordConfirmationChange: (value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}

export function RegisterForm({
  email,
  username,
  password,
  passwordConfirmation,
  loading,
  onEmailChange,
  onUsernameChange,
  onPasswordChange,
  onPasswordConfirmationChange,
  onSubmit,
}: RegisterFormProps) {
  return (
    <form className="form" onSubmit={onSubmit}>
      <Field label="Email">
        <Input
          value={email}
          onChange={(event) => onEmailChange(event.target.value)}
          placeholder="user@example.com"
          autoComplete="email"
        />
      </Field>

      <Field label="Username">
        <Input
          value={username}
          onChange={(event) => onUsernameChange(event.target.value)}
          placeholder="username"
          autoComplete="username"
        />
      </Field>

      <Field label="Password">
        <Input
          value={password}
          onChange={(event) => onPasswordChange(event.target.value)}
          placeholder="минимум 8 символов"
          type="password"
          autoComplete="new-password"
        />
      </Field>

      <Field label="Repeat password">
        <Input
          value={passwordConfirmation}
          onChange={(event) => onPasswordConfirmationChange(event.target.value)}
          placeholder="повтори пароль"
          type="password"
          autoComplete="new-password"
        />
      </Field>

      <div className="actions">
        <Button type="submit" disabled={loading}>
          {loading ? 'Регистрирую...' : 'Зарегистрировать'}
        </Button>
      </div>
    </form>
  )
}
