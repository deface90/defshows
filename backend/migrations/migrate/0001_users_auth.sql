-- +goose Up
CREATE TABLE users (
    id               BIGSERIAL PRIMARY KEY,
    email            TEXT UNIQUE,
    password_hash    TEXT,
    display_name     TEXT        NOT NULL DEFAULT '',
    role             TEXT        NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    timezone         TEXT        NOT NULL DEFAULT 'UTC',
    telegram_chat_id BIGINT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- telegram_chat_id must be unique when set (many NULLs allowed)
CREATE UNIQUE INDEX users_telegram_chat_id_key
    ON users (telegram_chat_id) WHERE telegram_chat_id IS NOT NULL;

CREATE TABLE user_identities (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider         TEXT        NOT NULL,
    provider_user_id TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_user_id)
);
CREATE INDEX user_identities_user_id_idx ON user_identities (user_id);

CREATE TABLE refresh_tokens (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash    TEXT        NOT NULL UNIQUE,
    family_id     UUID        NOT NULL,
    prev_token_id BIGINT      REFERENCES refresh_tokens (id) ON DELETE SET NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked_at    TIMESTAMPTZ,
    user_agent    TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_family_id_idx ON refresh_tokens (family_id);

-- +goose Down
DROP TABLE refresh_tokens;
DROP TABLE user_identities;
DROP TABLE users;
