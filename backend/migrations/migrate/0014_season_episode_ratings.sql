-- +goose Up
ALTER TABLE seasons ADD COLUMN vote_average double precision NOT NULL DEFAULT 0;
ALTER TABLE episodes ADD COLUMN vote_average double precision NOT NULL DEFAULT 0,
    ADD COLUMN vote_count bigint NOT NULL DEFAULT 0;
-- Refresh existing catalog entries to populate the newly imported ratings.
UPDATE shows SET last_synced_at = NULL;

-- +goose Down
ALTER TABLE episodes DROP COLUMN vote_count, DROP COLUMN vote_average;
ALTER TABLE seasons DROP COLUMN vote_average;
