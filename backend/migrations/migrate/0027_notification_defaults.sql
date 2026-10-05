-- +goose Up
ALTER TABLE notification_prefs ALTER COLUMN episode_release SET DEFAULT false;
ALTER TABLE notification_prefs ALTER COLUMN season_finale SET DEFAULT false;

-- +goose Down
ALTER TABLE notification_prefs ALTER COLUMN episode_release SET DEFAULT true;
ALTER TABLE notification_prefs ALTER COLUMN season_finale SET DEFAULT true;
