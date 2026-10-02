# defShows frontend — guide for AI agents

React 19 + Vite + TS (strict) SPA over the backend API. UI: Mantine 9. Server-state:
TanStack Query. Auth-state: zustand. Forms: react-hook-form + zod. Tests: Vitest + RTL
+ MSW. Structure: Feature-Sliced Design lite.

## Layers (import direction: app → pages → features → entities → shared)

```
src/app        providers (Mantine/Query), router, guards, theme, error boundary
src/pages      route components (login, search, my-shows, show-detail, notifications,
               settings, admin/*)
src/features   user actions (auth, add-show, mark-watched, show-links, notes,
               notif-prefs, telegram-link, admin)
src/entities   domain presentation (show, episode, notification, recap)
src/shared     api (generated orval client + http mutator), auth (store/session),
               ui (Layout, state primitives), lib (env), testing (MSW server)
```

## Conventions

- **API codegen (orval)**: `npm run api:gen` regenerates `src/shared/api/<name>/` from
  `../backend/api/openapi/*.yaml`. Never hand-edit generated files. Only `useQuery` hooks
  are generated — **wrap mutations in TanStack `useMutation` over the raw request
  functions** (e.g. `createNote`, `updateSettings`), then invalidate the relevant
  `get<Name>QueryKey()`.
- **HTTP layer**: `src/shared/api/http.ts` owns the axios instance + `customInstance`
  mutator (Bearer attach, single-flight refresh-on-401, logout on failure). It does NOT
  import the store — `session.ts` wires it via `setAuthHooks` at app init.
- **Auth**: `authStore` (zustand) keeps access token + user in memory, refresh token in
  localStorage. `bootstrapSession()` runs before first render; `applySession()` on
  login/register/oauth-exchange. Refresh rotation is single-use with reuse-detection, so
  concurrent 401s share one in-flight refresh (do not break that invariant).
- **Guards**: `RequireAuth` (→ /login) and `RequireRole('admin')` (→ /) in
  `src/app/guards.tsx`. Public routes: /login, /register, /auth/callback.
- **State UI**: use `LoadingState` / `EmptyState` / `ErrorState` from `shared/ui/states`
  on every data page. Unexpected render errors are caught by `QueryErrorBoundary`
  (wraps the Layout Outlet); mutation failures without a local `onError` surface a global
  toast (MutationCache in `app/providers.tsx`).
- **Forms**: react-hook-form + `zodResolver`. Mantine controlled inputs (Select, Switch,
  NumberInput) go through `<Controller>`; plain inputs use `register`.
- **Runtime env**: read the API base URL only via `getApiBaseUrl()` (`shared/lib/env.ts`).
- **Social / feeds**: the `social` client covers follows + feeds. `entities/activity/`
  holds the reusable `ActivityFeed` (cursor "Показать ещё", `fetchPage(cursor?)` memoized
  with `useCallback` by the parent) and `FeedCardView`; it powers both `/feed` (home) and the
  profile **Активность** tab. That tab uses `keepMounted={false}` so a private/own profile
  never fires a feed request. `FollowButton` is 4-state, driven by `is_following`; the nav
  "Подписки" badge counts `GET /me/follows/incoming`. Follow notifications render via
  `NotificationItem` (type labels + `actor_id` → link to the actor profile).
- **Moderation (block / report)**: `features/block-user/BlockButton.tsx` (+ `useBlockMutations`
  hook) and `features/report-user/ReportModal.tsx` (reason select + optional note). Both are
  composed into a shared `ModerationMenu`. Because entities must not import features (FSD rule),
  `FeedCardView`/`ActivityFeed` expose a page-supplied `renderActorMenu` slot; the pages
  (`UserProfilePage`, feed pages) pass the menu in. Block mutations invalidate the profile +
  directory queries; the server tears down edges/visibility, so blocked actors just need a
  refetch (an `onBlocked`/`reload` slot refreshes `ActivityFeed`'s local state). Settings adds
  `features/profile/DisplayNameForm.tsx` (RHF+zod, 1..50, blank = reset, via `updateSettings`)
  and `features/profile/BlockedUsersList.tsx` (`GET /me/blocks` + Unblock, reuses `BlockButton`).
  `FollowsPage` gains a `RemoveFollowerButton` (`DELETE /me/followers/{id}`) on the followers
  tab only.
- **Admin reports**: `pages/admin/ReportsPage.tsx` at `/admin/reports` (under the existing
  `RequireRole('admin')` guard + shared `AdminNav` sub-nav). Table with a `SegmentedControl`
  status filter (open/resolved/dismissed/all), reporter/target links, reason/note, and
  Resolve/Dismiss actions → `POST /admin/reports/{id}/resolve` (invalidates the list).

## Testing

- Vitest + RTL; render via `src/test/render.tsx` (`renderWithProviders`, wraps
  Mantine + Query + MemoryRouter). Mock the network with **MSW** — `server.use(...)` per
  test; the shared server is started in `src/test/setup.ts`.
- Test meaningful units: zod schemas, auth store/session (incl. single-flight refresh),
  guards, mutation flows (success + error), non-trivial hook logic. Don't snapshot
  presentational components.
- jsdom gotcha: Mantine `Textarea autosize` breaks under jsdom — use `rows` instead.
  Modal/portal content mounts after a transition — query it with `findBy*`, not `getBy*`.
- Gate before moving on: `npx tsc -b`, `npx vitest run`, `npm run build` all green.

## Commands

```
npm run dev | build | test | test:watch | lint | api:gen
```
