import { useMutation } from '@tanstack/react-query'
import { authApi } from '../api/authApi'

export type RegistrationRequest = {
  email: string
  username: string
  password: string
}

export function useRegistration() {
  const registerMutation = useMutation({
    mutationFn: (request: RegistrationRequest) => authApi.register(request),
  })

  return { registerMutation }
}
