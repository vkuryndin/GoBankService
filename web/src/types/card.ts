export type CardResponse = {
  id: number
  account_id: number
  number?: string
  masked_number: string
  expiry?: string
  cvv?: string
  status: string
  closed_at?: string
  created_at: string
}

export type CardPaymentResponse = {
  transaction_id: number
  card_id: number
  account_id: number
  amount: string
  status: string
}

export type CardTransferResponse = {
  transaction_id: number
  from_card_id: number
  to_card_id: number
  from_account_id: number
  to_account_id: number
  amount: string
  status: string
}

export type CloseCardResponse = {
  id: number
  account_id: number
  status: string
  closed_at: string
}
