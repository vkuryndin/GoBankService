import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { adminApi } from '../api/adminApi'
import { queryKeys } from '../api/queryKeys'

export function useAdmin(token: string, enabled: boolean) {
  const queryClient = useQueryClient()

  const usersQuery = useQuery({
    queryKey: queryKeys.admin.users,
    queryFn: () => adminApi.listUsers(token),
    enabled,
  })

  const sessionsQuery = useQuery({
    queryKey: queryKeys.admin.sessions,
    queryFn: () => adminApi.listSessions(token),
    enabled,
  })


  const statisticsQuery = useQuery({
    queryKey: queryKeys.admin.statistics,
    queryFn: () => adminApi.getStatistics(token),
    enabled,
  })

  const blockMutation = useMutation({
    mutationFn: (accountID: number) => adminApi.blockAccount(token, accountID),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.admin.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
    },
  })

  const unblockMutation = useMutation({
    mutationFn: (accountID: number) => adminApi.unblockAccount(token, accountID),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.admin.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
    },
  })

  return {
    users: usersQuery.data || [],
    sessions: sessionsQuery.data || [],
    statistics: statisticsQuery.data || null,
    usersQuery,
    sessionsQuery,
    statisticsQuery,
    blockMutation,
    unblockMutation,
  }
}
