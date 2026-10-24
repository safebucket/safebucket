-- +goose Up
-- +goose StatementBegin

CREATE TABLE outbox
    (
        id TEXT PRIMARY KEY,
        type TEXT NOT NULL,
        name TEXT,
        payload TEXT NOT NULL,
        attempts INTEGER NOT NULL DEFAULT 0,
        last_error TEXT,
        next_attempt_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        failed_at DATETIME,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE INDEX idx_outbox_pending
    ON outbox (type, next_attempt_at, created_at)
    WHERE failed_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_outbox_pending;
DROP TABLE IF EXISTS outbox;

-- +goose StatementEnd
