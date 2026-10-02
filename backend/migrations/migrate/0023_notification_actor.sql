-- +goose Up
-- Follow notifications (follow_request / follow_accepted) reference the acting user
-- (the follower). Existing notification types leave this NULL.
ALTER TABLE notifications ADD COLUMN actor_id BIGINT REFERENCES users (id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE notifications DROP COLUMN actor_id;
