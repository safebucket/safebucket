-- +goose Up
-- +goose StatementBegin

-- Add MFA fields to users table
ALTER TABLE users
    ADD COLUMN mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN mfa_secret_encrypted TEXT,
    ADD COLUMN mfa_enabled_at TIMESTAMP;

-- Create index for MFA queries
CREATE INDEX idx_users_mfa_enabled ON users (mfa_enabled) WHERE mfa_enabled = TRUE;

-- Add mfa_reset to challenge_type enum
ALTER TYPE challenge_type ADD VALUE 'mfa_reset';

-- Create MFA device type enum (extensible for future passkeys/email)
CREATE TYPE mfa_device_type AS ENUM ('totp');

-- Create user_mfa_devices table
CREATE TABLE user_mfa_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Device metadata
    name VARCHAR(100) NOT NULL,
    type mfa_device_type NOT NULL DEFAULT 'totp',

    -- TOTP-specific fields
    secret_encrypted TEXT NOT NULL,

    -- Device management
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verified_at TIMESTAMP,
    last_used_at TIMESTAMP,

    -- Constraints
    CONSTRAINT unique_user_device_name UNIQUE (user_id, name)
);

-- Ensure only one primary device per user (among verified devices)
CREATE UNIQUE INDEX idx_user_primary_device
    ON user_mfa_devices (user_id)
    WHERE is_primary = TRUE AND is_verified = TRUE;

-- Index for user device lookups
CREATE INDEX idx_user_mfa_devices_user_id ON user_mfa_devices (user_id);

-- Index for finding verified devices
CREATE INDEX idx_user_mfa_devices_verified
    ON user_mfa_devices (user_id)
    WHERE is_verified = TRUE;

-- Migrate existing MFA data from users table to user_mfa_devices
-- This creates a device named "Authenticator" for each user with MFA enabled
INSERT INTO user_mfa_devices (
    user_id,
    name,
    type,
    secret_encrypted,
    is_primary,
    is_verified,
    created_at,
    updated_at,
    verified_at
)
SELECT
    id AS user_id,
    'Authenticator' AS name,
    'totp' AS type,
    mfa_secret_encrypted AS secret_encrypted,
    TRUE AS is_primary,
    TRUE AS is_verified,
    COALESCE(mfa_enabled_at, created_at) AS created_at,
    updated_at,
    mfa_enabled_at AS verified_at
FROM users
WHERE mfa_enabled = TRUE
  AND mfa_secret_encrypted IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove migrated devices
DELETE FROM user_mfa_devices
WHERE name = 'Authenticator'
  AND is_primary = TRUE
  AND is_verified = TRUE;

-- Drop user_mfa_devices table and indexes
DROP INDEX IF EXISTS idx_user_mfa_devices_verified;
DROP INDEX IF EXISTS idx_user_mfa_devices_user_id;
DROP INDEX IF EXISTS idx_user_primary_device;
DROP TABLE IF EXISTS user_mfa_devices;
DROP TYPE IF EXISTS mfa_device_type;

-- Drop MFA columns from users
DROP INDEX IF EXISTS idx_users_mfa_enabled;
ALTER TABLE users DROP COLUMN IF EXISTS mfa_enabled_at;
ALTER TABLE users DROP COLUMN IF EXISTS mfa_secret_encrypted;
ALTER TABLE users DROP COLUMN IF EXISTS mfa_enabled;

-- Note: Cannot remove enum value in PostgreSQL, mfa_reset will remain in challenge_type

-- +goose StatementEnd