-- +goose Up
-- +goose StatementBegin
CREATE TABLE folders_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'created' CHECK(status IN ('created', 'deleted', 'restoring')),
    folder_id TEXT,
    bucket_id TEXT NOT NULL,
    deleted_by TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CONSTRAINT fk_folders_folder_id FOREIGN KEY (folder_id) REFERENCES folders_new (id) ON UPDATE CASCADE ON DELETE SET NULL,
    CONSTRAINT fk_folders_bucket_id FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_folders_deleted_by FOREIGN KEY (deleted_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL
);
INSERT INTO folders_new SELECT id, name, CASE WHEN status IS NULL OR status = '' OR status IN ('uploading', 'uploaded') THEN 'created' ELSE status END, folder_id, bucket_id, deleted_by, created_at, updated_at, deleted_at FROM folders;
DROP TABLE folders;
ALTER TABLE folders_new RENAME TO folders;
CREATE INDEX idx_folders_bucket_parent ON folders (bucket_id, folder_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_folders_deleted_by ON folders (deleted_by) WHERE deleted_by IS NOT NULL;
CREATE UNIQUE INDEX idx_folders_unique_name ON folders (bucket_id, COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'), name) WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE folders_old (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT CHECK(status IN ('uploading', 'uploaded', 'deleting', 'deleted', 'restoring')),
    folder_id TEXT,
    bucket_id TEXT NOT NULL,
    deleted_by TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CONSTRAINT fk_folders_folder_id FOREIGN KEY (folder_id) REFERENCES folders_old (id) ON UPDATE CASCADE ON DELETE SET NULL,
    CONSTRAINT fk_folders_bucket_id FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_folders_deleted_by FOREIGN KEY (deleted_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL
);
INSERT INTO folders_old SELECT id, name, CASE WHEN status = 'created' THEN NULL ELSE status END, folder_id, bucket_id, deleted_by, created_at, updated_at, deleted_at FROM folders;
DROP TABLE folders;
ALTER TABLE folders_old RENAME TO folders;
CREATE INDEX idx_folders_bucket_parent ON folders (bucket_id, folder_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_folders_deleted_by ON folders (deleted_by) WHERE deleted_by IS NOT NULL;
CREATE UNIQUE INDEX idx_folders_unique_name ON folders (bucket_id, COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'), name) WHERE deleted_at IS NULL;
-- +goose StatementEnd
