# Moderation: Block + Report + Remove-Follower + Editable display_name

## Overview

Add the moderation surface App Store **Guideline 1.2** (apps with social networking /
user-generated content) requires, plus make `display_name` editable. The social layer
(follows, feeds, in-app notifications) already exists
(see [20261001-social-features.md](../completed/20261001-social-features.md)); it currently has **no**
block, **no** report, **no** way to remove an already-accepted follower, and `display_name`
is read-only everywhere.

Apple rejects social apps that let users see/interact with each other but cannot **block**
an abusive user or **report** them. Unfollow is not a substitute: it only curates the
viewer's own feed; it does not stop the other user from viewing/following/re-requesting.

Scope (full parity across backend + iOS + web, as the social feature itself did):

1. **Block** — mutual cut-off. Blocking removes follow edges both ways, forbids
   (re-)follow/requests, makes `CanViewProfile` false in both directions, and drops the
   blocked user out of the directory search. Feeds/lists exclude by construction (edges are
   gone). Endpoints: `POST`/`DELETE /me/blocks/{userId}`, `GET /me/blocks`.
2. **Report** — a `reports` table for admin moderation (reporter, target, reason, note,
   status). `POST /me/reports`; admin `GET /admin/reports` + `POST /admin/reports/{id}/resolve`.
   Apple requires acting on reports ≤24h — a durable queue (not fire-and-forget) backs that.
3. **Remove follower** — eject an already-accepted follower. `DELETE /me/followers/{userId}`.
4. **Editable display_name** — in Settings (web + iOS). Extend the existing `updateSettings`
   contract; validate length.

### Problem it solves / benefits

Unblocks App Store review for the social features and gives users real protection from
abuse (block + report), plus the long-missing ability to edit their display name and remove
unwanted followers.

### Integration

Backend: extend `social_repo`/`social_uc` with blocks + remove-follower; a new
`report_repo`/`report_uc` + handler methods; block-awareness folded into the existing
`CanViewProfile` and `Follow`; directory search gains a block exclusion; `display_name`
rides the existing `updateSettings` path (auth usecase). Web: block/report controls on the
profile + feed, a blocked-list and name field in Settings, remove-follower in `/follows`, an
admin reports section. iOS: hand-written models + SwiftUI mirroring the web.

## Context (from discovery)

- **Already exists (reuse, do NOT rebuild):**
  - `entity/social.go` (`Follow`, `ActivityEvent`), `repository/social_repo.go`
    (`SocialRepository`: Create/Get/Delete/SetFollowStatus, List*, Counts,
    `AcceptedFolloweeIDs`), `usecase/social_uc.go` (`SocialUsecase`, `SocialRepo` iface,
    `Follow`/`Unfollow`/`Approve`/`Reject`, `CanViewProfile`, `FollowState`,
    `OnProfileMadePublic`, follow-notification helpers).
  - `gateways/http/social_handler.go` (follows + feeds), `feed_uc.go`.
  - `users_handler.go`: `ListUsers` (directory search via `auth.ListUsers(ctx,q,limit,offset)`),
    `GetUserProfile`, `ListUserShows` (gated by `CanViewProfile`). `displayNameOf(u)` helper.
  - Settings path: `TrackingHandler.UpdateSettings` (`tracking.yaml` `updateSettings`,
    `UpdateSettingsRequest{timezone, is_public?}` → `Settings{timezone, is_public}`) calls
    `auth.UpdateTimezone`, `auth.SetProfileVisibility`, then `social.OnProfileMadePublic`.
  - `AuthUsecase.SetProfileVisibility` → `repo.SetPublic`; `display_name` lives on
    `entity.User` and the auth `User` schema already exposes it (read-only today).
  - Admin: role guard on `/admin/*` (`router.go` `requireAdminForAdminPaths`);
    `AdminHandler` uses the generic `crud.Repository[T]` (`admin.yaml` dubbing-studios CRUD).
  - `FollowNotifier.DeleteByDedupeKeys` already clears follow notifications (used on
    unfollow/reject) — reuse when a block tears down edges.
  - Wiring in `cmd/web/main.go`: `socialRepo`/`socialUC`/`feedUC`/`usersH`/`socialH`/`adminH`
    all constructed there; `router.go` `RegisterHandlers` auto-registers routes.
