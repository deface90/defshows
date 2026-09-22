-- +goose Up
-- Password-reset tokens: single-use, short-lived, delivered by email.
CREATE TABLE password_reset_tokens (
  id         bigserial PRIMARY KEY,
  user_id    bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  used_at    timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_password_reset_tokens_user ON password_reset_tokens(user_id);

-- +goose Down
DROP TABLE password_reset_tokens;
