-- +goose Up
-- Profile visibility: private by default (existing and new users), owner opts in.
ALTER TABLE users ADD COLUMN is_public boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE users DROP COLUMN is_public;
