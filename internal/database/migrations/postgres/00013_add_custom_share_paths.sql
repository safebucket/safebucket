-- +goose Up
-- +goose StatementBegin

ALTER TABLE shares ADD COLUMN path TEXT;
UPDATE shares SET path = id::text;
ALTER TABLE shares ALTER COLUMN path SET NOT NULL;
CREATE UNIQUE INDEX idx_shares_path ON shares (path) WHERE deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_shares_path;
ALTER TABLE shares DROP COLUMN path;

-- +goose StatementEnd
