import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { accountsApi } from '../api/accountsApi'
import { queryKeys } from '../api/queryKeys'

export function useAccounts(token: string) {
  const queryClient = useQueryClient()
  const enabled = token.trim() !== ''

  const listQuery = useQuery({
    queryKey: queryKeys.accounts.list,
    queryFn: () => accountsApi.list(token),
    enabled,
  })

  const detailMutation = useMutation({
    mutationFn: (accountID: number) => accountsApi.get(token, accountID),
    onSuccess: (account) => {
      queryClient.setQueryData(queryKeys.accounts.detail(account.id), account)
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
    },
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
      queryClient.setQueryData(queryKeys.accounts.detail(account.id), account)
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.analytics.all })
    },
  })

  const withdrawMutation = useMutation({
    mutationFn: ({ accountID, amount, mfaCode }: { accountID: number; amount: string; mfaCode: string }) =>
      accountsApi.withdraw(token, accountID, amount, mfaCode),
    onSuccess: (account) => {
      queryClient.setQueryData(queryKeys.accounts.detail(account.id), account)
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.analytics.all })
    },
  })

  const closeMutation = useMutation({
    mutationFn: (accountID: number) => accountsApi.close(token, accountID),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.cards.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.credits.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.analytics.all })
    },
  })

  const predictionMutation = useMutation({
    mutationFn: ({ accountID, days }: { accountID: number; days: number }) =>
      accountsApi.predict(token, accountID, days),
    onSuccess: (_prediction, request) => {
      queryClient.setQueryData(
        queryKeys.accounts.prediction(request.accountID, request.days),
        _prediction,
      )
    },
  })

  const operationStatisticsMutation = useMutation({
    mutationFn: ({ accountID, limit }: { accountID: number; limit: number }) =>
      accountsApi.operationStatistics(token, accountID, limit),
  })

  return {
    accounts: listQuery.data || [],
    listQuery,
    detailMutation,
    createMutation,
    depositMutation,
    withdrawMutation,
    closeMutation,
    predictionMutation,
    operationStatisticsMutation,
  }
}
