import type { FormEvent } from 'react'
import { Button } from '../../components/ui/Button'
import { Field } from '../../components/ui/Field'
import { Input } from '../../components/ui/Input'

type AuthLoginFormProps = {
  login: string
  password: string
  loading: boolean
  onLoginChange: (value: string) => void
  onPasswordChange: (value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}

export function AuthLoginForm({
  login,
  password,
  loading,
  onLoginChange,
  onPasswordChange,
  onSubmit,
}: AuthLoginFormProps) {
  return (
    <form className="form" onSubmit={onSubmit}>
      <Field label="Login">
        <Input
          value={login}
          onChange={(event) => onLoginChange(event.target.value)}
          placeholder="email или username"
          autoComplete="username"
        />
      </Field>

      <Field label="Password">
        <Input
          value={password}
          onChange={(event) => onPasswordChange(event.target.value)}
          placeholder="password"
          type="password"
          autoComplete="current-password"
        />
      </Field>

      <div className="actions">
        <Button type="submit" disabled={loading}>
          {loading ? 'Вхожу...' : 'Войти'}
        </Button>
      </div>
    </form>
  )
}
