-- +goose Up
CREATE TABLE show_ratings (
    id         BIGSERIAL PRIMARY KEY,
    show_id    BIGINT      NOT NULL REFERENCES shows (id) ON DELETE CASCADE,
    source     TEXT        NOT NULL,
    value      TEXT        NOT NULL,
    votes      BIGINT,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (show_id, source)
);
CREATE INDEX show_ratings_show_id_idx ON show_ratings (show_id);

-- +goose Down
DROP TABLE show_ratings;
