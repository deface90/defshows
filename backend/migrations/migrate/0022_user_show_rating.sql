-- +goose Up
ALTER TABLE user_shows ADD COLUMN rating SMALLINT CHECK (rating BETWEEN 1 AND 10);

-- +goose Down
ALTER TABLE user_shows DROP COLUMN rating;
