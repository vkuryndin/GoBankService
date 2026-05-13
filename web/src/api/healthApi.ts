import { apiClient, getErrorMessage } from './client'

export type HealthResult = {
  ok: boolean
  statusCode: number
  body: unknown
}

export const healthApi = {
  async check(): Promise<HealthResult> {
    const response = await apiClient.request<unknown>({
      url: '/api/health',
      method: 'GET',
      headers: {
        Accept: 'application/json',
      },
      validateStatus: () => true,
    })

    return {
      ok: response.status >= 200 && response.status < 300,
      statusCode: response.status,
      body: response.data,
    }
  },

  getErrorMessage(body: unknown): string {
    return getErrorMessage(body)
  },
}
