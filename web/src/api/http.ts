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
    const record = body as Record<string, unknown>

    if (typeof record.error === 'string') {
      return record.error
    }

    if (typeof record.message === 'string') {
      return record.message
    }
  }

  return ''
}

export async function parseResponse<T>(response: Response): Promise<T> {
  const body = await readResponseBody(response)

  if (!response.ok) {
    throw new Error(getErrorMessage(body) || `HTTP ${response.status}`)
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

type ApiRequestOptions = {
  token?: string
  method?: string
  body?: unknown
  headers?: Record<string, string>
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

  const response = await fetch(path, {
    method: options.method || 'GET',
    headers,
    body: hasBody ? JSON.stringify(options.body) : undefined,
  })

  return parseResponse<T>(response)
}
