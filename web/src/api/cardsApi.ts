import type { OperationStatisticsResponse } from '../types/analytics'
import type { CardPaymentResponse, CardResponse, CardRevealRequest, CardTransferResponse, CloseCardResponse } from '../types/card'
import { createIdempotencyKey } from '../utils/idempotency'
import { apiRequest } from './client'

export const cardsApi = {
  list(_token = ''): Promise<CardResponse[]> {
    return apiRequest<CardResponse[]>('/api/cards')
  },

  create(_token: string, accountID: number): Promise<CardResponse> {
    return apiRequest<CardResponse>('/api/cards', {
      method: 'POST',
      body: { account_id: accountID },
    })
  },

  get(_token: string, cardID: number): Promise<CardResponse> {
    return apiRequest<CardResponse>(`/api/cards/${cardID}`)
  },

  reveal(_token: string, cardID: number, body: CardRevealRequest): Promise<CardResponse> {
    return apiRequest<CardResponse>(`/api/cards/${cardID}/reveal`, {
      method: 'POST',
      body,
    })
  },

  pay(
    _token: string,
    cardID: number,
    body: {
      amount: string
      cvv: string
      mfa_code: string
      description: string
    },
  ): Promise<CardPaymentResponse> {
    return apiRequest<CardPaymentResponse>(`/api/cards/${cardID}/pay`, {
      method: 'POST',
      headers: { 'Idempotency-Key': createIdempotencyKey('card') },
      body,
    })
  },

  transfer(
    _token: string,
    cardID: number,
    body: {
      to_card_id: number
      amount: string
      cvv: string
      mfa_code: string
      description: string
    },
  ): Promise<CardTransferResponse> {
    return apiRequest<CardTransferResponse>(`/api/cards/${cardID}/transfer`, {
      method: 'POST',
      headers: { 'Idempotency-Key': createIdempotencyKey('card') },
      body,
    })
  },

  close(_token: string, cardID: number): Promise<CloseCardResponse> {
    return apiRequest<CloseCardResponse>(`/api/cards/${cardID}/close`, {
      method: 'POST',
      headers: { 'Idempotency-Key': createIdempotencyKey('card') },
    })
  },

  operationStatistics(
    _token: string,
    cardID: number,
    limit: number,
  ): Promise<OperationStatisticsResponse> {
    return apiRequest<OperationStatisticsResponse>(
      `/api/cards/${cardID}/operations/statistics?limit=${limit}`,
    )
  },
}
