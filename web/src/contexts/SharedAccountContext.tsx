import { createContext, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

type SharedAccountContextValue = {
  sharedAccountId: string
  setSharedAccountId: (accountId: string) => void
  clearSharedAccountId: () => void
}

export const SharedAccountContext = createContext<SharedAccountContextValue | null>(null)

type SharedAccountProviderProps = {
  children: ReactNode
}

export function SharedAccountProvider({ children }: SharedAccountProviderProps) {
  const [sharedAccountId, setSharedAccountId] = useState('')

  const value = useMemo(
    () => ({
      sharedAccountId,
      setSharedAccountId,
      clearSharedAccountId: () => setSharedAccountId(''),
    }),
    [sharedAccountId],
  )

  return (
    <SharedAccountContext.Provider value={value}>
      {children}
    </SharedAccountContext.Provider>
  )
}
