import type { MessageResponse } from '../types/notification'
import { apiRequest } from './client'

export const notificationsApi = {
  sendTestEmail(token: string): Promise<MessageResponse> {
    return apiRequest<MessageResponse>('/api/notifications/test', { token })
  },
}
