export type ApiErrorBody = {
  error?: string
  message?: string
  code?: string
  details?: unknown
}

export type ApiErrorResponse = ApiErrorBody

export type NormalizedApiError = {
  message: string
  status?: number
  code?: string
  details?: unknown
  body?: unknown
}

export function isApiErrorBody(value: unknown): value is ApiErrorBody {
  if (!value || typeof value !== 'object') {
    return false
  }

  const record = value as Record<string, unknown>

  return (
    record.error === undefined || typeof record.error === 'string'
  ) && (
    record.message === undefined || typeof record.message === 'string'
  ) && (
    record.code === undefined || typeof record.code === 'string'
  )
}
