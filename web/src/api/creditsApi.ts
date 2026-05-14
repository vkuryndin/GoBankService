import type { CreditCheckResponse, CreditResponse, PaymentScheduleResponse } from '../types/credit'
import { createIdempotencyKey } from '../utils/idempotency'
import { apiRequest } from './client'

export type CreditBaseRequest = {
  account_id: number
  principal_amount: string
  term_months: number
}

export const creditsApi = {
  check(_token: string, request: CreditBaseRequest): Promise<CreditCheckResponse> {
    return apiRequest<CreditCheckResponse>('/api/credits/check', {
      method: 'POST',
      body: request,
    })
  },

  create(
    _token: string,
    request: CreditBaseRequest & { mfa_code: string },
  ): Promise<CreditResponse> {
    return apiRequest<CreditResponse>('/api/credits', {
      method: 'POST',
      headers: { 'Idempotency-Key': createIdempotencyKey('credit') },
      body: request,
    })
  },

  list(_token = ''): Promise<CreditResponse[]> {
    return apiRequest<CreditResponse[]>('/api/credits')
  },

  async listByAccount(_token: string, accountID: number): Promise<CreditResponse[]> {
    const credits = await apiRequest<CreditResponse[]>('/api/credits')
    return credits.filter((credit) => credit.account_id === accountID)
  },

  get(_token: string, creditID: number): Promise<CreditResponse> {
    return apiRequest<CreditResponse>(`/api/credits/${creditID}`)
  },

  schedule(_token: string, creditID: number): Promise<PaymentScheduleResponse[]> {
    return apiRequest<PaymentScheduleResponse[]>(`/api/credits/${creditID}/schedule`)
  },
}
