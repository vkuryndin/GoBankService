export type CreditCheckResponse = {
  eligible: boolean
  reason?: string
  reasons?: string[]
  policy_enabled: boolean
  account_id: number
  principal_amount: string
  max_principal_amount: string
  term_months: number
  interest_rate: string
  monthly_payment: string
  active_credits_count: number
  max_active_credits: number
  has_overdue_credit: boolean
  total_principal_amount: string
  max_total_principal_amount: string
  monthly_income: string
  current_monthly_payments: string
  requested_monthly_payment: string
  total_monthly_payments: string
  max_allowed_monthly_payments: string
  debt_load_percent: string
  max_debt_load_percent: number
}

export type CreditResponse = {
  id: number
  account_id: number
  principal_amount: string
  interest_rate: string
  term_months: number
  monthly_payment: string
  status: string
  created_at: string
}

export type CreditPrepaymentMode = 'reduce_payment' | 'reduce_term' | 'full_close'

export type CreditPrepaymentResponse = {
  transaction_id: number
  credit: CreditResponse
  amount: string
  mode: CreditPrepaymentMode
  old_monthly_payment: string
  new_monthly_payment: string
  old_term_months: number
  new_term_months: number
  remaining_debt: string
  closed: boolean
}

export type PaymentScheduleResponse = {
  id: number
  credit_id: number
  payment_date: string
  amount: string
  penalty_amount: string
  status: string
  paid_at?: string
}
