export type ApiErrorResponse = {
  error?: string
  message?: string
  code?: string
  details?: unknown
}

export type NormalizedApiError = {
  message: string
  status?: number
  code?: string
  details?: unknown
  body?: unknown
}
