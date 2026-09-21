import { useCallback } from 'react'
import { useAuthGate, type AuthIntent } from './authGate'
import { useAuthStore } from './authStore'

/**
 * useRequireAuth returns a gate: call it with an action. Authenticated users run
 * the action immediately; guests get the AuthModal, and the action is replayed
 * once they sign in. Use it to wrap conversion actions (add show, mark watched)
 * so guests can trigger them and land authenticated with the action done.
 */
export function useRequireAuth() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const open = useAuthGate((s) => s.open)
  return useCallback(
    (action: AuthIntent) => {
      if (isAuthenticated) {
        void action()
        return
      }
      open({ intent: action })
    },
    [isAuthenticated, open],
  )
}
