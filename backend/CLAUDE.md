# defShows backend — guide for AI agents

Go (echo) distributed monolith: several `cmd/` binaries sharing one Postgres DB.
No message bus. See [../docs/2026-09-15-defshows.md](../docs/completed/2026-09-15-defshows.md).

## Layout & layers

```
cmd/{auth,web,notifier,worker}   thin mains wiring config→db→repos→usecases→handlers
internal/service/entity          gorm models (schema owned by goose, not AutoMigrate)
internal/service/repository      gorm data access; errors: ErrNotFound / ErrConflict
internal/service/usecase         business logic; depends on interfaces, not concretes
internal/service/workers         background jobs (catalog sync, notify scanner/sender)
internal/gateways/http           echo handlers implementing generated ServerInterfaces
internal/gateways/providers/{tmdb,omdb}  external provider impls over generated clients
internal/testutil                shared testcontainers Postgres (skips if no Docker)
pkg/config,log,db,auth,notify,oauth,crud   reusable infra
pkg/server/<svc>                 GENERATED echo servers (do not edit)
pkg/clients/<svc>                GENERATED http clients (do not edit)
api/openapi/*.yaml               service contracts  → pkg/server via oapi-codegen
deps/<svc>/openapi/*.yaml        third-party contracts → pkg/clients
migrations/migrate/*.sql         goose migrations (embedded via migrations pkg)
deploy/                          docker-compose, Dockerfile, .env.example
```

## Conventions

- **Codegen**: `make generate` (oapi-codegen as a go tool; directives in
  `api/openapi/generate.go`). Regenerate after editing any `*.yaml`. Never hand-edit
  `*.gen.go`.
- **Adding an endpoint**: edit the contract yaml → `make generate` → implement the new
  method on the handler (compile fails until you do, via the `var _ ServerInterface`
  assertion) → register is automatic.
- **Migrations**: `make migrate-new name=...`; up/down required. Nullable-unique needs a
  partial or COALESCE unique index (see 0001 telegram, 0006 recaps).
- **Usecases** own an interface for their repo/provider deps → unit-tested with fakes;
  repos tested against a real container via `testutil.MigratedPostgresDB`.
- **Errors**: repo returns `ErrNotFound`/`ErrConflict`; usecases translate to domain
  errors (e.g. `ErrAlreadyTracked`); handlers map to HTTP status.
- **Auth**: JWT HS256 access + single-use rotating refresh with reuse-detection (family
  revoke). `pkg/auth` middleware; web router guards all but public auth paths, plus an
  admin-role guard on `/admin/*`. OAuth uses a one-time handoff code (tokens never in URL).
  Password reset (`/auth/password/forgot` → emailed link → `/auth/password/reset`) and
  change (`/auth/password/change`) both revoke all of the user's refresh tokens; forgot
  never reveals whether an email exists. Mail goes through `pkg/mailer` (SMTP, or a
  log-only fallback when `SMTP_*` is unset).
- **gorm gotcha**: struct zero-values are sent on insert, bypassing SQL defaults — set
  enum/status fields explicitly (repos default them defensively, e.g. airing_status).
- **Social layer** (`social_repo`/`social_uc`, `activity_repo`, `feed_uc`): follows carry a
  `status` (pending/accepted); `CanViewProfile(viewer,target)` = owner || public ||
  accepted-follower centralizes privacy (collection + feeds route through it). Going public
  auto-accepts pending — orchestrated at the **handler** level (`UpdateSettings` calls
  `social.OnProfileMadePublic` after `SetProfileVisibility`) to avoid an auth→social dep.
