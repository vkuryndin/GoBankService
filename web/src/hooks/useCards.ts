import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { cardsApi } from '../api/cardsApi'
import { queryKeys } from '../api/queryKeys'

export function useCards(token: string) {
  const queryClient = useQueryClient()
  const enabled = token.trim() !== ''

  const listQuery = useQuery({
    queryKey: queryKeys.cards.list(token),
    queryFn: () => cardsApi.list(token),
    enabled,
  })

  const createMutation = useMutation({
    mutationFn: (accountID: number) => cardsApi.create(token, accountID),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.cards.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
    },
  })

  const closeMutation = useMutation({
    mutationFn: (cardID: number) => cardsApi.close(token, cardID),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.cards.all })
    },
  })

  return {
    cards: listQuery.data || [],
    listQuery,
    createMutation,
    closeMutation,
  }
}
