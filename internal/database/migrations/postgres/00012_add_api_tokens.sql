-- +goose NO TRANSACTION

-- +goose Up
ALTER TYPE provider_type ADD VALUE IF NOT EXISTS 'service_account';

-- +goose StatementBegin
CREATE TABLE api_tokens
    (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        token_hash TEXT NOT NULL,
        user_id UUID NOT NULL,
        name TEXT NOT NULL,
        expires_at TIMESTAMP NOT NULL,
        last_used_at TIMESTAMP,
        created_by UUID,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        revoked_at TIMESTAMP,
        deleted_at TIMESTAMP,

        CONSTRAINT fk_api_tokens_user_id
            FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_api_tokens_created_by
            FOREIGN KEY (created_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL
    );
-- +goose StatementEnd

CREATE UNIQUE INDEX idx_api_tokens_token_hash ON api_tokens (token_hash) WHERE deleted_at IS NULL;
CREATE INDEX idx_api_tokens_user_id ON api_tokens (user_id) WHERE deleted_at IS NULL;

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS api_tokens;
-- +goose StatementEnd
