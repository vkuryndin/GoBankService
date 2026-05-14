import { createContext, useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authApi } from '../api/authApi'
import { queryKeys } from '../api/queryKeys'
import { sessionExpiredEventName } from '../utils/sessionEvents'
import type { SessionExpiredEventDetail } from '../utils/sessionEvents'
import {
  authEventStorageKey,
  clearAuthToken,
  isOwnAuthEvent,
  parseAuthStorageEvent,
  setAuthToken,
} from '../utils/authTokenStorage'
import { useToast } from '../hooks/useToast'
import type { LoginRequest } from '../api/authApi'
import type { CurrentUser, LoginResponse } from '../types/auth'
import { emptyState, type RequestState } from '../types/common'

type ClearSessionOptions = {
  message?: string
  reason?: 'logout' | 'session_expired'
  broadcast?: boolean
}

type AuthContextValue = {
  token: string
  hasToken: boolean
  currentUser: CurrentUser | null
  isAuthenticated: boolean
  isAuthChecking: boolean
  authCheckState: RequestState
  logoutState: RequestState
  loginValue: string
  loginLoading: boolean
  setLoginValue: (login: string) => void
  login: (request: LoginRequest) => Promise<LoginResponse>
  logout: () => Promise<void>
  checkCurrentUser: () => Promise<void>
  clearSession: (options?: ClearSessionOptions | string) => void
}

export const AuthContext = createContext<AuthContextValue | null>(null)

type AuthProviderProps = {
  children: ReactNode
}

const cookieSessionMarker = 'cookie-session'

function getErrorText(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback
}

function normalizeClearSessionOptions(options?: ClearSessionOptions | string): ClearSessionOptions {
  if (typeof options === 'string') {
    return { message: options, reason: 'logout', broadcast: true }
  }

  return {
    message: options?.message || '',
    reason: options?.reason || 'logout',
    broadcast: options?.broadcast ?? true,
  }
}

export function AuthProvider({ children }: AuthProviderProps) {
  const queryClient = useQueryClient()
  const { showToast } = useToast()
  const [token, setTokenState] = useState('')
  const [loginValue, setLoginValue] = useState('')
  const [sessionError, setSessionError] = useState('')
  const [logoutState, setLogoutState] = useState<RequestState>(emptyState)

  const currentUserQuery = useQuery({
    queryKey: queryKeys.auth.currentUser,
    queryFn: () => authApi.check(),
    retry: false,
    staleTime: 30_000,
  })

  const loginMutation = useMutation({
    mutationFn: (request: LoginRequest) => authApi.login(request),
  })

  const logoutMutation = useMutation({
    mutationFn: () => authApi.logout(),
  })

  const clearSession = useCallback((options?: ClearSessionOptions | string) => {
    const normalized = normalizeClearSessionOptions(options)

    setTokenState('')
    setSessionError(normalized.message || '')
    clearAuthToken(normalized.reason, normalized.message, normalized.broadcast)
    queryClient.clear()
  }, [queryClient])

  useEffect(() => {
    const onSessionExpired = (event: Event) => {
      const customEvent = event as CustomEvent<SessionExpiredEventDetail>
      const message = customEvent.detail?.message || 'Сессия истекла. Войдите снова.'
      clearSession({ message, reason: 'session_expired', broadcast: false })
      showToast(message, 'error')
    }

    window.addEventListener(sessionExpiredEventName, onSessionExpired)

    return () => {
      window.removeEventListener(sessionExpiredEventName, onSessionExpired)
    }
  }, [clearSession, showToast])

  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key !== authEventStorageKey) {
        return
      }

      const authEvent = parseAuthStorageEvent(event.newValue)
      if (!authEvent || isOwnAuthEvent(authEvent)) {
        return
      }

      if (authEvent.type === 'token_changed') {
        setTokenState(cookieSessionMarker)
        setSessionError('')
        queryClient.invalidateQueries({ queryKey: queryKeys.auth.currentUser })
        return
      }

      const message = authEvent.message || (
        authEvent.type === 'session_expired'
          ? 'Сессия истекла. Войдите снова.'
          : 'Сессия завершена в другой вкладке.'
      )

      setTokenState('')
      setSessionError(message)
      queryClient.clear()
      showToast(message, authEvent.type === 'session_expired' ? 'error' : 'info')
    }

    window.addEventListener('storage', onStorage)

    return () => {
      window.removeEventListener('storage', onStorage)
    }
  }, [queryClient, showToast])

  useEffect(() => {
    if (currentUserQuery.data?.authenticated) {
      setTokenState(cookieSessionMarker)
      setSessionError('')
      return
    }

    if (currentUserQuery.isError) {
      setTokenState('')
    }
  }, [currentUserQuery.data, currentUserQuery.isError])

  const login = useCallback(
    async (request: LoginRequest) => {
      setSessionError('')
      setLogoutState(emptyState)

      const data = await loginMutation.mutateAsync(request)
      setAuthToken(cookieSessionMarker)
      setTokenState(cookieSessionMarker)
      await queryClient.invalidateQueries({ queryKey: queryKeys.auth.currentUser })
      await currentUserQuery.refetch()

      return data
    },
    [currentUserQuery, loginMutation, queryClient],
  )

  const logout = useCallback(async () => {
    setLogoutState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      await logoutMutation.mutateAsync()
      clearSession({ reason: 'logout', broadcast: true })
      setLogoutState({
        loading: false,
        error: '',
        success: 'Вы вышли из системы.',
      })
    } catch (error) {
      setLogoutState({
        loading: false,
        error: getErrorText(error, 'Logout failed'),
        success: '',
      })
    }
  }, [clearSession, logoutMutation])

  const checkCurrentUser = useCallback(async () => {
    setSessionError('')
    await currentUserQuery.refetch()
  }, [currentUserQuery])

  const currentUser: CurrentUser | null = currentUserQuery.data?.authenticated
    ? {
        authenticated: currentUserQuery.data.authenticated,
        user_id: currentUserQuery.data.user_id,
        email: currentUserQuery.data.email,
        username: currentUserQuery.data.username,
        is_admin: currentUserQuery.data.is_admin,
      }
    : null

  const isAuthChecking = currentUserQuery.isLoading || currentUserQuery.isFetching
  const isAuthenticated = Boolean(currentUser?.authenticated)
  const hasToken = token.trim() !== '' || isAuthenticated
  const effectiveToken = hasToken ? cookieSessionMarker : ''

  const authCheckState: RequestState = useMemo(() => {
    if (isAuthChecking) {
      return {
        loading: true,
        error: '',
        success: '',
      }
    }

    if (sessionError) {
      return {
        loading: false,
        error: sessionError,
        success: '',
      }
    }

    if (currentUser) {
      return {
        loading: false,
        error: '',
        success: `Вы вошли как ${currentUser.email}. Роль: ${
          currentUser.is_admin ? 'администратор' : 'пользователь'
        }.`,
      }
    }

    return emptyState
  }, [currentUser, isAuthChecking, sessionError])

  const value = useMemo<AuthContextValue>(
    () => ({
      token: effectiveToken,
      hasToken,
      currentUser,
      isAuthenticated,
      isAuthChecking,
      authCheckState,
      logoutState,
      loginValue,
      loginLoading: loginMutation.isPending,
      setLoginValue,
      login,
      logout,
      checkCurrentUser,
      clearSession,
    }),
    [
      authCheckState,
      checkCurrentUser,
      clearSession,
      currentUser,
      effectiveToken,
      hasToken,
      isAuthenticated,
      isAuthChecking,
      login,
      loginMutation.isPending,
      loginValue,
      logout,
      logoutState,
    ],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