- **Moderation — blocks** (`social_repo`/`social_uc`, migration `0024_blocks`): directed
  `blocks(blocker_id, blocked_id)` row, but visibility is treated **symmetrically**. `Block`
  tears down the follow edges **both directions** in one call (`DeleteFollowEither`) and clears
  the pair's follow notifications (`DeleteByDedupeKeys`), so no stale request/accepted lingers;
  it is idempotent (`ErrConflict`→no-op) and rejects self-block (`ErrSelfBlock`). Blocking is
  **not** notified to the blocked user. `CanViewProfile` short-circuits to false when
  `IsBlockedEither`; `Follow` returns `ErrBlocked` (→403); `Approve` is block-safe (a missing
  edge torn down by a block is a no-op, but a genuinely absent edge still 404s). `GetUserProfile`
  (not `CanViewProfile`-gated) gains its own `IsBlockedEither(viewer,target)` check and returns
  **404** (don't confirm existence). Directory search excludes blocked ids **in SQL** —
  `auth.ListUsers(ctx, query, excludeIDs, limit, offset)` with `excludeIDs` from
  `social.BlockedIDsEither(viewerID)`; the repo **skips the `NOT IN` clause entirely when
  `excludeIDs` is empty** (gorm would emit `NOT IN (NULL)` and blank the directory for guests).
  Follow-lists need no new filtering — they join `follows` and the edges are already gone. The
  Follow/Block TOCTOU is closed **at the source**: `Follow` inserts through
  `SocialRepository.CreateFollowGuarded`, a transaction that re-checks `blocks` (either
  direction) atomically with the edge insert and returns `repository.ErrBlocked` (→ `ErrBlocked`
  → 403) if a concurrent `Block` committed its row first — so no surviving edge/notification can
  leak past a block or be resurrected by a later `Unblock`. As defence-in-depth the home feed
  still excludes blocked ids (either direction) **in SQL** inside `AcceptedFolloweeIDs`. Profile
  feed stays `CanViewProfile`-gated (block-aware).
  Endpoints: `POST`/`DELETE /me/blocks/{userId}`, `GET /me/blocks` (reuses `FollowUserList`).
- **Moderation — reports** (`report_repo`/`report_uc`, migration `0025_reports`): dedicated
  repo/usecase (not generic `crud`) because they need a reporter/target join, a `status`
  lifecycle, and validation. `reports(reporter_id, target_user_id, reason, note, status,
  resolved_at, resolved_by)`; reason ∈ {spam,harassment,inappropriate,other}, status ∈
  {open,resolved,dismissed}. `Report` validates reason, `note` ≤1000 runes, self-report
  (`ErrSelfReport`), target exists; inserts `open`. Admin `ListReports(status,limit,offset)`
  (`""`=all) and `ResolveReport(id,adminID,status)` (sets `resolved_at`/`resolved_by`).
  Endpoints: user `POST /me/reports`; admin `GET /admin/reports` + `POST /admin/reports/{id}/resolve`
  (behind the existing `/admin/*` role guard). `AdminHandler` was extended to take the report
  usecase + a user lookup (enriches reporter/target summaries); wired in `cmd/web/main.go`.
  Apple requires acting on reports ≤24h — the durable queue backs that.
- **Remove follower** (`social_uc.RemoveFollower`): ejects an already-accepted (or pending)
  follower by deleting the follower→owner edge (`DeleteFollow(followerID, ownerID)`) and
  clearing its follow notifications; idempotent. Endpoint `DELETE /me/followers/{userId}` (204).
- **Editable display_name** (`auth_uc.SetDisplayName`, `UserRepository.UpdateDisplayName`):
  rides the existing `updateSettings` path (`tracking.yaml`). Trim; blank → store `""` (reset,
  so `displayNameOf` falls back to the derived default); else enforce 1..50 runes else
  `ErrInvalidName`→400. The `Settings` response echoes the effective name. ⚠️ Known inherited
  mismatch (intentional): after a reset, `displayNameOf` falls back to the email local part
  while `actorName` (notification payloads) falls back to `Пользователь #id` — labels may
  differ across surfaces; acceptable for now.
- **Events-in-transaction**: `activity_events` rows are written in the *same* `Transaction`
  as the tracking change that produced them — tracking repo mutations take variadic
  `events ...entity.ActivityEvent` and call the package helper `insertActivityEvents(tx,…)`,
  so the feed can never drift from reality. finished_season/finished_show detection lives in
  the usecase, not the repo. `ActivityRepository.InsertEvents` is the off-hot-path writer
  (used by `cmd/backfill-activity`, an idempotent history seeder).
- **Feeds**: `feed_uc` does read-time grouping (consecutive watched_episode of one
  `(actor,show)` within `Social.FeedGroupWindow`) + keyset pagination (opaque base64
  `created_at|id` cursor, `created_at DESC, id DESC`); `eventIter` fetches further pages so a
  long binge still fills a page. Home feed = accepted followees; profile feed gated by
  `CanViewProfile`.
- **Notifications = event + deliveries** (migration `0028_notification_deliveries`): a
  `notifications` row is ONE logical event and the in-app feed entry (`ListForUser` shows it
  once); per-channel outbox delivery lives in `notification_deliveries` (channel, status,
  scheduled_for, sent_at, `UNIQUE(notification_id, channel)`, `ON DELETE CASCADE`).
  `EnqueueNotification(ctx, n, channels)` inserts the event (deduped by `dedupe_key`) and, when
  newly created, one pending delivery per channel — all in one tx; **no channels → feed-only**
  (still in the feed, never dispatched). This removed the old per-channel feed dupes (the scanner
  used to write one `notifications` row per channel). The sender's `pendingQueries` now join
  `notification_deliveries → notifications → users`; `PendingNotification.ID` is a *delivery* id,
  and `MarkSent`/`MarkFailed` update the delivery. `notifications` no longer has
  channel/status/scheduled_for/sent_at columns. `NotificationItem` API dropped `status` (an event
  has no single delivery status). **Dead targets**: when a channel returns
  `notify.UnregisteredTargetError` (APNs 410/Unregistered or FCM 404/UNREGISTERED — NOT APNs
  `BadDeviceToken`, which usually means an env/topic misconfig and must not mass-wipe), the sender
  calls `SenderRepo.ClearTarget(channel, target)` to null that token/chat (matched by value) so it
  stops being retried; the device re-registers on next launch. No automatic re-send of the failed
  delivery (would need to re-queue on re-registration — deliberately out of scope).
- **User display name**: `entity.User.DisplayLabel()` (display_name → email local part →
  `Пользователь #id`) is the single source for the shown name — used by both `displayNameOf` (API
  responses) and `actorName` (notification payloads) so a user is labelled identically everywhere.
- **Backfill (`cmd/backfill-notifications`, `usecase/backfill_notifications.go`)**: one-shot,
  idempotent seeder that reserves the scanner's dedupe keys (`release:*`/`finale:*`) for
  already-aired episodes as feed markers with **no deliveries** (and pre-marked read), so turning
  on a previously-broken channel (APNs tokens existed but the `apns_token` column tag was wrong →
  no apns rows were ever created) doesn't fire a catch-up push burst on first run. Run once before
  enabling; window = `NOTIFIER_LOOKBACK` (the scanner never looks back further). Mirrors
  `cmd/backfill-activity`.
- **Follow notifications**: three types — `follow_request` (→ followee, on a pending request),
  `follow_new` (→ followee, when a public profile is followed and accepted instantly), and
  `follow_accepted` (→ follower, on approval). All go through `SocialUsecase.emitFollow`, which
  (1) honours the recipient's `social_follows` pref (one toggle gates all three — skip if off) and
  (2) delivers to every channel the recipient has linked (`followChannels`: telegram+apns+fcm),
  feed-only if none. `DedupeKey`+`Payload` set explicitly; unfollow/reject/block/remove-follower
  clear the rows (`DeleteByDedupeKeys`, all three dedupe bases, deliveries cascade) so a re-follow
  re-fires.

## Commands

```
make build | test | lint | generate
make migrate-up | migrate-down | migrate-new name=xxx
make up | down            # docker compose (infra; add --profile app for services)
```

Tests require Docker (testcontainers). `go test ./...` — container-backed suites skip
gracefully if Docker is unavailable.
