-- +goose Up
-- Notify about new followers / follow requests / accepted requests. One toggle
-- gates every follow (social) notification; defaults on.
ALTER TABLE notification_prefs ADD COLUMN social_follows BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE notification_prefs DROP COLUMN social_follows;
