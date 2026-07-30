-- +goose Up
ALTER TABLE users DROP COLUMN IF EXISTS is_initialized;

-- +goose Down
ALTER TABLE users ADD COLUMN is_initialized BOOLEAN NOT NULL DEFAULT FALSE;