- **Patterns:**
  - Codegen: edit `api/openapi/*.yaml` → `make generate` → implement the new method
    (compile fails via `var _ ServerInterface` until done). Never hand-edit `*.gen.go`.
  - Repo errors `ErrNotFound`/`ErrConflict`; usecase → domain errors; handler → HTTP.
  - Usecases own an interface for their deps → unit-tested with fakes; repos tested against
    testcontainers (`internal/testutil`, skips without Docker).
  - Go toolchain: `export PATH=$PATH:/usr/local/go/bin`; `make generate|build|test`.
  - Web: orval `useQuery` only (`npm run api:gen`); mutations wrapped in `useMutation`,
    invalidate `get<Name>QueryKey()`. FSD-lite (`features/`, `entities/`, `pages/`),
    `RequireAuth` guard. Gate: `npx tsc -b` + `npx vitest run` + `npm run build`.
  - iOS (`ios/`): native SwiftUI, **no codegen** — hand-written `Decodable` in the
    `DefShowsModels` module (`Models.swift`/`Social.swift`/`BrowseModels.swift`); networking
    via one `Session` (`get(path,query)` / `mutate(path,method,body)`, `convertFromSnakeCase`).
    Screens: `MainTabsView`/`MoreView`, `UsersView`+`PublicProfileView`, `FollowsView`,
    `FeedView`/`FeedList`, `NotificationsView`.
- **⚠️ Environment caveat:** this dev box is **Linux (no Xcode/Swift)**. iOS code is written
  and reviewed here but **built/tested on the user's Mac or CI** (`swift test` + `xcodebuild`).
  Flag every iOS gate accordingly. Backend repo tests need Docker (testcontainers).

### Key simplifications from discovery (less code than it looks)

- **Feeds need no new filtering.** Home feed only queries *accepted followees*; blocking
  deletes those edges, so blocked actors vanish automatically. Profile feed is gated by
  `CanViewProfile`, which becomes block-aware. → no `feed_uc.go` / `activity_repo` changes.
  - ⚠️ **Revised (code review):** this held *only* if no follow edge can ever coexist with a
    block. But `Follow` and `Block` are not serialized: a follow request can pass its
    `IsBlockedEither` check, then — after `Block` has already run `DeleteFollowEither` — insert
    an accepted edge, which then lingers alongside the block. The home feed read
    (`AcceptedFolloweeIDs`) would surface that blocked actor. Lowest-risk robust fix adopted:
    `AcceptedFolloweeIDs` now excludes blocked ids (either direction) **in SQL**, so the home
    feed never shows a blocked actor regardless of a lingering edge. (Profile feed stays
    `CanViewProfile`-gated, which is already block-aware, so it needs nothing further.)
- **Follower/following lists need no new filtering.** They join `follows`; block deletes the
  edges, so blocked users drop out, and the list view is already `CanViewProfile`-gated.
