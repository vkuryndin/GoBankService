import type { CardPaymentResponse, CardResponse, CardTransferResponse, CloseCardResponse } from '../types/card'
import { createIdempotencyKey } from '../utils/idempotency'
import { apiRequest } from './client'

export const cardsApi = {
  list(token: string): Promise<CardResponse[]> {
    return apiRequest<CardResponse[]>('/api/cards', { token })
  },

  create(token: string, accountID: number): Promise<CardResponse> {
    return apiRequest<CardResponse>('/api/cards', {
      method: 'POST',
      token,
      body: { account_id: accountID },
    })
  },

  get(token: string, cardID: number): Promise<CardResponse> {
    return apiRequest<CardResponse>(`/api/cards/${cardID}`, { token })
  },

  pay(
    token: string,
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
      token,
      headers: { 'Idempotency-Key': createIdempotencyKey('card') },
      body,
    })
  },

  transfer(
    token: string,
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
      token,
      headers: { 'Idempotency-Key': createIdempotencyKey('card') },
      body,
    })
  },

  close(token: string, cardID: number): Promise<CloseCardResponse> {
    return apiRequest<CloseCardResponse>(`/api/cards/${cardID}/close`, {
      method: 'POST',
      token,
      headers: { 'Idempotency-Key': createIdempotencyKey('card') },
    })
  },
}
