import type { KeyRateResponse } from '../types/rate'
import { apiRequest } from './client'

export const ratesApi = {
  keyRate(token: string): Promise<KeyRateResponse> {
    return apiRequest<KeyRateResponse>('/api/rates/key', { token })
  },
}
