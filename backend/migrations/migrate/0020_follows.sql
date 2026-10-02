-- +goose Up
CREATE TABLE follows (
    id          BIGSERIAL   PRIMARY KEY,
    follower_id BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    followee_id BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status      TEXT        NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (follower_id <> followee_id),
    UNIQUE (follower_id, followee_id)
);
CREATE INDEX follows_followee_status_idx ON follows (followee_id, status);
CREATE INDEX follows_follower_status_idx ON follows (follower_id, status);

-- +goose Down
DROP TABLE follows;
