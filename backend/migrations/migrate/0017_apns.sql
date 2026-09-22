-- +goose Up
ALTER TABLE users ADD COLUMN apns_device_token TEXT UNIQUE;

-- +goose Down
ALTER TABLE users DROP COLUMN apns_device_token;
