-- +goose NO TRANSACTION

-- +goose Up
ALTER TYPE folder_status ADD VALUE IF NOT EXISTS 'deleting';

CREATE INDEX IF NOT EXISTS idx_files_deleting
    ON files (bucket_id, deleted_by)
    WHERE status = 'deleting' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_folders_deleting
    ON folders (bucket_id, deleted_by)
    WHERE status = 'deleting' AND deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_folders_deleting;
DROP INDEX IF EXISTS idx_files_deleting;

ALTER TABLE folders ALTER COLUMN status DROP DEFAULT;
ALTER TABLE folders ALTER COLUMN status TYPE TEXT;
DROP TYPE folder_status;
CREATE TYPE folder_status AS ENUM ('created', 'deleted', 'restoring');
ALTER TABLE folders
    ALTER COLUMN status TYPE folder_status USING status::folder_status,
    ALTER COLUMN status SET DEFAULT 'created'::folder_status;
