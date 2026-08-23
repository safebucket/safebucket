-- +goose Up
-- +goose StatementBegin

ALTER TABLE files ADD COLUMN current_version_id uuid;
ALTER TABLE files ADD COLUMN last_version_number INTEGER NOT NULL DEFAULT 0;

CREATE TABLE file_versions
    (
        id uuid
            PRIMARY KEY DEFAULT gen_random_uuid(),
        file_id uuid NOT NULL,
        version_number INTEGER NOT NULL,
        size BIGINT NOT NULL DEFAULT 0,
        status file_status NOT NULL,
        uploaded_by uuid,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

        -- Foreign Keys
        CONSTRAINT fk_file_versions_file_id
            FOREIGN KEY (file_id) REFERENCES files (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_file_versions_uploaded_by
            FOREIGN KEY (uploaded_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL,

        -- Constraints
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
ALTER TABLE files DROP COLUMN IF EXISTS current_version_id;
ALTER TABLE files DROP COLUMN IF EXISTS last_version_number;

-- +goose StatementEnd
