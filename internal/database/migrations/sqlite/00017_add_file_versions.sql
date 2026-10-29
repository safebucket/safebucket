-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;

CREATE TABLE file_versions
    (
        id TEXT PRIMARY KEY,
        file_id TEXT NOT NULL,
        version INTEGER NOT NULL,
        size INTEGER NOT NULL DEFAULT 0,
        status TEXT NOT NULL,
        uploaded_by TEXT,
        cleanup_after DATETIME,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        CONSTRAINT fk_file_versions_file_id
            FOREIGN KEY (file_id) REFERENCES files (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_file_versions_uploaded_by
            FOREIGN KEY (uploaded_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL,
        CONSTRAINT chk_file_versions_size_positive
            CHECK (size >= 0)
    );

CREATE INDEX idx_file_versions_file_id ON file_versions (file_id);
CREATE UNIQUE INDEX idx_file_versions_file_version ON file_versions (file_id, version);
CREATE UNIQUE INDEX idx_file_versions_pending ON file_versions (file_id) WHERE status = 'uploading';

INSERT INTO file_versions (id, file_id, version, size, status, uploaded_by, created_at)
SELECT id, id, 1, COALESCE(size, 0),
    CASE WHEN status IN ('deleted', 'deleting', 'restoring') THEN 'uploaded' ELSE status END,
    NULL, created_at
FROM files;

CREATE TABLE files_new
    (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        extension TEXT,
        status TEXT,
        bucket_id TEXT NOT NULL,
        folder_id TEXT,
        current_version_id TEXT,
        deleted_by TEXT,
        expires_at DATETIME,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME,
        CONSTRAINT fk_files_bucket_id
            FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_files_folder_id
            FOREIGN KEY (folder_id) REFERENCES folders (id) ON UPDATE CASCADE ON DELETE SET NULL,
        CONSTRAINT fk_files_current_version_id
            FOREIGN KEY (current_version_id) REFERENCES file_versions (id) ON UPDATE CASCADE ON DELETE SET NULL,
        CONSTRAINT fk_files_deleted_by
            FOREIGN KEY (deleted_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL
    );

INSERT INTO files_new (id, name, extension, status, bucket_id, folder_id, current_version_id,
    deleted_by, expires_at, created_at, updated_at, deleted_at)
SELECT id, name, extension, status, bucket_id, folder_id,
    CASE WHEN status = 'uploading' THEN NULL ELSE id END,
    deleted_by, expires_at, created_at, updated_at, deleted_at
FROM files;

DROP TABLE files;
ALTER TABLE files_new RENAME TO files;

CREATE INDEX idx_files_bucket_folder ON files (bucket_id, folder_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_deleted_by ON files (deleted_by) WHERE deleted_by IS NOT NULL;
CREATE UNIQUE INDEX idx_files_unique_name ON files (bucket_id,
    COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'), name) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_expires_at ON files (expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_files_deleting
    ON files (bucket_id, deleted_by)
    WHERE status = 'deleting' AND deleted_at IS NULL;

COMMIT;
PRAGMA foreign_keys=ON;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;

CREATE TABLE files_old
    (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        extension TEXT,
        status TEXT,
        bucket_id TEXT NOT NULL,
        folder_id TEXT,
        size INTEGER NOT NULL DEFAULT 0,
        deleted_by TEXT,
        expires_at DATETIME,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME,
        CONSTRAINT fk_files_bucket_id
            FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_files_folder_id
            FOREIGN KEY (folder_id) REFERENCES folders (id) ON UPDATE CASCADE ON DELETE SET NULL,
        CONSTRAINT fk_files_deleted_by
            FOREIGN KEY (deleted_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL,
        CONSTRAINT chk_files_size_positive
            CHECK (size >= 0)
    );

INSERT INTO files_old (id, name, extension, status, bucket_id, folder_id, size,
    deleted_by, expires_at, created_at, updated_at, deleted_at)
SELECT files.id, files.name, files.extension, files.status, files.bucket_id, files.folder_id,
    COALESCE(file_versions.size, 0), files.deleted_by, files.expires_at,
    files.created_at, files.updated_at, files.deleted_at
FROM files
LEFT JOIN file_versions ON file_versions.id = COALESCE(files.current_version_id, files.id);

DROP TABLE files;
ALTER TABLE files_old RENAME TO files;

CREATE INDEX idx_files_bucket_folder ON files (bucket_id, folder_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_deleted_by ON files (deleted_by) WHERE deleted_by IS NOT NULL;
CREATE UNIQUE INDEX idx_files_unique_name ON files (bucket_id,
    COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'), name) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_expires_at ON files (expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_files_deleting
    ON files (bucket_id, deleted_by)
    WHERE status = 'deleting' AND deleted_at IS NULL;

DROP TABLE file_versions;

COMMIT;
PRAGMA foreign_keys=ON;
-- +goose StatementEnd
