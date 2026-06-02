-- +goose NO TRANSACTION

-- +goose Up
ALTER TYPE provider_type ADD VALUE IF NOT EXISTS 'ldap';

-- +goose Down
ALTER TABLE users ALTER COLUMN provider_type TYPE TEXT;
DROP TYPE provider_type;
CREATE TYPE provider_type AS ENUM ('local', 'oidc');
ALTER TABLE users
    ALTER COLUMN provider_type TYPE provider_type USING provider_type::text::provider_type;
