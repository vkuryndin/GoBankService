import type { AccountResponse, CloseAccountResponse, PredictBalanceResponse } from '../types/account'
import { createIdempotencyKey } from '../utils/idempotency'
import { apiRequest } from './client'

export const accountsApi = {
  list(_token = ''): Promise<AccountResponse[]> {
    return apiRequest<AccountResponse[]>('/api/accounts')
  },

  create(_token = ''): Promise<AccountResponse> {
    return apiRequest<AccountResponse>('/api/accounts', {
      method: 'POST',
    })
  },

  get(_token: string, accountID: number): Promise<AccountResponse> {
    return apiRequest<AccountResponse>(`/api/accounts/${accountID}`)
  },

  deposit(_token: string, accountID: number, amount: string): Promise<AccountResponse> {
    return apiRequest<AccountResponse>(`/api/accounts/${accountID}/deposit`, {
      method: 'POST',
      headers: { 'Idempotency-Key': createIdempotencyKey('account') },
      body: { amount },
    })
  },

  withdraw(
    _token: string,
    accountID: number,
    amount: string,
    mfaCode: string,
  ): Promise<AccountResponse> {
    return apiRequest<AccountResponse>(`/api/accounts/${accountID}/withdraw`, {
      method: 'POST',
      headers: { 'Idempotency-Key': createIdempotencyKey('account') },
      body: {
        amount,
        mfa_code: mfaCode,
      },
    })
  },

  close(_token: string, accountID: number): Promise<CloseAccountResponse> {
    return apiRequest<CloseAccountResponse>(`/api/accounts/${accountID}/close`, {
      method: 'POST',
      headers: { 'Idempotency-Key': createIdempotencyKey('account') },
    })
  },

  predict(
    _token: string,
    accountID: number,
    days: number,
  ): Promise<PredictBalanceResponse> {
    return apiRequest<PredictBalanceResponse>(
      `/api/accounts/${accountID}/predict?days=${days}`,
    )
  },
}
