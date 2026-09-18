import { MantineProvider } from '@mantine/core'
import { Notifications, notifications } from '@mantine/notifications'
import { MutationCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { theme } from './theme'

import '@mantine/core/styles.css'
import '@mantine/notifications/styles.css'
import '@mantine/dates/styles.css'

const queryClient = new QueryClient({
  // Global fallback toast for mutations that don't define their own onError.
  mutationCache: new MutationCache({
    onError: (_error, _vars, _ctx, mutation) => {
      if (mutation.options.onError) return
      notifications.show({ message: 'Не удалось выполнить действие', color: 'red' })
    },
  }),
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false, staleTime: 30_000 },
  },
})

export function Providers({ children }: { children: ReactNode }) {
  return (
    <MantineProvider theme={theme} defaultColorScheme="dark">
      <Notifications position="top-right" />
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    </MantineProvider>
  )
}
