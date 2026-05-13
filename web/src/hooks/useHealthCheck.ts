import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { healthApi, type HealthResult } from '../api/healthApi'
import { emptyState, type RequestState } from '../types/common'

export function useHealthCheck() {
  const [healthState, setHealthState] = useState<RequestState>(emptyState)
  const [healthResult, setHealthResult] = useState<HealthResult | null>(null)

  const mutation = useMutation({
    mutationFn: () => healthApi.check(),
    onMutate: () => {
      setHealthState({
        loading: true,
        error: '',
        success: '',
      })
      setHealthResult(null)
    },
    onSuccess: (data) => {
      setHealthResult(data)
      setHealthState({
        loading: false,
        error: data.ok
          ? ''
          : healthApi.getErrorMessage(data.body) || `HTTP ${data.statusCode}`,
        success: data.ok ? 'Backend отвечает.' : '',
      })
    },
    onError: (error) => {
      setHealthState({
        loading: false,
        error: error instanceof Error ? error.message : 'Health check failed',
        success: '',
      })
    },
  })

  return {
    healthState,
    healthResult,
    checkHealth: () => mutation.mutate(),
  }
}
