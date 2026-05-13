import type { PredictBalanceResponse } from '../types/account'
import type { AnalyticsResponse } from '../types/analytics'
import { accountsApi } from './accountsApi'
import { apiRequest } from './client'

export const analyticsApi = {
  summary(token: string): Promise<AnalyticsResponse> {
    return apiRequest<AnalyticsResponse>('/api/analytics', { token })
  },

  predictBalance(
    token: string,
    accountID: number,
    days: number,
  ): Promise<PredictBalanceResponse> {
    return accountsApi.predict(token, accountID, days)
  },
}
