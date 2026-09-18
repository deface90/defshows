-- +goose Up
CREATE TABLE notes (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    show_id        BIGINT      NOT NULL REFERENCES shows (id) ON DELETE CASCADE,
    scope          TEXT        NOT NULL CHECK (scope IN ('show', 'season', 'episode')),
    season_number  INT,
    episode_number INT,
    body           TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX notes_user_show_idx ON notes (user_id, show_id);

CREATE TABLE recaps (
    id             BIGSERIAL PRIMARY KEY,
    show_id        BIGINT      NOT NULL REFERENCES shows (id) ON DELETE CASCADE,
    scope          TEXT        NOT NULL CHECK (scope IN ('show', 'season', 'episode')),
    season_number  INT,
    episode_number INT,
    body           TEXT        NOT NULL,
    language       TEXT        NOT NULL DEFAULT 'ru',
    model          TEXT        NOT NULL DEFAULT '',
    generated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Uniqueness with NULL season/episode treated as -1 so show-scope recaps dedupe.
CREATE UNIQUE INDEX recaps_unique_idx ON recaps
    (show_id, scope, COALESCE(season_number, -1), COALESCE(episode_number, -1), language);

-- +goose Down
DROP TABLE recaps;
DROP TABLE notes;
