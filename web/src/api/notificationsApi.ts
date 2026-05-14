import type { MessageResponse } from '../types/notification'
import { apiRequest } from './client'

export const notificationsApi = {
  sendTestEmail(_token = ''): Promise<MessageResponse> {
    return apiRequest<MessageResponse>('/api/notifications/test')
  },
}
