export const queryKeys = {
  auth: {
    all: ['auth'] as const,
    currentUser: ['auth', 'current-user'] as const,
  },
  health: {
    all: ['health'] as const,
  },
  accounts: {
    all: ['accounts'] as const,
    list: ['accounts', 'list'] as const,
    detail: (accountID: number) => ['accounts', 'detail', accountID] as const,
    prediction: (accountID: number, days: number) => [
      'accounts',
      'prediction',
      accountID,
      days,
    ] as const,
  },
  cards: {
    all: ['cards'] as const,
    list: ['cards', 'list'] as const,
    detail: (cardID: number) => ['cards', 'detail', cardID] as const,
  },
  credits: {
    all: ['credits'] as const,
    list: ['credits', 'list'] as const,
    byAccount: (accountID: number) => ['credits', 'by-account', accountID] as const,
    detail: (creditID: number) => ['credits', 'detail', creditID] as const,
    schedule: (creditID: number) => ['credits', 'schedule', creditID] as const,
  },
  admin: {
    all: ['admin'] as const,
    users: ['admin', 'users'] as const,
    sessions: ['admin', 'sessions'] as const,
    statistics: ['admin', 'statistics'] as const,
  },
  analytics: {
    all: ['analytics'] as const,
    summary: ['analytics', 'summary'] as const,
  },
  rates: {
    all: ['rates'] as const,
    keyRate: ['rates', 'key-rate'] as const,
  },
} as const
