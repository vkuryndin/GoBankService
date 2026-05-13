import axios from 'axios'
import type { AxiosError } from 'axios'
import type { ApiErrorResponse, NormalizedApiError } from '../types/error'

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
  headers: {
    Accept: 'application/json',
  },
})

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

  if (body && typeof body === 'object') {
    const record = body as ApiErrorResponse

    if (typeof record.error === 'string') {
      return record.error
    }

    if (typeof record.message === 'string') {
      return record.message
    }
  }

  return ''
}

export function getErrorCode(body: unknown): string | undefined {
  if (body && typeof body === 'object') {
    const record = body as ApiErrorResponse
    return typeof record.code === 'string' ? record.code : undefined
  }

  return undefined
}

export function getErrorDetails(body: unknown): unknown {
  if (body && typeof body === 'object') {
    return (body as ApiErrorResponse).details
  }

  return undefined
}

export function normalizeApiError(error: unknown, fallback = 'Request failed'): ApiError {
  if (error instanceof ApiError) {
    return error
  }

  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<unknown>
    const body = axiosError.response?.data
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
    throw new ApiError({
      message: getErrorMessage(body) || `HTTP ${response.status}`,
      status: response.status,
      code: getErrorCode(body),
      details: getErrorDetails(body),
      body,
    })
  }

  return body as T
}

export function authHeaders(token: string, withJSON = false): Record<string, string> {
  const headers: Record<string, string> = {
    Accept: 'application/json',
    Authorization: `Bearer ${token}`,
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

  if (options.token) {
    headers.Authorization = `Bearer ${options.token}`
  }

  const hasBody = options.body !== undefined
  if (hasBody && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json'
  }

  try {
    const response = await apiClient.request<unknown>({
      url: path,
      method: options.method || 'GET',
      headers,
      data: hasBody ? options.body : undefined,
      validateStatus: () => true,
    })

    if (response.status < 200 || response.status >= 300) {
      throw new ApiError({
        message: getErrorMessage(response.data) || `HTTP ${response.status}`,
        status: response.status,
        code: getErrorCode(response.data),
        details: getErrorDetails(response.data),
        body: response.data,
      })
    }

    return response.data as T
  } catch (error) {
    throw normalizeApiError(error)
  }
}
