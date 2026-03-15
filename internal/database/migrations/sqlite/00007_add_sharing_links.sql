-- +goose Up
-- +goose StatementBegin

CREATE TABLE shares
    (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        bucket_id TEXT NOT NULL,
        folder_id TEXT,
        expires_at DATETIME,
        max_views INTEGER,
        current_views INTEGER NOT NULL DEFAULT 0,
        hashed_password TEXT,
        type TEXT NOT NULL CHECK(type IN ('files', 'folder', 'bucket')),
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

CREATE INDEX idx_shares_bucket_id ON shares (bucket_id) WHERE deleted_at IS NULL;

CREATE TABLE share_files
    (
        id TEXT PRIMARY KEY,
        share_id TEXT NOT NULL,
        file_id TEXT NOT NULL,

        CONSTRAINT fk_share_files_share_id
            FOREIGN KEY (share_id) REFERENCES shares (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_share_files_file_id
            FOREIGN KEY (file_id) REFERENCES files (id) ON UPDATE CASCADE ON DELETE CASCADE,

        CONSTRAINT idx_share_files_unique
            UNIQUE (share_id, file_id)
    );

CREATE INDEX idx_share_files_file_id ON share_files (file_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS share_files;
DROP TABLE IF EXISTS shares;

-- +goose StatementEnd
