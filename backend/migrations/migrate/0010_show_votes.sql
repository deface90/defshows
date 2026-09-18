-- +goose Up
-- TMDB vote aggregates mirrored with the catalog entry (shown as a rating badge).
ALTER TABLE shows ADD COLUMN vote_average double precision NOT NULL DEFAULT 0;
ALTER TABLE shows ADD COLUMN vote_count bigint NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE shows DROP COLUMN vote_count;
ALTER TABLE shows DROP COLUMN vote_average;
