export type RegisterResponse = {
  id: number
  email: string
  username: string
}

export type LoginResponse = {
  token: string
}

export type AuthCheckResponse = {
  authenticated: boolean
  user_id: number
  email: string
  username: string
  is_admin: boolean
}

export type CurrentUser = {
  authenticated: boolean
  user_id: number
  email: string
  username: string
  is_admin: boolean
}
