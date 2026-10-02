-- +goose Up
CREATE TABLE activity_events (
    id            BIGSERIAL   PRIMARY KEY,
    user_id       BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type          TEXT        NOT NULL
        CHECK (type IN ('watched_episode', 'finished_season', 'finished_show', 'added_show', 'rated_show')),
    show_id       BIGINT      NOT NULL REFERENCES shows (id) ON DELETE CASCADE,
    season_number INT,
    episode_id    BIGINT      REFERENCES episodes (id) ON DELETE SET NULL,
    rating        SMALLINT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Serves both the profile feed (single user) and the home feed (user_id IN (...)).
CREATE INDEX activity_events_user_created_idx ON activity_events (user_id, created_at DESC, id DESC);

-- +goose Down
DROP TABLE activity_events;
