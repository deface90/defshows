-- +goose Up
ALTER TABLE users ADD COLUMN fcm_device_token TEXT UNIQUE;

-- +goose Down
ALTER TABLE users DROP COLUMN fcm_device_token;
