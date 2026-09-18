-- +goose Up
ALTER TABLE user_show_links DROP CONSTRAINT user_show_links_kind_check;
ALTER TABLE user_show_links ADD CONSTRAINT user_show_links_kind_check
    CHECK (kind IN ('dubbing', 'download', 'streaming', 'wiki', 'imdb', 'kinopoisk'));

-- +goose Down
-- NOT VALID preserves existing reference links on rollback while restoring the
-- old restriction for new/updated rows.
ALTER TABLE user_show_links DROP CONSTRAINT user_show_links_kind_check;
ALTER TABLE user_show_links ADD CONSTRAINT user_show_links_kind_check
    CHECK (kind IN ('dubbing', 'download', 'streaming')) NOT VALID;
