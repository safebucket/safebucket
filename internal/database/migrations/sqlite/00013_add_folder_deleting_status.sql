-- +goose Up
CREATE INDEX idx_files_deleting
    ON files (bucket_id, deleted_by)
    WHERE status = 'deleting' AND deleted_at IS NULL;

CREATE INDEX idx_folders_deleting
    ON folders (bucket_id, deleted_by)
    WHERE status = 'deleting' AND deleted_at IS NULL;

-- +goose Down
DROP INDEX idx_folders_deleting;
DROP INDEX idx_files_deleting;
