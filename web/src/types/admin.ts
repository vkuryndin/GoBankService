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
