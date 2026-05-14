import type { PredictBalanceResponse } from '../types/account'
import type { AnalyticsResponse } from '../types/analytics'
import { accountsApi } from './accountsApi'
import { apiRequest } from './client'

export const analyticsApi = {
  summary(_token = ''): Promise<AnalyticsResponse> {
    return apiRequest<AnalyticsResponse>('/api/analytics')
  },

  predictBalance(
    _token: string,
    accountID: number,
    days: number,
  ): Promise<PredictBalanceResponse> {
    return accountsApi.predict(_token, accountID, days)
  },
}
