export type AnalyticsResponse = {
  income_this_month: string
  expense_this_month: string
  credit_load: string
  active_credits_count: number
}

export type OperationDirection = 'income' | 'expense' | 'neutral'

export type OperationTypeStatistics = {
  type: string
  count: number
  total_income: string
  total_expense: string
  net_amount: string
}

export type OperationStatusStatistics = {
  status: string
  count: number
  total_amount: string
}

export type OperationHistoryItem = {
  id: number
  direction: OperationDirection
  type: string
  status: string
  amount: string
  currency: string
  description?: string
  from_account_id?: number
  to_account_id?: number
  from_card_id?: number
  to_card_id?: number
  created_at: string
}

export type OperationStatisticsResponse = {
  entity_type: 'account' | 'card'
  entity_id: number
  currency: string
  operation_count: number
  income_count: number
  expense_count: number
  total_income: string
  total_expense: string
  net_amount: string
  by_type: OperationTypeStatistics[]
  by_status: OperationStatusStatistics[]
  operations: OperationHistoryItem[]
}
