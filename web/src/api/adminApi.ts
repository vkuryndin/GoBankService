import type { AdminAccountStatusResponse, AdminSession, AdminUser } from '../types/admin'
import { apiRequest } from './client'

export const adminApi = {
  listUsers(token: string): Promise<AdminUser[]> {
    return apiRequest<AdminUser[]>('/api/admin/users', { token })
  },

  listSessions(token: string): Promise<AdminSession[]> {
    return apiRequest<AdminSession[]>('/api/admin/logged-in-users', { token })
  },

  blockAccount(token: string, accountID: number): Promise<AdminAccountStatusResponse> {
    return apiRequest<AdminAccountStatusResponse>(`/api/admin/accounts/${accountID}/block`, {
      method: 'POST',
      token,
    })
  },

  unblockAccount(token: string, accountID: number): Promise<AdminAccountStatusResponse> {
    return apiRequest<AdminAccountStatusResponse>(`/api/admin/accounts/${accountID}/unblock`, {
      method: 'POST',
      token,
    })
  },
}
