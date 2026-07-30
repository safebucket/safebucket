-- +goose Up
ALTER TABLE users DROP COLUMN is_initialized;

-- +goose Down
ALTER TABLE users ADD COLUMN is_initialized INTEGER NOT NULL DEFAULT 0;
