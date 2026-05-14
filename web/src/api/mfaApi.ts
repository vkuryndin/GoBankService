import { apiRequest } from './client'

export type MFARequest = Record<string, string | number | boolean>

export const mfaApi = {
  request(_token: string, body: MFARequest): Promise<{ message: string }> {
    return apiRequest<{ message: string }>('/api/mfa/request', {
      method: 'POST',
      body,
    })
  },
}
