import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { creditsApi, type CreditBaseRequest } from '../api/creditsApi'
import { queryKeys } from '../api/queryKeys'

export function useCredits(token: string, accountID?: number) {
  const queryClient = useQueryClient()
  const enabled = token.trim() !== ''

  const listQuery = useQuery({
    queryKey: queryKeys.credits.list,
    queryFn: () => creditsApi.list(token),
    enabled,
  })

  const accountCreditsQuery = useQuery({
    queryKey: queryKeys.credits.byAccount(accountID || 0),
    queryFn: () => creditsApi.listByAccount(token, accountID || 0),
    enabled: enabled && Boolean(accountID),
  })

  const listByAccountMutation = useMutation({
    mutationFn: (nextAccountID: number) => creditsApi.listByAccount(token, nextAccountID),
    onSuccess: (credits, nextAccountID) => {
      queryClient.setQueryData(queryKeys.credits.byAccount(nextAccountID), credits)
    },
  })

  const checkMutation = useMutation({
    mutationFn: (request: CreditBaseRequest) => creditsApi.check(token, request),
  })

  const createMutation = useMutation({
    mutationFn: (request: CreditBaseRequest & { mfa_code: string }) => creditsApi.create(token, request),
    onSuccess: (credit) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.credits.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.credits.byAccount(credit.account_id) })
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.analytics.all })
    },
  })

  const detailMutation = useMutation({
    mutationFn: (creditID: number) => creditsApi.get(token, creditID),
    onSuccess: (credit) => {
      queryClient.setQueryData(queryKeys.credits.detail(credit.id), credit)
      void queryClient.invalidateQueries({ queryKey: queryKeys.credits.byAccount(credit.account_id) })
    },
  })

  const prepayMutation = useMutation({
    mutationFn: ({
      creditID,
      body,
    }: {
      creditID: number
      body: Parameters<typeof creditsApi.prepay>[2]
    }) => creditsApi.prepay(token, creditID, body),
    onSuccess: (response) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.credits.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.credits.byAccount(response.credit.account_id) })
      void queryClient.invalidateQueries({ queryKey: queryKeys.credits.detail(response.credit.id) })
      void queryClient.invalidateQueries({ queryKey: queryKeys.credits.schedule(response.credit.id) })
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.analytics.all })
    },
  })

  const scheduleMutation = useMutation({
    mutationFn: (creditID: number) => creditsApi.schedule(token, creditID),
    onSuccess: (schedule, creditID) => {
      queryClient.setQueryData(queryKeys.credits.schedule(creditID), schedule)
    },
  })

  return {
    credits: listQuery.data || [],
    accountCredits: accountCreditsQuery.data || [],
    listQuery,
    accountCreditsQuery,
    listByAccountMutation,
    checkMutation,
    createMutation,
    detailMutation,
    prepayMutation,
    scheduleMutation,
  }
}
