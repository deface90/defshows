-- +goose Up
-- Cache for the personal home's heavy part (taste profile + recommendations),
-- recomputed on demand past a TTL. Progress blocks stay live via /me/shows.
CREATE TABLE user_home_cache (
    user_id     bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    payload     jsonb NOT NULL,
    computed_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE user_home_cache;
