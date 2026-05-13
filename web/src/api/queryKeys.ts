export const queryKeys = {
  auth: {
    all: ['auth'] as const,
    currentUser: (token: string) => ['auth', 'current-user', token] as const,
  },
  health: {
    all: ['health'] as const,
  },
  accounts: {
    all: ['accounts'] as const,
    list: (token: string) => ['accounts', 'list', token] as const,
    detail: (token: string, accountID: number) => ['accounts', 'detail', token, accountID] as const,
    prediction: (token: string, accountID: number, days: number) => [
      'accounts',
      'prediction',
      token,
      accountID,
      days,
    ] as const,
  },
  cards: {
    all: ['cards'] as const,
    list: (token: string) => ['cards', 'list', token] as const,
    detail: (token: string, cardID: number) => ['cards', 'detail', token, cardID] as const,
  },
  credits: {
    all: ['credits'] as const,
    list: (token: string) => ['credits', 'list', token] as const,
    byAccount: (token: string, accountID: number) => ['credits', 'by-account', token, accountID] as const,
    detail: (token: string, creditID: number) => ['credits', 'detail', token, creditID] as const,
    schedule: (token: string, creditID: number) => ['credits', 'schedule', token, creditID] as const,
  },
  admin: {
    all: ['admin'] as const,
    users: (token: string) => ['admin', 'users', token] as const,
    sessions: (token: string) => ['admin', 'sessions', token] as const,
  },
  analytics: {
    all: ['analytics'] as const,
    summary: (token: string) => ['analytics', 'summary', token] as const,
  },
  rates: {
    all: ['rates'] as const,
    keyRate: (token: string) => ['rates', 'key-rate', token] as const,
  },
} as const
