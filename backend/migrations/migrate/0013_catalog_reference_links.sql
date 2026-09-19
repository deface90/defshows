-- +goose Up
ALTER TABLE shows ADD COLUMN imdb_url text, ADD COLUMN wikipedia_url text;
-- Prioritize existing titles for enrichment by the regular catalog worker.
UPDATE shows SET last_synced_at = NULL;

-- +goose Down
ALTER TABLE shows DROP COLUMN wikipedia_url, DROP COLUMN imdb_url;
