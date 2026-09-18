-- +goose Up
CREATE TABLE notification_prefs (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT  NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    episode_release BOOLEAN NOT NULL DEFAULT true,
    season_start    BOOLEAN NOT NULL DEFAULT true,
    weekly_digest   BOOLEAN NOT NULL DEFAULT false,
    channel         TEXT    NOT NULL DEFAULT 'telegram',
    lead_time_hours INT     NOT NULL DEFAULT 24
);

CREATE TABLE notifications (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type          TEXT        NOT NULL,
    show_id       BIGINT      REFERENCES shows (id) ON DELETE CASCADE,
    episode_id    BIGINT      REFERENCES episodes (id) ON DELETE CASCADE,
    channel       TEXT        NOT NULL DEFAULT 'telegram',
    status        TEXT        NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'failed')),
    scheduled_for TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at       TIMESTAMPTZ,
    dedupe_key    TEXT        NOT NULL UNIQUE,
    payload       TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX notifications_user_id_idx ON notifications (user_id);
CREATE INDEX notifications_status_idx ON notifications (status);

CREATE TABLE telegram_link_tokens (
    token      TEXT        PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE telegram_link_tokens;
DROP TABLE notifications;
DROP TABLE notification_prefs;
