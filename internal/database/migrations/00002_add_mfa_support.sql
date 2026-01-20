-- +goose Up
-- +goose StatementBegin

ALTER TYPE challenge_type ADD VALUE 'mfa_reset';

CREATE TYPE mfa_device_type AS ENUM ('totp');

CREATE TABLE mfa_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Device metadata
    name VARCHAR(100) NOT NULL,
    type mfa_device_type NOT NULL DEFAULT 'totp',

    -- TOTP-specific fields
    encrypted_secret TEXT NOT NULL,

    -- Device management
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verified_at TIMESTAMP,
    last_used_at TIMESTAMP,

    -- Constraints
    CONSTRAINT unique_user_device_name UNIQUE (user_id, name)
);

CREATE UNIQUE INDEX idx_mfa_devices_one_default_per_user
    ON mfa_devices (user_id)
    WHERE is_default = TRUE;

CREATE INDEX idx_mfa_devices_user_id ON mfa_devices (user_id);

CREATE INDEX idx_mfa_devices_verified
    ON mfa_devices (user_id)
    WHERE is_verified = TRUE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_mfa_devices_verified;
DROP INDEX IF EXISTS idx_mfa_devices_user_id;
DROP INDEX IF EXISTS idx_mfa_devices_one_default_per_user;
DROP TABLE IF EXISTS mfa_devices;
DROP TYPE IF EXISTS mfa_device_type;

-- +goose StatementEnd
