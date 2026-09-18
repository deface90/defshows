import { defineConfig } from 'orval'

// Generates typed TanStack Query hooks + models from the backend OpenAPI
// contracts. Each contract lands in its own folder under src/shared/api.
// Requires the backend contracts to exist (backend Tasks 7/10/12/15/16/18).
const contract = (name: string) => ({
  input: `../backend/api/openapi/${name}.yaml`,
  output: {
    mode: 'single' as const,
    target: `src/shared/api/${name}/endpoints.ts`,
    schemas: `src/shared/api/${name}/model`,
    client: 'react-query' as const,
    httpClient: 'axios' as const,
    clean: true,
    prettier: false,
    override: {
      mutator: { path: 'src/shared/api/http.ts', name: 'customInstance' },
      query: { useQuery: true },
    },
  },
})

export default defineConfig({
  auth: contract('auth'),
  shows: contract('shows'),
  tracking: contract('tracking'),
  notifications: contract('notifications'),
  notes: contract('notes'),
  admin: contract('admin'),
  users: contract('users'),
})
