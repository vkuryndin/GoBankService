import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { mfaApi, type MFARequest } from '../api/mfaApi'
import { emptyState, type RequestState } from '../types/common'

export function useMfaFlow(token: string) {
  const [code, setCode] = useState('')
  const [state, setState] = useState<RequestState>(emptyState)

  const requestMutation = useMutation({
    mutationFn: (body: MFARequest) => mfaApi.request(token, body),
  })

  const requestCode = async (body: MFARequest, successMessage = 'MFA-код отправлен.') => {
    if (!token) {
      setState({ loading: false, error: 'Сначала нужно войти в систему.', success: '' })
      return false
    }

    setState({ loading: true, error: '', success: '' })

    try {
      await requestMutation.mutateAsync(body)
      setState({ loading: false, error: '', success: successMessage })
      return true
    } catch (error) {
      setState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to request MFA code',
        success: '',
      })
      return false
    }
  }

  const reset = () => {
    setCode('')
    setState(emptyState)
  }

  return {
    code,
    setCode,
    state,
    setState,
    requestMutation,
    requestCode,
    reset,
    loading: state.loading,
  }
}
