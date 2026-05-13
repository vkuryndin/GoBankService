import type { AuthCheckResponse, LoginResponse, RegisterResponse } from '../types/auth'
import { apiRequest } from './client'

export type RegisterRequest = {
  email: string
  username: string
  password: string
}

export type LoginRequest = {
  login: string
  password: string
}

export const authApi = {
  register(request: RegisterRequest): Promise<RegisterResponse> {
    return apiRequest<RegisterResponse>('/api/register', {
      method: 'POST',
      body: request,
    })
  },

  login(request: LoginRequest): Promise<LoginResponse> {
    return apiRequest<LoginResponse>('/api/login', {
      method: 'POST',
      body: request,
    })
  },

  check(token: string): Promise<AuthCheckResponse> {
    return apiRequest<AuthCheckResponse>('/api/auth/check', { token })
  },

  logout(token: string): Promise<{ message: string }> {
    return apiRequest<{ message: string }>('/api/logout', {
      method: 'POST',
      token,
    })
  },
}
