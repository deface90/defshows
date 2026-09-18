-- +goose Up
CREATE TABLE dubbing_studios (
    id       BIGSERIAL PRIMARY KEY,
    name     TEXT    NOT NULL UNIQUE,
    site_url TEXT    NOT NULL DEFAULT '',
    active   BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE user_shows (
    id                BIGSERIAL PRIMARY KEY,
    user_id           BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    show_id           BIGINT      NOT NULL REFERENCES shows (id) ON DELETE CASCADE,
    status            TEXT        NOT NULL DEFAULT 'watching'
        CHECK (status IN ('watching', 'plan_to_watch', 'on_hold', 'completed', 'dropped')),
    favorite          BOOLEAN     NOT NULL DEFAULT false,
    preferred_dubbing TEXT        NOT NULL DEFAULT '',
    added_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, show_id)
);
CREATE INDEX user_shows_user_id_idx ON user_shows (user_id);

CREATE TABLE user_episodes (
    id           BIGSERIAL PRIMARY KEY,
    user_show_id BIGINT      NOT NULL REFERENCES user_shows (id) ON DELETE CASCADE,
    episode_id   BIGINT      NOT NULL REFERENCES episodes (id) ON DELETE CASCADE,
    watched      BOOLEAN     NOT NULL DEFAULT true,
    watched_at   TIMESTAMPTZ,
    rating       INT,
    UNIQUE (user_show_id, episode_id)
);

CREATE TABLE user_show_links (
    id           BIGSERIAL PRIMARY KEY,
    user_show_id BIGINT NOT NULL REFERENCES user_shows (id) ON DELETE CASCADE,
    kind         TEXT   NOT NULL CHECK (kind IN ('dubbing', 'download', 'streaming')),
    label        TEXT   NOT NULL DEFAULT '',
    url          TEXT   NOT NULL
);
CREATE INDEX user_show_links_user_show_id_idx ON user_show_links (user_show_id);

-- +goose Down
DROP TABLE user_show_links;
DROP TABLE user_episodes;
DROP TABLE user_shows;
DROP TABLE dubbing_studios;
