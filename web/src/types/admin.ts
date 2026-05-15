export type AdminUser = {
  id: number
  email: string
  username: string
  is_admin: boolean
  accounts_count: number
  blocked_accounts_count: number
  created_at: string
}

export type AdminSession = {
  session_id: number
  user_id: number
  email: string
  username: string
  created_at: string
  expires_at: string
}

export type AdminAccountStatusResponse = {
  id: number
  user_id: number
  account_number: string
  is_blocked: boolean
  message: string
}


export type AdminSystemStatistics = {
  generated_at: string
  users: AdminUsersStatistics
  accounts: AdminAccountsStatistics
  cards: AdminCardsStatistics
  credits: AdminCreditsStatistics
  transactions: AdminTransactionsStatistics
  audit: AdminAuditStatistics
}

export type AdminUsersStatistics = {
  total: number
  admins: number
  regular_users: number
  new_last_24h: number
  active_sessions: number
}

export type AdminAccountsStatistics = {
  total: number
  active: number
  closed: number
  blocked: number
  total_balance: string
  currency: string
}

export type AdminCardsStatistics = {
  total: number
  active: number
  closed: number
}

export type AdminCreditsStatistics = {
  total: number
  active: number
  closed: number
  overdue: number
  active_principal_amount: string
  active_monthly_payment: string
  currency: string
}

export type AdminTransactionsStatistics = {
  total: number
  completed: number
  failed: number
  last_24h: number
  completed_amount: string
  completed_this_month: string
  currency: string
  by_type: AdminTransactionTypeStatistics[]
  recent: AdminRecentTransaction[]
}

export type AdminTransactionTypeStatistics = {
  type: string
  count: number
  total_amount: string
}

export type AdminRecentTransaction = {
  id: number
  user_id: number
  type: string
  status: string
  amount: string
  currency: string
  description?: string
  created_at: string
}

export type AdminAuditStatistics = {
  total: number
  success: number
  failed: number
  blocked: number
  recent: AdminRecentAuditEvent[]
}

export type AdminRecentAuditEvent = {
  id: number
  user_id?: number
  action: string
  resource_type?: string
  resource_id?: number
  status: string
  created_at: string
}
