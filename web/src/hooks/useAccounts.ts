import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { accountsApi } from '../api/accountsApi'
import { queryKeys } from '../api/queryKeys'

export function useAccounts(token: string) {
  const queryClient = useQueryClient()
  const enabled = token.trim() !== ''

  const listQuery = useQuery({
    queryKey: queryKeys.accounts.list(token),
    queryFn: () => accountsApi.list(token),
    enabled,
  })

  const createMutation = useMutation({
    mutationFn: () => accountsApi.create(token),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
    },
  })

  const depositMutation = useMutation({
    mutationFn: ({ accountID, amount }: { accountID: number; amount: string }) =>
      accountsApi.deposit(token, accountID, amount),
    onSuccess: (account) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.detail(token, account.id) })
    },
  })

  const withdrawMutation = useMutation({
    mutationFn: ({ accountID, amount, mfaCode }: { accountID: number; amount: string; mfaCode: string }) =>
      accountsApi.withdraw(token, accountID, amount, mfaCode),
    onSuccess: (account) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.detail(token, account.id) })
    },
  })

  const closeMutation = useMutation({
    mutationFn: (accountID: number) => accountsApi.close(token, accountID),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.cards.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.credits.all })
    },
  })

  return {
    accounts: listQuery.data || [],
    listQuery,
    createMutation,
    depositMutation,
    withdrawMutation,
    closeMutation,
  }
}
