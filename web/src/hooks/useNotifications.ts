import { useMutation } from '@tanstack/react-query'
import { notificationsApi } from '../api/notificationsApi'

export function useNotifications(token: string) {
  const sendTestEmailMutation = useMutation({
    mutationFn: () => notificationsApi.sendTestEmail(token),
  })

  return { sendTestEmailMutation }
}
