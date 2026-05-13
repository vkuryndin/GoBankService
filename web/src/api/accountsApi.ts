import type { AccountResponse, CloseAccountResponse, PredictBalanceResponse } from '../types/account'
import { createIdempotencyKey } from '../utils/format'
import { apiRequest } from './client'

export const accountsApi = {
  list(token: string): Promise<AccountResponse[]> {
    return apiRequest<AccountResponse[]>('/api/accounts', { token })
  },

  create(token: string): Promise<AccountResponse> {
    return apiRequest<AccountResponse>('/api/accounts', {
      method: 'POST',
      token,
    })
  },

  get(token: string, accountID: number): Promise<AccountResponse> {
    return apiRequest<AccountResponse>(`/api/accounts/${accountID}`, { token })
  },

  deposit(token: string, accountID: number, amount: string): Promise<AccountResponse> {
    return apiRequest<AccountResponse>(`/api/accounts/${accountID}/deposit`, {
      method: 'POST',
      token,
      headers: { 'Idempotency-Key': createIdempotencyKey() },
      body: { amount },
    })
  },

  withdraw(
    token: string,
    accountID: number,
    amount: string,
    mfaCode: string,
  ): Promise<AccountResponse> {
    return apiRequest<AccountResponse>(`/api/accounts/${accountID}/withdraw`, {
      method: 'POST',
      token,
      headers: { 'Idempotency-Key': createIdempotencyKey() },
      body: {
        amount,
        mfa_code: mfaCode,
      },
    })
  },

  close(token: string, accountID: number): Promise<CloseAccountResponse> {
    return apiRequest<CloseAccountResponse>(`/api/accounts/${accountID}/close`, {
      method: 'POST',
      token,
      headers: { 'Idempotency-Key': createIdempotencyKey() },
    })
  },

  predict(
    token: string,
    accountID: number,
    days: number,
  ): Promise<PredictBalanceResponse> {
    return apiRequest<PredictBalanceResponse>(
      `/api/accounts/${accountID}/predict?days=${days}`,
      { token },
    )
  },
}
