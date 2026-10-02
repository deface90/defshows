-- +goose Up
CREATE TABLE reports (
    id             BIGSERIAL   PRIMARY KEY,
    reporter_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    target_user_id BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    reason         TEXT        NOT NULL
        CHECK (reason IN ('spam', 'harassment', 'inappropriate', 'other')),
    note           TEXT        NOT NULL DEFAULT '',
    status         TEXT        NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'resolved', 'dismissed')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at    TIMESTAMPTZ,
    resolved_by    BIGINT      REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX reports_status_idx ON reports (status, created_at DESC);

-- +goose Down
DROP TABLE reports;
