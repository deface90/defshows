-- +goose Up
CREATE TABLE shows (
    id                    BIGSERIAL PRIMARY KEY,
    tmdb_id               BIGINT      NOT NULL UNIQUE,
    title                 TEXT        NOT NULL,
    original_title        TEXT        NOT NULL DEFAULT '',
    overview              TEXT        NOT NULL DEFAULT '',
    poster_key            TEXT,
    backdrop_key          TEXT,
    status                TEXT        NOT NULL DEFAULT '',
    in_production         BOOLEAN     NOT NULL DEFAULT false,
    first_air_date        DATE,
    last_air_date         DATE,
    next_episode_id       BIGINT,
    next_episode_air_date DATE,
    last_episode_air_date DATE,
    airing_status         TEXT        NOT NULL DEFAULT 'not_started'
        CHECK (airing_status IN ('not_started', 'airing', 'between_seasons', 'ended')),
    original_language     TEXT        NOT NULL DEFAULT '',
    popularity            DOUBLE PRECISION NOT NULL DEFAULT 0,
    last_synced_at        TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE seasons (
    id            BIGSERIAL PRIMARY KEY,
    show_id       BIGINT NOT NULL REFERENCES shows (id) ON DELETE CASCADE,
    tmdb_id       BIGINT,
    season_number INT    NOT NULL,
    name          TEXT   NOT NULL DEFAULT '',
    overview      TEXT   NOT NULL DEFAULT '',
    air_date      DATE,
    episode_count INT    NOT NULL DEFAULT 0,
    poster_key    TEXT,
    UNIQUE (show_id, season_number)
);

CREATE TABLE episodes (
    id             BIGSERIAL PRIMARY KEY,
    season_id      BIGINT NOT NULL REFERENCES seasons (id) ON DELETE CASCADE,
    show_id        BIGINT NOT NULL REFERENCES shows (id) ON DELETE CASCADE,
    tmdb_id        BIGINT,
    season_number  INT    NOT NULL,
    episode_number INT    NOT NULL,
    name           TEXT   NOT NULL DEFAULT '',
    overview       TEXT   NOT NULL DEFAULT '',
    air_date       DATE,
    runtime        INT,
    still_key      TEXT,
    UNIQUE (show_id, season_number, episode_number)
);
CREATE INDEX episodes_air_date_idx ON episodes (air_date);
CREATE INDEX episodes_show_id_idx ON episodes (show_id);

-- shows.next_episode_id references episodes; add the FK after episodes exists.
ALTER TABLE shows
    ADD CONSTRAINT shows_next_episode_fk
        FOREIGN KEY (next_episode_id) REFERENCES episodes (id) ON DELETE SET NULL;

CREATE TABLE genres (
    id      BIGSERIAL PRIMARY KEY,
    tmdb_id BIGINT NOT NULL UNIQUE,
    name    TEXT   NOT NULL
);

CREATE TABLE show_genres (
    show_id  BIGINT NOT NULL REFERENCES shows (id) ON DELETE CASCADE,
    genre_id BIGINT NOT NULL REFERENCES genres (id) ON DELETE CASCADE,
    PRIMARY KEY (show_id, genre_id)
);

-- +goose Down
DROP TABLE show_genres;
DROP TABLE genres;
ALTER TABLE shows DROP CONSTRAINT shows_next_episode_fk;
DROP TABLE episodes;
DROP TABLE seasons;
DROP TABLE shows;
