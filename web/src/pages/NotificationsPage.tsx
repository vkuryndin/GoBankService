import { useState } from 'react'
import { RequestStatus } from '../components/RequestStatus'
import { emptyState, type RequestState } from '../types/common'
import type { MessageResponse } from '../types/notification'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { EmptyState } from '../components/ui/EmptyState'
import { useNotifications } from '../hooks/useNotifications'
import { useToast } from '../hooks/useToast'

type NotificationsPageProps = {
  token: string
}

export function NotificationsPage({ token }: NotificationsPageProps) {
  const { sendTestEmailMutation } = useNotifications(token)
  const { showToast } = useToast()
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
      const data = await sendTestEmailMutation.mutateAsync()

      setMessage(data)
      setNotificationState({
        loading: false,
        error: '',
        success: 'Тестовое email-уведомление отправлено.',
      })
      showToast('Тестовое email-уведомление отправлено.', 'success')
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to send test email'
      setNotificationState({
        loading: false,
        error: errorMessage,
        success: '',
      })
      showToast(errorMessage, 'error')
    }
  }

  return (
    <Card variant="plain" className="panel notificationsPage">
      <div className="panelHeader">
        <div>
          <h2>Уведомления</h2>
          <p>
            Проверка SMTP-интеграции: backend отправляет тестовое письмо на email текущего пользователя.
          </p>
        </div>

        <div className="actions">
          <Button
            type="button"
            onClick={sendTestEmail}
            disabled={notificationState.loading || !token}
          >
            {notificationState.loading ? 'Отправляю...' : 'Отправить test email'}
          </Button>
        </div>
      </div>

      <RequestStatus state={notificationState} />

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
        <EmptyState>
          Нажми “Отправить test email”. Если SMTP отключен в env, backend вернет понятную ошибку.
        </EmptyState>
      )}
    </Card>
  )
}
