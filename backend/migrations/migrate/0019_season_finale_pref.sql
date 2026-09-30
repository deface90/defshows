-- +goose Up
-- Notify when a season's finale airs. Defaults on, like season_start; the
-- lead_time_hours setting does not apply (it fires on release, not before).
ALTER TABLE notification_prefs ADD COLUMN season_finale BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE notification_prefs DROP COLUMN season_finale;
