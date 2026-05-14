import type { AnalyticsResponse } from '../types/analytics'
import { apiRequest } from './client'

export const analyticsApi = {
  summary(_token = ''): Promise<AnalyticsResponse> {
    return apiRequest<AnalyticsResponse>('/api/analytics')
  },
}
