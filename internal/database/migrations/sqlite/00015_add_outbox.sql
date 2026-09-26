-- +goose Up
CREATE TABLE queue_outbox_messages
(
    id TEXT PRIMARY KEY,
    message_type TEXT NOT NULL,
    payload TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_error TEXT,
    claim_id TEXT,
    claimed_until DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_queue_outbox_messages_pending
    ON queue_outbox_messages (available_at, created_at);

CREATE TABLE activity_outbox_messages
(
    id TEXT PRIMARY KEY,
    payload TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_error TEXT,
    claim_id TEXT,
    claimed_until DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_activity_outbox_messages_pending
    ON activity_outbox_messages (available_at, created_at);

-- +goose Down
DROP TABLE IF EXISTS activity_outbox_messages;
DROP TABLE IF EXISTS queue_outbox_messages;
