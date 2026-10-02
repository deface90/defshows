-- +goose Up
CREATE TABLE blocks (
    id         BIGSERIAL   PRIMARY KEY,
    blocker_id BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    blocked_id BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (blocker_id <> blocked_id),
    UNIQUE (blocker_id, blocked_id)
);
CREATE INDEX blocks_blocked_idx ON blocks (blocked_id);

-- +goose Down
DROP TABLE blocks;
