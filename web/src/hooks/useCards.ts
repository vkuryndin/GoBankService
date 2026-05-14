import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { cardsApi } from '../api/cardsApi'
import { queryKeys } from '../api/queryKeys'

export function useCards(token: string) {
  const queryClient = useQueryClient()
  const enabled = token.trim() !== ''

  const listQuery = useQuery({
    queryKey: queryKeys.cards.list,
    queryFn: () => cardsApi.list(token),
    enabled,
  })

  const detailMutation = useMutation({
    mutationFn: (cardID: number) => cardsApi.get(token, cardID),
    onSuccess: (card) => {
      queryClient.setQueryData(queryKeys.cards.detail(card.id), card)
      void queryClient.invalidateQueries({ queryKey: queryKeys.cards.all })
    },
  })

  const createMutation = useMutation({
    mutationFn: (accountID: number) => cardsApi.create(token, accountID),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.cards.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
    },
  })

  const payMutation = useMutation({
    mutationFn: ({
      cardID,
      body,
    }: {
      cardID: number
      body: Parameters<typeof cardsApi.pay>[2]
    }) => cardsApi.pay(token, cardID, body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.cards.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.analytics.all })
    },
  })

  const transferMutation = useMutation({
    mutationFn: ({
      cardID,
      body,
    }: {
      cardID: number
      body: Parameters<typeof cardsApi.transfer>[2]
    }) => cardsApi.transfer(token, cardID, body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.cards.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.analytics.all })
    },
  })

  const closeMutation = useMutation({
    mutationFn: (cardID: number) => cardsApi.close(token, cardID),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.cards.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
    },
  })

  return {
    cards: listQuery.data || [],
    listQuery,
    detailMutation,
    createMutation,
    payMutation,
    transferMutation,
    closeMutation,
  }
}
