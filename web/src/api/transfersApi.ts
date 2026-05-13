import type { TransferResponse } from '../types/transfer'
import { createIdempotencyKey } from '../utils/idempotency'
import { apiRequest } from './client'

export const transfersApi = {
  transfer(
    token: string,
    body: {
      from_account_id: number
      to_account_id: number
      amount: string
      description: string
      mfa_code: string
    },
  ): Promise<TransferResponse> {
    return apiRequest<TransferResponse>('/api/transfer', {
      method: 'POST',
      token,
      headers: { 'Idempotency-Key': createIdempotencyKey('transfer') },
      body,
    })
  },
}
