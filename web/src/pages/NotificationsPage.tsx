import { useState } from 'react'
import { apiRequest } from '../api/http'
import { RequestMessage } from '../components/RequestMessage'
import { emptyState, type RequestState } from '../types/common'
import type { MessageResponse } from '../types/notification'

type NotificationsPageProps = {
  token: string
}

export function NotificationsPage({ token }: NotificationsPageProps) {
  const [notificationState, setNotificationState] = useState<RequestState>(emptyState)
  const [message, setMessage] = useState<MessageResponse | null>(null)

  const sendTestEmail = async () => {
    if (!token) {
      setNotificationState({
        loading: false,
        error: 'Сначала нужно войти в систему.',
        success: '',
      })
      return
    }

    setNotificationState({
      loading: true,
      error: '',
      success: '',
    })
    setMessage(null)

    try {
      const data = await apiRequest<MessageResponse>('/api/notifications/test', {
        token,
      })

      setMessage(data)
      setNotificationState({
        loading: false,
        error: '',
        success: 'Тестовое email-уведомление отправлено.',
      })
    } catch (error) {
      setNotificationState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to send test email',
        success: '',
      })
    }
  }

  return (
    <section className="panel notificationsPage">
      <div className="panelHeader">
        <div>
          <h2>Уведомления</h2>
          <p>
            Проверка SMTP-интеграции: backend отправляет тестовое письмо на email текущего пользователя.
          </p>
        </div>

        <div className="actions">
          <button
            type="button"
            onClick={sendTestEmail}
            disabled={notificationState.loading || !token}
          >
            {notificationState.loading ? 'Отправляю...' : 'Отправить test email'}
          </button>
        </div>
      </div>

      <RequestMessage state={notificationState} />

      <div className="notificationInfoGrid">
        <div className="notificationInfoCard">
          <span>Endpoint</span>
          <strong>GET /notifications/test</strong>
        </div>

        <div className="notificationInfoCard">
          <span>Назначение</span>
          <strong>SMTP smoke test</strong>
        </div>

        <div className="notificationInfoCard">
          <span>Получатель</span>
          <strong>Email текущего пользователя</strong>
        </div>
      </div>

      {message && (
        <div className="result success">
          <strong>Ответ backend</strong>
          <pre>{JSON.stringify(message, null, 2)}</pre>
        </div>
      )}

      {!message && !notificationState.error && (
        <div className="empty">
          Нажми “Отправить test email”. Если SMTP отключен в env, backend вернет понятную ошибку.
        </div>
      )}
    </section>
  )
}
