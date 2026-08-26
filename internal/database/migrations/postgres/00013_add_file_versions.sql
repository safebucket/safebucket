-- +goose Up
-- +goose StatementBegin

CREATE TABLE file_versions
    (
        id uuid
            PRIMARY KEY DEFAULT gen_random_uuid(),
        file_id uuid NOT NULL,
        version INTEGER NOT NULL,
        size BIGINT NOT NULL DEFAULT 0,
        status file_status NOT NULL,
        uploaded_by uuid,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

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
CREATE UNIQUE INDEX idx_file_versions_file_version ON file_versions (file_id, version);

ALTER TABLE files ADD COLUMN current_version_id uuid;

INSERT INTO file_versions (id, file_id, version, size, status, uploaded_by, created_at)
SELECT id, id, 1, size,
    status,
    NULL, created_at
FROM files;

UPDATE files SET current_version_id = id;

ALTER TABLE files
    ADD CONSTRAINT fk_files_current_version_id
        FOREIGN KEY (current_version_id) REFERENCES file_versions (id) ON UPDATE CASCADE ON DELETE SET NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE files DROP CONSTRAINT IF EXISTS fk_files_current_version_id;
DROP TABLE IF EXISTS file_versions;
ALTER TABLE files DROP COLUMN IF EXISTS current_version_id;

-- +goose StatementEnd

-- TODO(RBE): drop duplicated columns from files (size) once versioning replaces their usage.
