import { useMutation, useQueryClient } from '@tanstack/react-query'
import { queryKeys } from '../api/queryKeys'
import { transfersApi } from '../api/transfersApi'

export type TransferMutationRequest = {
  from_account_id: number
  to_account_id?: number
  to_account_number?: string
  amount: string
  description: string
  mfa_code: string
}

export function useTransfers(token: string) {
  const queryClient = useQueryClient()

  const transferMutation = useMutation({
    mutationFn: (request: TransferMutationRequest) => transfersApi.transfer(token, request),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.accounts.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.analytics.all })
    },
  })

  return { transferMutation }
}
