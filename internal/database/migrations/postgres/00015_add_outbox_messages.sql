-- +goose Up
-- +goose StatementBegin

CREATE TABLE outbox_messages
    (
        id UUID PRIMARY KEY,
        kind TEXT NOT NULL CHECK(kind IN ('event', 'activity')),
        event_type TEXT,
        payload TEXT NOT NULL,
        attempts INTEGER NOT NULL DEFAULT 0,
        last_error TEXT,
        next_attempt_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        failed_at TIMESTAMP,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE INDEX idx_outbox_messages_pending
    ON outbox_messages (kind, next_attempt_at, created_at)
    WHERE failed_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_outbox_messages_pending;
DROP TABLE IF EXISTS outbox_messages;

-- +goose StatementEnd