- **Two read paths learn about blocks directly** (both return users independent of the follow
  graph, so edge-deletion does not cover them):
  1. **Directory search** (`GET /users`) — must exclude blocked ids from the query.
  2. **Profile-by-id** (`GET /users/{id}` = `GetUserProfile`) — ⚠️ this is **not**
     `CanViewProfile`-gated today (it returns id/display_name/is_public/counts/follow-state to
     anyone, guests included). A blocked user who already knows the id (cached feed card,
     shared link, iOS deep link) could still fetch the blocker's profile. It must gain a block
     check (return **404** — don't confirm existence, matching the password-forgot stance).
     This path was the one hole the earlier "no filtering" framing missed.

## Development Approach

- **Testing approach**: Regular (code first, then tests within the same task) — matches the
  social plan and `backend/CLAUDE.md`.
- Complete each task fully before the next; small, focused changes.
- **CRITICAL: every task MUST include new/updated tests** (success + error/edge), listed as
  separate checklist items.
- **CRITICAL: all tests pass before starting the next task.**
- **CRITICAL: update this plan file when scope changes during implementation.**
- Backward compatible: new tables, new nullable columns, additive endpoints.
- Run `make generate` / `npm run api:gen` after editing any contract.

## Testing Strategy

- **Backend unit (usecase, fakes)**: block (idempotent, self-block error, tears down both
  follow edges + clears their notifications, forbids re-follow), `CanViewProfile` matrix
  extended with "blocked either direction → false", `Follow` → `ErrBlocked`, remove-follower
  (accepted only; idempotent), report (valid reason, self-report error, target-exists,
  default status `open`), admin resolve (open→resolved/dismissed), `SetDisplayName`
  validation (trim, length bounds, empty→reset-to-default behavior).
- **Backend repo (testcontainers)**: `blocks` unique + self-block CHECK; `BlockedIDsEither`;
  `DeleteFollowEither`/remove-follower RowsAffected semantics; `reports` insert + status
  filter + resolve; directory search excludes blocked ids.
- **Web (Vitest + RTL + MSW)**: BlockButton/ReportModal states + mutations; blocked-list in
  Settings (unblock); display_name field (save + validation); remove-follower in `/follows`;
  admin reports list + resolve.
- **iOS (XCTest, `swift test`)**: decoding for `BlockedUser`/blocked list, report request
  round-trip, settings model with `display_name`; title/label helpers. View logic kept thin.
- **E2e**: project has no Playwright/Cypress suite — none added.

## Progress Tracking

- Mark completed items `[x]` immediately when done.
- New tasks get ➕ prefix; blockers get ⚠️ prefix.
- Keep the plan in sync with actual work.

## Solution Overview

- **New backend units:** `repository/report_repo.go`, `usecase/report_uc.go`; block methods
  added to `social_repo.go`/`social_uc.go`; report + block handler methods on
  `social_handler.go` (or a small `moderation_handler.go` if `social_handler.go` grows too
  large — decide in Task 4); admin report methods on `AdminHandler`.
- **Blocks** are a directed `blocks(blocker_id, blocked_id)` table; the usecase treats a
  block as symmetric for *visibility* (`BlockedIDsEither`) while keeping the directed row so
  "who blocked whom" is known (only the blocker can unblock).
- **`CanViewProfile`** gains a first-class block check (short-circuits before public/follower
  logic). `Follow` rejects when a block exists in either direction.
- **Reports** persist to a durable table with a `status` lifecycle for admin review.
- **display_name** rides the existing `updateSettings` contract + `AuthUsecase`.

### Key design decisions

- Block tears down follow edges **both directions** in one repo call and clears the pair's
  follow notifications (reuse `DeleteByDedupeKeys`) so no stale request/accepted lingers.
- Blocking is **not** reported to the blocked user (no notification) — standard anti-abuse UX.
- Directory search exclusion is done in SQL (`NOT IN` blocked set), not post-filtering, to
  keep pagination correct. `auth.ListUsers` gains an `excludeIDs []int64` argument.
- Reports use a dedicated repo/usecase (not generic `crud`) because they need a reporter
  join, a status lifecycle, and validation — richer than the dubbing-studio CRUD.
- `display_name` empty/blank input resets to the derived default (email local part /
  `Пользователь #id` via `displayNameOf`), rather than storing an empty string.

## Technical Details

- **Schema** (goose up+down; numbers continue past `0023_notification_actor` — re-check the
  tree with `make migrate-new` and adjust):
  - `0024_blocks`:
    ```sql
    CREATE TABLE blocks (
      id         BIGSERIAL PRIMARY KEY,
      blocker_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      blocked_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      CHECK (blocker_id <> blocked_id),
      UNIQUE (blocker_id, blocked_id)
    );
    CREATE INDEX blocks_blocked_idx ON blocks (blocked_id);
    ```
  - `0025_reports`:
    ```sql
    CREATE TABLE reports (
      id             BIGSERIAL PRIMARY KEY,
      reporter_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      target_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      reason         TEXT NOT NULL CHECK (reason IN ('spam','harassment','inappropriate','other')),
      note           TEXT NOT NULL DEFAULT '',
      status         TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','resolved','dismissed')),
      created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
      resolved_at    TIMESTAMPTZ,
      resolved_by    BIGINT REFERENCES users(id) ON DELETE SET NULL
    );
    CREATE INDEX reports_status_idx ON reports (status, created_at DESC);
    ```
  - `display_name` already exists on `users` (no migration).
- **Entities:** `Block{ID, BlockerID, BlockedID, CreatedAt}` (`TableName "blocks"`),
  `Report{ID, ReporterID, TargetUserID, Reason, Note, Status, CreatedAt, ResolvedAt *time.Time,
  ResolvedBy *int64}` (`TableName "reports"`), `ReportReason`/`ReportStatus` string consts.
- **Block usecase flow** `Block(ctx, blockerID, blockedID)`:
  self-block → `ErrSelfBlock`; target exists; `CreateBlock` (idempotent on `ErrConflict`);
  `DeleteFollowEither(blockerID, blockedID)` (both rows); `clearFollowNotifications` both
  directions. `Unblock` deletes the directed row (idempotent). `Blocked(ctx, userID)` lists
  the users *I* blocked (for the Settings list).
- **Visibility:** add `BlockedIDsEither(ctx, userID) ([]int64, error)` and
  `IsBlockedEither(ctx, a, b) (bool, error)` to the repo. `CanViewProfile` returns false if
  `IsBlockedEither`. `Follow` returns `ErrBlocked` (→ 403) if `IsBlockedEither`. `Approve`
  becomes block-safe: a missing edge (torn down by a block) is treated as idempotent instead
  of surfacing a confusing 404 to the approver. `GetUserProfile` returns 404 if
  `IsBlockedEither(viewerID, targetID)`.
- **Report usecase:** `Report(ctx, reporterID, targetID, reason, note)` validates reason ∈
  set, `note` length ≤ 1000, self-report → `ErrSelfReport`, target exists; inserts `open`.
  Admin: `ListReports(ctx, status, limit, offset)`, `ResolveReport(ctx, id, adminID, status)`
  (`status` ∈ {resolved, dismissed}; sets `resolved_at`/`resolved_by`).
- **display_name:** `AuthUsecase.SetDisplayName(ctx, userID, name)` — trim; if blank, clear
  to `""` (stored) so `displayNameOf` falls back; else enforce 1..50 runes (`ErrInvalidName`).
  `updateSettings` request/response grows a `display_name` field; `Settings` echoes the
  effective name.
- **Directory search:** `auth.ListUsers(ctx, query, excludeIDs, limit, offset)`; the
  `UsersHandler.ListUsers` computes `excludeIDs` from `social.BlockedIDsEither(viewerID)`.
- **Contracts:** extend `social.yaml` (blocks, remove-follower, reports) + `tracking.yaml`
  (`display_name`) + `admin.yaml` (reports). `make generate` after each.

## What Goes Where

- **Implementation Steps** (checkboxes): all code, contracts, codegen, tests, docs.
- **Post-Completion** (no checkboxes): goose migrate on real DB; App Store Connect updates
  (privacy nutrition label, age-rating questionnaire, EULA/guideline URL, review notes with a
  test account pointing reviewers at Block/Report); ship the iOS build after the API deploys.

## Implementation Steps

### Slice 1 — Schema & entities

#### Task 1: Migrations for blocks + reports

**Files:**
- Create: `backend/migrations/migrate/0024_blocks.sql`
- Create: `backend/migrations/migrate/0025_reports.sql`

- [x] `make migrate-new name=blocks` / `name=reports` (tree was at `0023`, so files are
      `0024_blocks.sql` / `0025_reports.sql`; written directly to keep the plan's numbering)
- [x] write `0024_blocks` up (table + self-block CHECK + unique + `blocks_blocked_idx`) and down
- [x] write `0025_reports` up (table + reason/status CHECKs + `reports_status_idx`) and down
- [x] apply up+down locally (extended `pkg/db` `TestMigrations_UpDown` to assert `blocks`/`reports`
      exist after up and survive the full up→reset→re-up cycle on testcontainers; passed)
- [x] run `go build ./...` — must pass before next task

#### Task 2: Block + Report entities

**Files:**
- Modify: `backend/internal/service/entity/social.go`
- Create: `backend/internal/service/entity/report.go`
- Modify/Create: `backend/internal/service/entity/social_test.go` (+ report entity test)

- [x] add `Block` struct (+ `TableName "blocks"`)
- [x] add `Report` struct (+ `TableName "reports"`), `ReportReason` consts
      (`spam`/`harassment`/`inappropriate`/`other`), `ReportStatus` consts
      (`open`/`resolved`/`dismissed`)
- [x] write entity tests asserting `TableName()` values (blocks, reports)
- [x] run `go build ./...` + `go test ./internal/service/entity/...` — green before next task

### Slice 2 — Block (backend)

#### Task 3: Block storage in social repo

**Files:**
- Modify: `backend/internal/service/repository/social_repo.go`
- Modify: `backend/internal/service/repository/social_repo_test.go`

- [x] add `CreateBlock(ctx, *entity.Block) error` (unique-violation → `ErrConflict`)
- [x] add `DeleteBlock(ctx, blockerID, blockedID int64) error` (`RowsAffected==0` → `ErrNotFound`)
- [x] add `ListBlocked(ctx, blockerID, limit, offset int) ([]entity.User, int64, error)`
      (users I blocked, join `blocks`, newest first — mirror `listEdgeUsers`)
- [x] add `IsBlockedEither(ctx, a, b int64) (bool, error)` and
      `BlockedIDsEither(ctx, userID int64) ([]int64, error)`
- [x] add `DeleteFollowEither(ctx, a, b int64) error` (delete both directed follow rows in one
      query; no error if none)
- [x] write testcontainers tests: unique(blocker,blocked), self-block CHECK rejected,
      `IsBlockedEither` both directions, `BlockedIDsEither`, `ListBlocked` pagination,
      `DeleteFollowEither` removes both edges
- [x] run `go test ./internal/service/repository/...` — green before next task

#### Task 4: Block usecase + block-aware CanViewProfile/Follow

**Files:**
- Modify: `backend/internal/service/usecase/social_uc.go`
- Modify: `backend/internal/service/usecase/social_uc_test.go`

- [x] extend the `SocialRepo` interface with the Task 3 methods
- [x] add `ErrSelfBlock`, `ErrBlocked` domain errors
- [x] implement `Block(ctx, blockerID, blockedID)`: self → `ErrSelfBlock`; target exists;
      `CreateBlock` (idempotent on `ErrConflict`); `DeleteFollowEither`;
      `clearFollowNotifications` both directions
- [x] implement `Unblock(ctx, blockerID, blockedID)` (idempotent) and
      `Blocked(ctx, userID, limit, offset)`
- [x] make `CanViewProfile` return `false` when `IsBlockedEither` (short-circuit first)
- [x] make `Follow` return `ErrBlocked` when `IsBlockedEither`
- [x] make `Approve` block-safe: if `IsBlockedEither`, return a no-op (do **not** blanket-
      swallow `ErrNotFound` — the handler's existing 404 "no pending request" for a genuinely
      absent edge must be preserved)
- [x] write usecase tests (fakes): block idempotency, self-block error, edges torn down +
      notifications cleared, re-follow after block → `ErrBlocked`, `CanViewProfile` blocked
      matrix (either direction), unblock restores viewability for a public target,
      `Approve` after the counterpart blocked = no-op (no error)
- [x] run `go test ./internal/service/usecase/...` — green before next task

#### Task 5: Block contract + handler + directory-search exclusion

**Files:**
- Modify: `backend/api/openapi/social.yaml`
- Modify: `backend/internal/gateways/http/social_handler.go`
- Modify: `backend/internal/gateways/http/social_handler_test.go`
- Modify: `backend/internal/service/usecase/auth_uc.go` (`ListUsers` gains `excludeIDs`)
- Modify: `backend/internal/service/repository/user_repo.go` (`ListUsers` `NOT IN` exclude)
- Modify: `backend/internal/gateways/http/users_handler.go` (pass blocked ids)
- Modify: `backend/internal/gateways/http/users_handler_test.go`
- Regenerate: `backend/pkg/server/social` (via `make generate`)

- [x] add to `social.yaml`: `POST /me/blocks/{userId}` (self → 400), `DELETE /me/blocks/{userId}`,
      `GET /me/blocks` (paginated, reuse `FollowUserList`); `make generate`
- [x] implement `BlockUser`/`UnblockUser`/`ListBlocks` on `SocialHandler`; map `ErrSelfBlock`
      → 400, `ErrUserNotFound` → 404
- [x] thread `ErrBlocked` → 403 through `FollowUser`
- [x] gate `GetUserProfile`: extract `viewerID` from claims (0 for guest; it already reads
      claims only for follow-state) and return **404** when `IsBlockedEither(viewerID, targetID)`
- [x] add `excludeIDs []int64` to `auth.ListUsers` + repo `ListUsers`; in the repo **skip the
      `NOT IN` clause entirely when `excludeIDs` is empty** (gorm emits `NOT IN (NULL)` which
      matches nothing — would blank the directory for guests)
- [x] `UsersHandler.ListUsers` must now **extract `viewerID` from claims** (0 if guest — this
      handler does not read claims today) and compute `excludeIDs` from
      `social.BlockedIDsEither(viewerID)` (empty for guests → clause skipped)
- [x] write handler tests: block 204/empty + self 400, unblock, list blocked; follow a blocker
      → 403; `GetUserProfile` of/by a blocked user → 404; directory search omits blocked users
      both directions; guest directory search still returns rows (empty `excludeIDs`)
- [x] write/extend repo test for `ListUsers` exclusion (incl. the empty-slice no-op case)
- [x] run `go test ./...` (backend) — green before next task

### Slice 3 — Remove follower (backend)

#### Task 6: Remove-follower endpoint

**Files:**
- Modify: `backend/internal/service/usecase/social_uc.go`
- Modify: `backend/internal/service/usecase/social_uc_test.go`
- Modify: `backend/api/openapi/social.yaml`
- Modify: `backend/internal/gateways/http/social_handler.go`
- Modify: `backend/internal/gateways/http/social_handler_test.go`
- Regenerate: `backend/pkg/server/social` (via `make generate`)

- [x] add `RemoveFollower(ctx, ownerID, followerID)` to the usecase: delete the
      follower→owner edge (reuse `DeleteFollow(followerID, ownerID)`), clear its follow
      notifications; idempotent (no-op if absent)
- [x] add `DELETE /me/followers/{userId}` to `social.yaml` (204); `make generate`
- [x] implement `RemoveFollower` handler method
- [x] write usecase tests: removes an accepted follower, idempotent when none, pending request
      also removable (equivalent to reject)
- [x] write handler test: 204 + the follower disappears from `GET /users/{me}/followers`
- [x] run `go test ./...` (backend) — green before next task

### Slice 4 — Report (backend)

#### Task 7: Report repository

**Files:**
- Create: `backend/internal/service/repository/report_repo.go`
- Create: `backend/internal/service/repository/report_repo_test.go`

- [x] implement `ReportRepository`: `Create(ctx, *entity.Report)`,
      `List(ctx, status string, limit, offset) ([]entity.Report, int64, error)`
      (status `""` = all), `Get(ctx, id)`, `Resolve(ctx, id, adminID int64, status entity.ReportStatus)`
      (sets `status`/`resolved_at`/`resolved_by`; `RowsAffected==0` → `ErrNotFound`)
- [x] write testcontainers tests: insert, list-by-status + pagination, resolve transitions,
      resolve missing → `ErrNotFound`
- [x] run `go test ./internal/service/repository/...` — green before next task

#### Task 8: Report usecase

**Files:**
- Create: `backend/internal/service/usecase/report_uc.go`
- Create: `backend/internal/service/usecase/report_uc_test.go`

- [x] define `ReportRepo` interface + a user-lookup dep (mirror the existing `SocialUserRepo`
      `FindUserByID` pattern — do not invent a new lookup shape); `ErrSelfReport`, `ErrInvalidReason`
- [x] `Report(ctx, reporterID, targetID, reason, note)`: validate reason ∈ set, `note` ≤ 1000
      runes, self-report → `ErrSelfReport`, target exists (else `ErrUserNotFound`); insert `open`
- [x] `ListReports(ctx, status, limit, offset)` and
      `ResolveReport(ctx, id, adminID, status)` (status ∈ {resolved, dismissed}, else error)
- [x] write usecase tests (fakes): valid report; invalid reason; self-report; unknown target;
      list filter; resolve valid + invalid status
- [x] run `go test ./internal/service/usecase/...` — green before next task

#### Task 9: Report contract + user handler + admin handler

**Files:**
- Modify: `backend/api/openapi/social.yaml` (`POST /me/reports`)
- Modify: `backend/api/openapi/admin.yaml` (`GET /admin/reports`, `POST /admin/reports/{id}/resolve`)
- Modify: `backend/internal/gateways/http/social_handler.go` (+ `_test.go`)
- Modify: `backend/internal/gateways/http/admin_handler.go` (+ `admin_handler_test.go`)
- Modify: `backend/internal/gateways/http/router.go` / `cmd/web/main.go` (construct report
  repo/usecase; pass into social + admin handlers)
- Regenerate: `backend/pkg/server/social`, `backend/pkg/server/admin` (via `make generate`)

- [x] `social.yaml`: `POST /me/reports` body `{target_user_id, reason, note?}` → 201/400/404
- [x] `admin.yaml`: `GET /admin/reports?status=&page=&page_size=` → list w/ reporter+target
      summaries; `POST /admin/reports/{id}/resolve` body `{status}` → 204; `make generate`
- [x] implement `CreateReport` on `SocialHandler` (map `ErrSelfReport` 400, `ErrInvalidReason`
      400, `ErrUserNotFound` 404)
- [x] implement `ListReports`/`ResolveReport` on `AdminHandler` (role already guarded by
      router). Note: `AdminHandler` today takes only `dubbing` — extend `NewAdminHandler` to
      also take the report usecase (+ a user lookup, since the list enriches reporter/target
      summaries) and inject it in `cmd/web/main.go`
- [x] wire report repo/usecase in `cmd/web/main.go`; thread into the social + admin handler
      constructors
- [x] write handler tests: create report 201 + self 400 + bad reason 400; admin list (filter)
      + resolve 204 + resolve missing 404; non-admin blocked by the existing guard
- [x] run `go test ./...` (backend) — green before next task

### Slice 5 — Editable display_name (backend)

#### Task 10: display_name in settings

**Files:**
- Modify: `backend/internal/service/usecase/auth_uc.go` (+ `auth_uc_test.go`)
- Modify: `backend/internal/service/repository/user_repo.go` (if no name setter exists)
- Modify: `backend/api/openapi/tracking.yaml` (`display_name` on request + `Settings`)
- Modify: `backend/internal/gateways/http/tracking_handler.go` (+ `tracking_handler_test.go`)
- Regenerate: `backend/pkg/server/tracking` (via `make generate`)

- [x] add `AuthUsecase.SetDisplayName(ctx, userID, name)`: trim; blank → store `""` (reset);
      else 1..50 runes else `ErrInvalidName`; return fresh user
- [x] add a `display_name` setter on the user repo (`UserRepository.UpdateDisplayName`)
- [x] extend `UpdateSettingsRequest` with optional `display_name`; add `display_name` to the
      `Settings` response schema (echo effective name via `displayNameOf`); `make generate`
- [x] handle `display_name` in `UpdateSettings` (call `SetDisplayName` when present;
      `ErrInvalidName` → 400)
- [x] note the inherited fallback mismatch (conscious choice, don't "fix" here): after a user
      blanks their name, `displayNameOf` falls back to the email local part while
      `actorName` (notification payloads, `social_uc.go`) falls back to `Пользователь #id` —
      labels may differ across surfaces; acceptable for now
- [x] write usecase tests: set name, trim, blank resets, too-long → error; handler test:
      settings PATCH sets name (echoed) + invalid 400 + blank-reset fallback
- [x] run `go test ./...` (backend) — green before next task

### Slice 6 — Web frontend

#### Task 11: Block + Report controls (profile + feed)

**Files:**
- Create: `frontend/src/features/block-user/BlockButton.tsx` (+ `.test.tsx`)
- Create: `frontend/src/features/report-user/ReportModal.tsx` (+ `.test.tsx`)
- Modify: `frontend/src/pages/UserProfilePage.tsx` (overflow menu: Block/Unblock + Report)
- Modify: `frontend/src/entities/activity/FeedCard.tsx` (actor overflow: Block/Report)
- Regenerate: `frontend/src/shared/api/social/*` (via `npm run api:gen`)

- [x] `npm run api:gen` for the new social endpoints
- [x] `BlockButton` (block/unblock) wired to mutations, invalidating the profile + directory
      queries; blocking a user also hides their collection/feed (handled server-side — just
      refetch) (added `useBlockMutations` hook + standalone `BlockButton` unblock control;
      feed refetch via an `onBlocked`/`reload` slot since `ActivityFeed` holds local state)
- [x] `ReportModal` (reason select + optional note) → `POST /me/reports`, success toast
- [x] mount both on the profile overflow menu and on feed-card actors (shared `ModerationMenu`
      composition; `FeedCardView`/`ActivityFeed` gained a page-supplied `renderActorMenu` slot
      to respect the FSD rule that entities can't import features)
- [x] write tests: BlockButton block/unblock mutation bodies; ReportModal submit body +
      validation (reason required); ModerationMenu block + report-open flows
- [x] run gate — `npx tsc -b` + `npx vitest run` + `npm run build` — green before next task

#### Task 12: Settings (blocked list + display_name) + remove-follower in /follows

**Files:**
- Modify: `frontend/src/features/profile/` (settings area) — add a display_name field + a
  blocked-users list with Unblock
- Modify: `frontend/src/pages/FollowsPage.tsx` (Remove button on the "Мои подписчики" tab)
- Regenerate: `frontend/src/shared/api/{social,tracking}/*` (via `npm run api:gen`)

- [x] add a `display_name` text field to Settings (RHF+zod, 1..50; blank allowed = reset),
      saved via `updateSettings`, invalidating the `getMe`/settings query
      (`features/profile/DisplayNameForm.tsx`)
- [x] add a "Заблокированные" list in Settings (`GET /me/blocks`) with an Unblock action
      (`features/profile/BlockedUsersList.tsx`, reuses the Task 11 `BlockButton`)
- [x] add a Remove (убрать из подписчиков) action on the followers tab → `DELETE /me/followers/{id}`
      (`RemoveFollowerButton` in `FollowsPage.tsx`, followers tab only)
- [x] write tests: display_name save + validation; blocked list renders + unblock; remove
      follower mutation + list refetch
- [x] run gate — `npx tsc -b` + `npx vitest run` + `npm run build` — green before next task

#### Task 13: Admin reports section (web)

**Files:**
- Create: `frontend/src/pages/admin/ReportsPage.tsx` (+ `.test.tsx`)
- Modify: admin nav/router (wherever the dubbing-studios admin section is registered)
- Regenerate: `frontend/src/shared/api/admin/*` (via `npm run api:gen`)

- [x] `npm run api:gen` for admin reports (already generated from `admin.yaml`; re-run is a no-op)
- [x] build a reports table (status filter; reporter/target links; reason/note) with
      Resolve/Dismiss actions → `POST /admin/reports/{id}/resolve`, invalidating the list
      (`pages/admin/ReportsPage.tsx`; `SegmentedControl` status filter open/resolved/dismissed/all)
- [x] register the section under the existing admin area (role-guarded) — new `/admin/reports`
      route under the existing `RequireRole('admin')` guard + shared `AdminNav` sub-nav
- [x] write tests: list renders + status filter; resolve/dismiss mutation + refetch
      (`pages/admin/ReportsPage.test.tsx`, 4 tests)
- [x] run gate — `npx tsc -b` + `npx vitest run` + `npm run build` — green before next task

### Slice 7 — iOS (write + review here; build/test on Mac/CI)

#### Task 14: iOS models + Session helpers (block/report/remove-follower/display_name)

**Files:**
- Modify: `ios/DefShows/Social.swift` (blocked-user model, report reason enum/request)
- Modify: `ios/DefShows/Models.swift` (settings model gains `displayName`)
- Modify: `ios/DefShows/Session.swift` (block/unblock/listBlocks, report, removeFollower,
  setDisplayName helpers)
- Modify: `ios/Tests/ModelsTests.swift`
- Modify: `ios/Package.swift` only if a new source file is added

- [x] add `Decodable` models + `Session` helpers for the new endpoints
      (`mutate`/`get` with `convertFromSnakeCase`): `ReportReason` enum + `Report` model in
      `Social.swift`; `displayName` on `AccountSettings` in `BrowseModels.swift`;
      `block`/`unblock`/`blocks`/`removeFollower`/`report`/`setDisplayName` on `Session`
      (blocked list reuses the existing `FollowUserList`). No new source file → `Package.swift`
      unchanged (`Social.swift`/`BrowseModels.swift` already in the module sources).
- [x] write decoding tests: blocked list (`FollowUserList`), `Report` decode, settings w/
      `display_name`, `ReportReason` raw values/labels (in `Tests/ModelsTests.swift`)
- [x] ⚠️ run gate (on Mac/CI) — `swift test` in `ios/`; cannot run on this Linux box; must be
      verified on Mac/CI. Reviewed by hand: Swift syntax, types, snake_case Decodable keys, and
      `Session` helper signatures match existing patterns.

#### Task 15: iOS screens (block/report + settings + remove-follower)

**Files:**
- Modify: `ios/DefShows/UsersView.swift` (`PublicProfileView`: Block/Unblock + Report in a menu)
- Modify: `ios/DefShows/FeedView.swift` (feed-card actor menu: Block/Report)
- Modify: `ios/DefShows/FollowsView.swift` (Remove on the followers tab)
- Modify: `ios/DefShows/MainTabsView.swift` / a Settings view (display_name field +
  "Заблокированные" list with Unblock)
- Modify: `ios/Tests/ModelsTests.swift` (any derived label/title helpers)

- [x] add Block/Unblock + Report menu to `PublicProfileView` and feed-card actors
      (shared `ModerationMenu` + `ReportSheet` in `UsersView.swift`; profile shows a
      toolbar menu + blocked-state `ContentUnavailableView` on 404; feed cards get a
      `contextMenu` Block/Report via a `FeedActorActions` modifier, home feed only.
      `HTTPFailure` made internal so views can branch on the 404 blocked state.)
- [x] add a display_name editor + blocked-users list (Unblock) to Settings
      (`TextField` in the Профиль section, validated via new testable
      `AccountSettings.isValidDisplayName`; saved through a new `Session.patch` that
      echoes the effective name; `BlockedUsersSection` lists `GET /me/blocks` with Unblock.)
- [x] add Remove-follower to `FollowsView` (swipe action on the followers tab only →
      `session.removeFollower`, list refetches)
- [x] write tests for any new derived helpers (e.g. report-reason labels)
      (report-reason labels already covered in Task 14; added
      `testDisplayNameValidationBounds` for the new name validator)
- [x] ⚠️ run gate (on Mac/CI) — `swift test` + Xcode build — cannot run on this Linux box;
      must be verified on Mac/CI. Reviewed by hand: SwiftUI view composition, `Session`
      helper calls, `@State` management, `Identifiable` conformances for `.sheet(item:)`,
      iOS 17 APIs (`TextField(axis:)`, `.topBarTrailing`). No new source file added, so
      `Package.swift` / `project.pbxproj` unchanged (new types live in already-registered
      `UsersView.swift` / `SettingsView.swift` / `BrowseModels.swift`).

### Task 16: Verify acceptance criteria

- [x] all Overview requirements implemented (block mutual cut-off, report + admin review,
      remove-follower, editable display_name)
- [x] edge cases: self-block/self-report, double-block idempotency, re-follow after block
      blocked, blocked users absent from search/feed/lists, name validation
- [x] full backend suite: `go test ./...` (with Docker) + `go vet` clean
- [x] web gate: `npx tsc -b` && `npx vitest run` && `npm run build` — green
- [x] ⚠️ iOS on Mac/CI: `swift test` + Xcode build — cannot run on Linux; verify on Mac/CI
- [x] confirm no generated files were hand-edited (re-running codegen is a no-op:
      `make generate` + `npm run api:gen` produced NO git diff)

### Task 17: Documentation

- [x] update `backend/CLAUDE.md` (blocks + block-aware `CanViewProfile`/`Follow`, reports
      repo/usecase + admin moderation, remove-follower, display_name in settings)
- [x] update `frontend/CLAUDE.md` (block/report features, admin reports section) and
      `ios/README.md` (new settings/moderation UI)
- [x] update memory `impl-status` with the moderation layer
- [ ] HELD — move to `docs/plans/completed/` after iOS Mac/CI gate passes

## Post-Completion
*Manual / external — no checkboxes.*

**Manual verification:**
- Exercise block end-to-end against a real DB: A blocks B → follow edges gone, B can't view A
  or re-follow, A/B absent from each other's search; unblock restores (public) viewability.
- File a report → appears in the admin section → resolve/dismiss updates status.
- iOS (on a Mac): build in Xcode, `swift test`, manually verify block/report/remove-follower
  and the display_name editor.

**External system updates:**
- Run goose migrations `0024`–`0025` on staging/prod.
- **App Store Connect** (the real point of this work): update the **App Privacy** nutrition
  label (social graph/user content), re-answer the **age-rating** questionnaire now that
  block/report exist, confirm the **EULA / guideline URL**, and in **review notes** give a
  test account plus explicit pointers to where Block and Report live. Ship the iOS build only
  after the API changes are deployed.
