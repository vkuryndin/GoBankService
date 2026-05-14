import { useQuery } from '@tanstack/react-query'
import { analyticsApi } from '../api/analyticsApi'
import { queryKeys } from '../api/queryKeys'

export function useAnalytics(token: string) {
  const enabled = token.trim() !== ''

  const summaryQuery = useQuery({
    queryKey: queryKeys.analytics.summary,
    queryFn: () => analyticsApi.summary(token),
    enabled: false && enabled,
  })

  return {
    summaryQuery,
  }
}
