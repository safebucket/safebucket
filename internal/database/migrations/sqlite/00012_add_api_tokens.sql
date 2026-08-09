-- +goose Up
-- +goose StatementBegin

CREATE TABLE api_tokens
    (
        id TEXT PRIMARY KEY,
        token_hash TEXT NOT NULL,
        user_id TEXT NOT NULL,
        name TEXT NOT NULL,
        expires_at DATETIME NOT NULL,
        last_used_at DATETIME,
        created_by TEXT,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        revoked_at DATETIME,
        deleted_at DATETIME,

        CONSTRAINT fk_api_tokens_user_id
            FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_api_tokens_created_by
            FOREIGN KEY (created_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL
    );

CREATE UNIQUE INDEX idx_api_tokens_token_hash ON api_tokens (token_hash) WHERE deleted_at IS NULL;
CREATE INDEX idx_api_tokens_user_id ON api_tokens (user_id) WHERE deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS api_tokens;

-- +goose StatementEnd
