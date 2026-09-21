import { create } from 'zustand'

/** AuthIntent is a deferred action replayed once the guest authenticates. */
export type AuthIntent = () => void | Promise<void>
export type AuthTab = 'login' | 'register'

interface AuthGateState {
  isOpen: boolean
  tab: AuthTab
  /** pendingIntent is the action to replay after a successful auth, if any. */
  pendingIntent: AuthIntent | null
  open: (opts?: { intent?: AuthIntent; tab?: AuthTab }) => void
  setTab: (tab: AuthTab) => void
  close: () => void
  /** consumeIntent closes the gate and runs the pending intent (if any). */
  consumeIntent: () => void
}

/**
 * useAuthGate drives the global AuthModal. A guest action opens it with an
 * intent; on successful login/registration the modal calls consumeIntent to
 * finish that action in place, without navigating away.
 */
export const useAuthGate = create<AuthGateState>((set, get) => ({
  isOpen: false,
  tab: 'login',
  pendingIntent: null,
  open: (opts) =>
    set({ isOpen: true, tab: opts?.tab ?? 'login', pendingIntent: opts?.intent ?? null }),
  setTab: (tab) => set({ tab }),
  close: () => set({ isOpen: false, pendingIntent: null }),
  consumeIntent: () => {
    const intent = get().pendingIntent
    set({ isOpen: false, pendingIntent: null })
    if (intent) void intent()
  },
}))
