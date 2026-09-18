-- +goose Up
-- One-off cleanup: remove already-imported TMDB "Specials" (season 0). Going
-- forward the importer skips season 0 entirely. Deleting the season 0 rows
-- cascades to their episodes (episodes.season_id ON DELETE CASCADE) and in turn
-- to any user watch marks (user_episodes.episode_id ON DELETE CASCADE).
-- Delete stray season-0 episodes first in case any were attached to a non-zero
-- season row, then the season rows themselves.
DELETE FROM episodes WHERE season_number = 0;
DELETE FROM seasons WHERE season_number = 0;

-- +goose Down
-- Irreversible data cleanup: specials are re-created on the next catalog sync
-- only if the importer is changed to keep them. Nothing to roll back.
SELECT 1;
