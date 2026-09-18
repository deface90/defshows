import { create } from 'zustand'
import type { User } from '@/shared/api/auth/model'

const REFRESH_KEY = 'defshows_refresh'

/** getRefreshToken reads the persisted refresh token (localStorage). */
export function getRefreshToken(): string | null {
  try {
    return localStorage.getItem(REFRESH_KEY)
  } catch {
    return null
  }
}

interface AuthState {
  accessToken: string | null
  user: User | null
  isAuthenticated: boolean
  /** setSession stores access+user in memory and persists the refresh token. */
  setSession: (accessToken: string, refreshToken: string, user: User) => void
  /** clear wipes the session and the persisted refresh token. */
  clear: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  user: null,
  isAuthenticated: false,
  setSession: (accessToken, refreshToken, user) => {
    try {
      localStorage.setItem(REFRESH_KEY, refreshToken)
    } catch {
      // ignore storage failures (private mode)
    }
    set({ accessToken, user, isAuthenticated: true })
  },
  clear: () => {
    try {
      localStorage.removeItem(REFRESH_KEY)
    } catch {
      // ignore
    }
    set({ accessToken: null, user: null, isAuthenticated: false })
  },
}))
