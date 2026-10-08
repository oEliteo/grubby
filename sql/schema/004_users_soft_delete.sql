-- +goose Up
ALTER TABLE users
ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;

-- +goose Down
ALTER TABLE users
DROP COLUMN deleted_at;
