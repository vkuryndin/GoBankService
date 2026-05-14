import axios from 'axios'
import type { AxiosError } from 'axios'
import type { ApiErrorBody, NormalizedApiError } from '../types/error'
import { isApiErrorBody } from '../types/error'
import { clearAuthToken } from '../utils/authTokenStorage'
import { emitSessionExpired } from '../utils/sessionEvents'

export type ApiRequestOptions = {
  token?: string
  method?: string
  body?: unknown
  headers?: Record<string, string>
}

export class ApiError extends Error implements NormalizedApiError {
  status?: number
  code?: string
  details?: unknown
  body?: unknown

  constructor(error: NormalizedApiError) {
    super(error.message)
    this.name = 'ApiError'
    this.status = error.status
    this.code = error.code
    this.details = error.details
    this.body = error.body
  }
}

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  withCredentials: true,
  headers: {
    Accept: 'application/json',
  },
})

apiClient.interceptors.response.use(
  (response) => response,
  (error: unknown) => {
    const normalizedError = normalizeApiError(error)

    const requestURL = axios.isAxiosError(error) ? error.config?.url || '' : ''
    const isAuthCheckRequest = requestURL.includes('/auth/check')

    if (normalizedError.status === 401 && !isAuthCheckRequest) {
      const message = normalizedError.message || 'Сессия истекла. Войдите снова.'
      clearAuthToken('session_expired', message)
      emitSessionExpired(message)
    }

    return Promise.reject(normalizedError)
  },
)

export async function readResponseBody(response: Response): Promise<unknown> {
  const contentType = response.headers.get('content-type') || ''

  if (contentType.includes('application/json')) {
    return response.json()
  }

  return response.text()
}

export function getErrorMessage(body: unknown): string {
  if (typeof body === 'string') {
    return body
  }

  if (isApiErrorBody(body)) {
    return body.error || body.message || ''
  }

  return ''
}

export function getErrorCode(body: unknown): string | undefined {
  if (isApiErrorBody(body)) {
    return body.code
  }

  return undefined
}

export function getErrorDetails(body: unknown): unknown {
  if (isApiErrorBody(body)) {
    return body.details
  }

  return undefined
}

function normalizeBody(body: unknown): ApiErrorBody | unknown {
  return isApiErrorBody(body) ? body : body
}

export function normalizeApiError(error: unknown, fallback = 'Request failed'): ApiError {
  if (error instanceof ApiError) {
    return error
  }

  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<unknown>
    const body = normalizeBody(axiosError.response?.data)

    return new ApiError({
      message: getErrorMessage(body) || axiosError.message || fallback,
      status: axiosError.response?.status,
      code: getErrorCode(body),
      details: getErrorDetails(body),
      body,
    })
  }

  if (error instanceof Error) {
    return new ApiError({ message: error.message || fallback })
  }

  return new ApiError({ message: fallback, body: error })
}

export async function parseResponse<T>(response: Response): Promise<T> {
  const body = await readResponseBody(response)

  if (!response.ok) {
    const apiError = new ApiError({
      message: getErrorMessage(body) || `HTTP ${response.status}`,
      status: response.status,
      code: getErrorCode(body),
      details: getErrorDetails(body),
      body,
    })

    if (apiError.status === 401) {
      const message = apiError.message || 'Сессия истекла. Войдите снова.'
      clearAuthToken('session_expired', message)
      emitSessionExpired(message)
    }

    throw apiError
  }

  return body as T
}

export function authHeaders(_token = '', withJSON = false): Record<string, string> {
  const headers: Record<string, string> = {
    Accept: 'application/json',
  }

  if (withJSON) {
    headers['Content-Type'] = 'application/json'
  }

  return headers
}

export async function apiRequest<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<T> {
  const headers: Record<string, string> = {
    Accept: 'application/json',
    ...(options.headers || {}),
  }

  const hasBody = options.body !== undefined
  if (hasBody && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json'
  }

  try {
    const response = await apiClient.request<T>({
      url: path,
      method: options.method || 'GET',
      headers,
      data: hasBody ? options.body : undefined,
    })

    return response.data
  } catch (error) {
    throw normalizeApiError(error)
  }
}
