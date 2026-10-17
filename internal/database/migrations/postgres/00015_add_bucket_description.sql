-- +goose Up
-- +goose StatementBegin

ALTER TABLE buckets ADD COLUMN description TEXT NOT NULL DEFAULT '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE buckets DROP COLUMN description;

-- +goose StatementEnd
