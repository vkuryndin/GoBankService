export const tokenStorageKey = 'bank_service_token'
export const authEventStorageKey = 'bank_service_auth_event'

export type AuthStorageEventType = 'token_changed' | 'logout' | 'session_expired'

export type AuthStorageEvent = {
  type: AuthStorageEventType
  token?: string
  message?: string
  at: number
  source: string
}

const sourceID = crypto.randomUUID()

function canUseStorage() {
  return typeof window !== 'undefined' && typeof localStorage !== 'undefined'
}

function createAuthEvent(
  type: AuthStorageEventType,
  token?: string,
  message?: string,
): AuthStorageEvent {
  return {
    type,
    token,
    message,
    at: Date.now(),
    source: sourceID,
  }
}

function broadcastAuthEvent(event: AuthStorageEvent) {
  if (!canUseStorage()) {
    return
  }

  localStorage.setItem(authEventStorageKey, JSON.stringify(event))
}

export function getAuthToken(): string {
  if (!canUseStorage()) {
    return ''
  }

  return localStorage.getItem(tokenStorageKey) || ''
}

export function setAuthToken(token: string, broadcast = true) {
  if (!canUseStorage()) {
    return
  }

  localStorage.setItem(tokenStorageKey, token)

  if (broadcast) {
    broadcastAuthEvent(createAuthEvent('token_changed', token))
  }
}

export function clearAuthToken(
  type: Exclude<AuthStorageEventType, 'token_changed'> = 'logout',
  message = '',
  broadcast = true,
) {
  if (!canUseStorage()) {
    return
  }

  localStorage.removeItem(tokenStorageKey)

  if (broadcast) {
    broadcastAuthEvent(createAuthEvent(type, undefined, message))
  }
}

export function isOwnAuthEvent(event: AuthStorageEvent) {
  return event.source === sourceID
}

export function parseAuthStorageEvent(rawValue: string | null): AuthStorageEvent | null {
  if (!rawValue) {
    return null
  }

  try {
    const parsed = JSON.parse(rawValue) as Partial<AuthStorageEvent>

    if (
      parsed.type !== 'token_changed' &&
      parsed.type !== 'logout' &&
      parsed.type !== 'session_expired'
    ) {
      return null
    }

    if (typeof parsed.at !== 'number' || typeof parsed.source !== 'string') {
      return null
    }

    return {
      type: parsed.type,
      token: typeof parsed.token === 'string' ? parsed.token : undefined,
      message: typeof parsed.message === 'string' ? parsed.message : undefined,
      at: parsed.at,
      source: parsed.source,
    }
  } catch {
    return null
  }
}
