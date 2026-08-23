-- +goose Up
-- +goose StatementBegin

ALTER TABLE files ADD COLUMN current_version_id TEXT;
ALTER TABLE files ADD COLUMN last_version_number INTEGER NOT NULL DEFAULT 0;

CREATE TABLE file_versions
    (
        id TEXT PRIMARY KEY,
        file_id TEXT NOT NULL,
        version_number INTEGER NOT NULL,
        size INTEGER NOT NULL DEFAULT 0,
        status TEXT NOT NULL,
        uploaded_by TEXT,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        CONSTRAINT fk_file_versions_file_id
            FOREIGN KEY (file_id) REFERENCES files (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_file_versions_uploaded_by
            FOREIGN KEY (uploaded_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL,
        CONSTRAINT chk_file_versions_size_positive
            CHECK (size >= 0)
    );

CREATE INDEX idx_file_versions_file_id ON file_versions (file_id);
CREATE UNIQUE INDEX idx_file_versions_file_number ON file_versions (file_id, version_number);

INSERT INTO file_versions (id, file_id, version_number, size, status, uploaded_by, created_at, updated_at)
SELECT id, id, 1, COALESCE(size, 0),
    status,
    NULL, created_at, created_at
FROM files;

UPDATE files SET current_version_id = id, last_version_number = 1;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS file_versions;
ALTER TABLE files DROP COLUMN current_version_id;
ALTER TABLE files DROP COLUMN last_version_number;

-- +goose StatementEnd
