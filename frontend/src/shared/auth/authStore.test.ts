import { afterEach, describe, expect, it } from 'vitest'
import type { User } from '@/shared/api/auth/model'
import { getRefreshToken, useAuthStore } from './authStore'

const user: User = { id: 1, display_name: 'A', role: 'user', timezone: 'UTC' }

describe('authStore', () => {
  afterEach(() => {
    useAuthStore.getState().clear()
    localStorage.clear()
  })

  it('setSession stores access/user and persists refresh', () => {
    useAuthStore.getState().setSession('access-1', 'refresh-1', user)
    const s = useAuthStore.getState()
    expect(s.accessToken).toBe('access-1')
    expect(s.isAuthenticated).toBe(true)
    expect(s.user?.id).toBe(1)
    expect(getRefreshToken()).toBe('refresh-1')
  })

  it('clear wipes state and refresh token', () => {
    useAuthStore.getState().setSession('a', 'r', user)
    useAuthStore.getState().clear()
    const s = useAuthStore.getState()
    expect(s.accessToken).toBeNull()
    expect(s.isAuthenticated).toBe(false)
    expect(getRefreshToken()).toBeNull()
  })
})
