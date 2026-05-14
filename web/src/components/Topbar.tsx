import type { CurrentUser } from '../types/auth'
import type { RequestState } from '../types/common'
import { Button } from './ui/Button'

type TopbarProps = {
  title: string
  currentUser: CurrentUser | null
  isAuthenticated: boolean
  authCheckState: RequestState
  logoutLoading: boolean
  onLogout: () => void
}

export function Topbar({
  title,
  currentUser,
  isAuthenticated,
  authCheckState,
  logoutLoading,
  onLogout,
}: TopbarProps) {
  const userText = currentUser
    ? `Вы вошли как ${currentUser.email}. Роль: ${
        currentUser.is_admin ? 'администратор' : 'пользователь'
      }.`
    : authCheckState.loading
      ? 'Проверяю текущего пользователя...'
      : authCheckState.error
        ? 'Сессия не подтверждена. Войдите заново.'
        : isAuthenticated
          ? 'Сессия сохранена, пользователь ещё не проверен.'
          : 'Вы не вошли в систему.'

  return (
    <header className="topbar">
      <div>
        <p className="eyebrow">Bank Service</p>
        <h1>{title}</h1>
        <p className="currentUser">{userText}</p>
      </div>

      {isAuthenticated && (
        <Button
          className="logoutButton"
          type="button"
          onClick={onLogout}
          disabled={logoutLoading}
          aria-label="Выйти из текущей сессии"
          aria-busy={logoutLoading}
        >
          {logoutLoading ? 'Выходим...' : 'Выйти'}
        </Button>
      )}
    </header>
  )
}
