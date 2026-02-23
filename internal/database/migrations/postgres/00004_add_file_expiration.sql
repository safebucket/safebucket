-- +goose Up
ALTER TABLE files ADD COLUMN expires_at TIMESTAMP;
CREATE INDEX idx_files_expires_at ON files (expires_at) WHERE expires_at IS NOT NULL;

-- +goose Down
ALTER TABLE files DROP COLUMN IF EXISTS expires_at;
DROP INDEX if EXISTS idx_files_expires_at;
