-- +goose Up
ALTER TABLE notifications ADD COLUMN read_at TIMESTAMPTZ;
-- Partial index to serve the "unread count" / unread filter cheaply.
CREATE INDEX notifications_user_unread_idx ON notifications (user_id) WHERE read_at IS NULL;

-- +goose Down
DROP INDEX notifications_user_unread_idx;
ALTER TABLE notifications DROP COLUMN read_at;
