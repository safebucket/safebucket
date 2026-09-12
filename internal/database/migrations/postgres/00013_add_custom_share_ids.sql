-- +goose Up
-- +goose StatementBegin

ALTER TABLE shares ADD COLUMN custom_id TEXT;
CREATE UNIQUE INDEX idx_shares_custom_id ON shares (custom_id) WHERE deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_shares_custom_id;
ALTER TABLE shares DROP COLUMN custom_id;

-- +goose StatementEnd
