import { useContext } from 'react'
import { SharedAccountContext } from '../contexts/SharedAccountContext'

export function useSharedAccount() {
  const context = useContext(SharedAccountContext)

  if (!context) {
    throw new Error('useSharedAccount must be used within SharedAccountProvider')
  }

  return context
}
