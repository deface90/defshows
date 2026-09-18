import axios from 'axios'
import type { AuthResponse, TokenPair, User } from '@/shared/api/auth/model'
import { setAuthHooks } from '@/shared/api/http'
import { getApiBaseUrl } from '@/shared/lib/env'
import { getRefreshToken, useAuthStore } from './authStore'

// A bare client (no interceptors) for refresh/me so token refresh never recurses
// through the main axios instance.
const bare = axios.create({ baseURL: getApiBaseUrl() })

async function doRefresh(): Promise<boolean> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) return false
  try {
    const { data: tokens } = await bare.post<TokenPair>('/auth/refresh', { refresh_token: refreshToken })
    const { data: user } = await bare.get<User>('/auth/me', {
      headers: { Authorization: `Bearer ${tokens.access_token}` },
    })
    useAuthStore.getState().setSession(tokens.access_token, tokens.refresh_token, user)
    return true
  } catch {
    useAuthStore.getState().clear()
    return false
  }
}

/** initAuth wires the store into the HTTP interceptors. Call once at app start. */
export function initAuth(): void {
  setAuthHooks({
    getAccessToken: () => useAuthStore.getState().accessToken,
    refresh: doRefresh,
    onLogout: () => useAuthStore.getState().clear(),
  })
}

/** bootstrapSession restores a session from a persisted refresh token, if any. */
export async function bootstrapSession(): Promise<void> {
  if (getRefreshToken()) {
    await doRefresh()
  }
}

/** applySession stores a fresh login/register/exchange result. */
export function applySession(result: AuthResponse): void {
  useAuthStore.getState().setSession(result.tokens.access_token, result.tokens.refresh_token, result.user)
}
