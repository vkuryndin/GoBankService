export type AccountResponse = {
  id: number
  account_number: string
  balance: string
  currency: string
  is_blocked: boolean
  status: string
  closed_at?: string
  created_at: string
}

export type CloseAccountResponse = {
  id: number
  account_number: string
  status: string
  closed_at: string
  message: string
}

export type PredictBalanceResponse = {
  account_id: number
  days: number
  current_balance: string
  expected_income: string
  expected_expense: string
  scheduled_credit_payments: string
  predicted_balance: string
  average_daily_income: string
  average_daily_expense: string
  statistics_period_days: number
}
