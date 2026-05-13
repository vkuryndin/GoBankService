export type MenuKey =
  | 'health'
  | 'register'
  | 'auth'
  | 'admin'
  | 'accounts'
  | 'cards'
  | 'transfers'
  | 'credits'
  | 'analytics'
  | 'rates'
  | 'notifications'

export type RequestState = {
  loading: boolean
  error: string
  success: string
}

export const emptyState: RequestState = {
  loading: false,
  error: '',
  success: '',
}

export type MenuItem = {
  key: MenuKey
  title: string
  description: string
  icon: string
  implemented: boolean
}
