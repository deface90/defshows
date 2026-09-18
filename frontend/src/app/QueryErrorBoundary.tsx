import { Button, Center, Stack, Text, Title } from '@mantine/core'
import { QueryErrorResetBoundary } from '@tanstack/react-query'
import { Component, type ErrorInfo, type ReactNode } from 'react'

interface BoundaryProps {
  onReset: () => void
  children: ReactNode
}
interface BoundaryState {
  error: Error | null
}

class ErrorBoundary extends Component<BoundaryProps, BoundaryState> {
  state: BoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): BoundaryState {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // Surface unexpected render errors in the console for debugging.
    console.error('Uncaught render error:', error, info.componentStack)
  }

  reset = () => {
    this.props.onReset()
    this.setState({ error: null })
  }

  render() {
    if (this.state.error) {
      return (
        <Center py="xl">
          <Stack align="center" gap="xs">
            <Title order={3}>Что-то пошло не так</Title>
            <Text c="dimmed" size="sm" ta="center">
              Произошла непредвиденная ошибка. Попробуйте ещё раз.
            </Text>
            <Button onClick={this.reset}>Повторить</Button>
          </Stack>
        </Center>
      )
    }
    return this.props.children
  }
}

/**
 * QueryErrorBoundary catches unexpected render errors (including those thrown by
 * queries configured with `throwOnError`) and offers a retry that also resets the
 * TanStack Query error state.
 */
export function QueryErrorBoundary({ children }: { children: ReactNode }) {
  return (
    <QueryErrorResetBoundary>
      {({ reset }) => <ErrorBoundary onReset={reset}>{children}</ErrorBoundary>}
    </QueryErrorResetBoundary>
  )
}
