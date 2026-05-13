import type { CreditCheckResponse, CreditResponse, PaymentScheduleResponse } from '../types/credit'
import { createIdempotencyKey } from '../utils/format'
import { apiRequest } from './client'

export type CreditBaseRequest = {
  account_id: number
  principal_amount: string
  term_months: number
}

export const creditsApi = {
  check(token: string, request: CreditBaseRequest): Promise<CreditCheckResponse> {
    return apiRequest<CreditCheckResponse>('/api/credits/check', {
      method: 'POST',
      token,
      body: request,
    })
  },

  create(
    token: string,
    request: CreditBaseRequest & { mfa_code: string },
  ): Promise<CreditResponse> {
    return apiRequest<CreditResponse>('/api/credits', {
      method: 'POST',
      token,
      headers: { 'Idempotency-Key': createIdempotencyKey() },
      body: request,
    })
  },

  list(token: string): Promise<CreditResponse[]> {
    return apiRequest<CreditResponse[]>('/api/credits', { token })
  },

  async listByAccount(token: string, accountID: number): Promise<CreditResponse[]> {
    const credits = await apiRequest<CreditResponse[]>('/api/credits', { token })
    return credits.filter((credit) => credit.account_id === accountID)
  },

  get(token: string, creditID: number): Promise<CreditResponse> {
    return apiRequest<CreditResponse>(`/api/credits/${creditID}`, { token })
  },

  schedule(token: string, creditID: number): Promise<PaymentScheduleResponse[]> {
    return apiRequest<PaymentScheduleResponse[]>(`/api/credits/${creditID}/schedule`, {
      token,
    })
  },
}
