-- +goose Up
-- +goose StatementBegin

CREATE TABLE sharing_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL UNIQUE REFERENCES files(id) ON DELETE CASCADE,
    expires_at TIMESTAMP,
    max_downloads INTEGER CHECK (max_downloads IS NULL OR max_downloads > 0),
    download_count INTEGER NOT NULL DEFAULT 0 CHECK (download_count >= 0),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sharing_options_file_id ON sharing_options(file_id);
CREATE INDEX idx_sharing_options_expires_at ON sharing_options(expires_at)
    WHERE expires_at IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_sharing_options_expires_at;
DROP INDEX IF EXISTS idx_sharing_options_file_id;
DROP TABLE IF EXISTS sharing_options;

-- +goose StatementEnd
