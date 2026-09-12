-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;

CREATE TABLE shares_new
    (
        id TEXT PRIMARY KEY,
        path TEXT NOT NULL,
        name TEXT NOT NULL,
        bucket_id TEXT NOT NULL,
        folder_id TEXT,
        expires_at DATETIME,
        max_views INTEGER,
        current_views INTEGER NOT NULL DEFAULT 0,
        hashed_password TEXT,
        type TEXT NOT NULL,
        allow_upload INTEGER NOT NULL DEFAULT 0,
        max_uploads INTEGER,
        current_uploads INTEGER NOT NULL DEFAULT 0,
        max_upload_size INTEGER,
        created_by TEXT NOT NULL,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME,
        CONSTRAINT fk_shares_bucket_id
            FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_shares_folder_id
            FOREIGN KEY (folder_id) REFERENCES folders (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_shares_created_by
            FOREIGN KEY (created_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT chk_shares_upload_type
            CHECK (NOT (allow_upload = 1 AND type = 'files')),
        CONSTRAINT chk_shares_current_views
            CHECK (current_views >= 0),
        CONSTRAINT chk_shares_current_uploads
            CHECK (current_uploads >= 0)
    );
INSERT INTO shares_new SELECT id, id, name, bucket_id, folder_id, expires_at, max_views,
    current_views, hashed_password, type, allow_upload, max_uploads, current_uploads,
    max_upload_size, created_by, created_at, updated_at, deleted_at FROM shares;
DROP TABLE shares;
ALTER TABLE shares_new RENAME TO shares;
CREATE INDEX idx_shares_bucket_id ON shares (bucket_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_shares_path ON shares (path) WHERE deleted_at IS NULL;

COMMIT;
PRAGMA foreign_keys=ON;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;

CREATE TABLE shares_new
    (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        bucket_id TEXT NOT NULL,
        folder_id TEXT,
        expires_at DATETIME,
        max_views INTEGER,
        current_views INTEGER NOT NULL DEFAULT 0,
        hashed_password TEXT,
        type TEXT NOT NULL,
        allow_upload INTEGER NOT NULL DEFAULT 0,
        max_uploads INTEGER,
        current_uploads INTEGER NOT NULL DEFAULT 0,
        max_upload_size INTEGER,
        created_by TEXT NOT NULL,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME,
        CONSTRAINT fk_shares_bucket_id
            FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_shares_folder_id
            FOREIGN KEY (folder_id) REFERENCES folders (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_shares_created_by
            FOREIGN KEY (created_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT chk_shares_upload_type
            CHECK (NOT (allow_upload = 1 AND type = 'files')),
        CONSTRAINT chk_shares_current_views
            CHECK (current_views >= 0),
        CONSTRAINT chk_shares_current_uploads
            CHECK (current_uploads >= 0)
    );
INSERT INTO shares_new SELECT id, name, bucket_id, folder_id, expires_at, max_views,
    current_views, hashed_password, type, allow_upload, max_uploads, current_uploads,
    max_upload_size, created_by, created_at, updated_at, deleted_at FROM shares;
DROP TABLE shares;
ALTER TABLE shares_new RENAME TO shares;
CREATE INDEX idx_shares_bucket_id ON shares (bucket_id) WHERE deleted_at IS NULL;

COMMIT;
PRAGMA foreign_keys=ON;
-- +goose StatementEnd
