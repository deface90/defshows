-- +goose Up
-- Split delivery out of the notification event: one notifications row = one feed entry,
-- N notification_deliveries rows = per-channel outbox. Removes the per-channel feed dupes
-- (the old model wrote one notifications row per channel) while still delivering to every
-- linked channel. A notification with no deliveries is feed-only.
CREATE TABLE notification_deliveries (
    id              BIGSERIAL   PRIMARY KEY,
    notification_id BIGINT      NOT NULL REFERENCES notifications (id) ON DELETE CASCADE,
    channel         TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'failed')),
    scheduled_for   TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at         TIMESTAMPTZ,
    UNIQUE (notification_id, channel)
);
CREATE INDEX notification_deliveries_pending_idx
    ON notification_deliveries (channel, status, scheduled_for);

-- Collapse the old per-channel episode/season rows into one event each (keep the lowest id),
-- then rewrite their dedupe_key to the new channel-less form the scanner now emits
-- ('<tag>:<user>:<episode>'). Without this the scanner would see every recently-aired,
-- unwatched episode as new again (its old key carried a ':<channel>' suffix) and re-notify.
-- Follow keys were already channel-less, so the regex below leaves them untouched.
DELETE FROM notifications a
USING notifications b
WHERE a.episode_id IS NOT NULL
  AND a.user_id = b.user_id
  AND a.type = b.type
  AND a.episode_id = b.episode_id
  AND a.id > b.id;

UPDATE notifications
SET dedupe_key = split_part(dedupe_key, ':', 1) || ':' || user_id || ':' || episode_id
WHERE episode_id IS NOT NULL
  AND dedupe_key ~ '^[a-z]+:[0-9]+:[0-9]+:(telegram|apns|fcm)$';

-- Carry existing deliverable notifications over (old in_app rows were feed-only → skipped).
INSERT INTO notification_deliveries (notification_id, channel, status, scheduled_for, sent_at)
SELECT id, channel, status, scheduled_for, sent_at
FROM notifications
WHERE channel <> 'in_app';

DROP INDEX IF EXISTS notifications_status_idx;
ALTER TABLE notifications
    DROP COLUMN channel,
    DROP COLUMN status,
    DROP COLUMN scheduled_for,
    DROP COLUMN sent_at;

-- +goose Down
ALTER TABLE notifications
    ADD COLUMN channel       TEXT        NOT NULL DEFAULT 'telegram',
    ADD COLUMN status        TEXT        NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'failed')),
    ADD COLUMN scheduled_for TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN sent_at       TIMESTAMPTZ;
CREATE INDEX notifications_status_idx ON notifications (status);

-- Best-effort restore from the first delivery per notification.
UPDATE notifications n
SET channel = d.channel, status = d.status, scheduled_for = d.scheduled_for, sent_at = d.sent_at
FROM (
    SELECT DISTINCT ON (notification_id) notification_id, channel, status, scheduled_for, sent_at
    FROM notification_deliveries
    ORDER BY notification_id, id
) d
WHERE d.notification_id = n.id;

DROP TABLE notification_deliveries;
