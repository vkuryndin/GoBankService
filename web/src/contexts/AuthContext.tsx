import { createContext, useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authApi } from '../api/authApi'
import { queryKeys } from '../api/queryKeys'
import type { LoginRequest } from '../api/authApi'
import type { CurrentUser, LoginResponse } from '../types/auth'
import { emptyState, type RequestState } from '../types/common'

const tokenStorageKey = 'bank_service_token'

type AuthContextValue = {
  token: string
  currentUser: CurrentUser | null
  isAuthenticated: boolean
  authCheckState: RequestState
  logoutState: RequestState
  loginValue: string
  loginLoading: boolean
  setLoginValue: (login: string) => void
  login: (request: LoginRequest) => Promise<LoginResponse>
  logout: () => Promise<void>
  checkCurrentUser: () => Promise<void>
  clearSession: () => void
}

export const AuthContext = createContext<AuthContextValue | null>(null)

type AuthProviderProps = {
  children: ReactNode
}

function getErrorText(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback
}

export function AuthProvider({ children }: AuthProviderProps) {
  const queryClient = useQueryClient()
  const [token, setTokenState] = useState(() => localStorage.getItem(tokenStorageKey) || '')
  const [loginValue, setLoginValue] = useState('test@example.com')
  const [sessionError, setSessionError] = useState('')
  const [logoutState, setLogoutState] = useState<RequestState>(emptyState)

  const isAuthenticated = token.trim() !== ''

  const currentUserQuery = useQuery({
    queryKey: queryKeys.auth.currentUser(token),
    queryFn: () => authApi.check(token),
    enabled: isAuthenticated,
    retry: false,
    staleTime: 30_000,
  })

  const loginMutation = useMutation({
    mutationFn: (request: LoginRequest) => authApi.login(request),
  })

  const logoutMutation = useMutation({
    mutationFn: (tokenToLogout: string) => authApi.logout(tokenToLogout),
  })

  const clearSession = useCallback(() => {
    setTokenState('')
    setSessionError('')
    localStorage.removeItem(tokenStorageKey)
    queryClient.removeQueries()
  }, [queryClient])

  useEffect(() => {
    if (token) {
      localStorage.setItem(tokenStorageKey, token)
      return
    }

    localStorage.removeItem(tokenStorageKey)
  }, [token])

  useEffect(() => {
    if (currentUserQuery.isError && token) {
      setSessionError(getErrorText(currentUserQuery.error, 'Failed to check current user'))
      setTokenState('')
      localStorage.removeItem(tokenStorageKey)
    }
  }, [currentUserQuery.error, currentUserQuery.isError, queryClient, token])

  const login = useCallback(
    async (request: LoginRequest) => {
      setSessionError('')
      setLogoutState(emptyState)

      const data = await loginMutation.mutateAsync(request)
      setTokenState(data.token)
      await queryClient.invalidateQueries({ queryKey: queryKeys.auth.all })

      return data
    },
    [loginMutation, queryClient],
  )

  const logout = useCallback(async () => {
    if (!token) {
      clearSession()
      return
    }

    setLogoutState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      await logoutMutation.mutateAsync(token)
      clearSession()
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
  }, [clearSession, logoutMutation, token])

  const checkCurrentUser = useCallback(async () => {
    if (!token) {
      setSessionError('Токен отсутствует.')
      return
    }

    setSessionError('')
    await currentUserQuery.refetch()
  }, [currentUserQuery, token])

  const currentUser: CurrentUser | null = currentUserQuery.data
    ? {
        authenticated: currentUserQuery.data.authenticated,
        user_id: currentUserQuery.data.user_id,
        email: currentUserQuery.data.email,
        username: currentUserQuery.data.username,
        is_admin: currentUserQuery.data.is_admin,
      }
    : null

  const authCheckState: RequestState = useMemo(() => {
    if (currentUserQuery.isFetching && token) {
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
  }, [currentUser, currentUserQuery.isFetching, sessionError, token])

  const value = useMemo<AuthContextValue>(
    () => ({
      token,
      currentUser,
      isAuthenticated,
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
      isAuthenticated,
      login,
      loginMutation.isPending,
      loginValue,
      logout,
      logoutState,
      token,
    ],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
