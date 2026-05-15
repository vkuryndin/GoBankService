import type { AdminAccountStatusResponse, AdminSession, AdminSystemStatistics, AdminUser } from '../types/admin'
import { apiRequest } from './client'

export const adminApi = {
  listUsers(_token = ''): Promise<AdminUser[]> {
    return apiRequest<AdminUser[]>('/api/admin/users')
  },

  listSessions(_token = ''): Promise<AdminSession[]> {
    return apiRequest<AdminSession[]>('/api/admin/logged-in-users')
  },


  getStatistics(_token = ''): Promise<AdminSystemStatistics> {
    return apiRequest<AdminSystemStatistics>('/api/admin/statistics')
  },

  blockAccount(_token: string, accountID: number): Promise<AdminAccountStatusResponse> {
    return apiRequest<AdminAccountStatusResponse>(`/api/admin/accounts/${accountID}/block`, {
      method: 'POST',
    })
  },

  unblockAccount(_token: string, accountID: number): Promise<AdminAccountStatusResponse> {
    return apiRequest<AdminAccountStatusResponse>(`/api/admin/accounts/${accountID}/unblock`, {
      method: 'POST',
    })
  },
}
