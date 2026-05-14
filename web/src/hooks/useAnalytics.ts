import { useMutation, useQuery } from '@tanstack/react-query'
import { analyticsApi } from '../api/analyticsApi'
import { queryKeys } from '../api/queryKeys'

export function useAnalytics(token: string) {
  const enabled = token.trim() !== ''

  const summaryQuery = useQuery({
    queryKey: queryKeys.analytics.summary(token),
    queryFn: () => analyticsApi.summary(token),
    enabled: false && enabled,
  })

  const predictionMutation = useMutation({
    mutationFn: ({ accountID, days }: { accountID: number; days: number }) =>
      analyticsApi.predictBalance(token, accountID, days),
  })

  return { summaryQuery, predictionMutation }
}
