-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;
CREATE TABLE users_new
    (
        id TEXT PRIMARY KEY,
        first_name TEXT,
        last_name TEXT,
        email TEXT NOT NULL,
        hashed_password TEXT,
        is_initialized INTEGER NOT NULL DEFAULT 0,
        provider_type TEXT NOT NULL CHECK(provider_type IN ('local', 'oidc', 'ldap')),
        provider_key TEXT NOT NULL,
        role TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('admin', 'user', 'guest')),
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME
    );
INSERT INTO users_new SELECT id, first_name, last_name, email, hashed_password, is_initialized,
    provider_type, provider_key, role, created_at, updated_at, deleted_at FROM users;
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;
CREATE INDEX idx_users_email ON users (email);
CREATE UNIQUE INDEX idx_users_email_provider_key ON users (email, provider_key) WHERE deleted_at IS NULL;
PRAGMA foreign_keys=ON;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;
CREATE TABLE users_old
    (
        id TEXT PRIMARY KEY,
        first_name TEXT,
        last_name TEXT,
        email TEXT NOT NULL,
        hashed_password TEXT,
        is_initialized INTEGER NOT NULL DEFAULT 0,
        provider_type TEXT NOT NULL CHECK(provider_type IN ('local', 'oidc')),
        provider_key TEXT NOT NULL,
        role TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('admin', 'user', 'guest')),
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME
    );
INSERT INTO users_old SELECT id, first_name, last_name, email, hashed_password, is_initialized,
    provider_type, provider_key, role, created_at, updated_at, deleted_at FROM users
    WHERE provider_type IN ('local', 'oidc');
DROP TABLE users;
ALTER TABLE users_old RENAME TO users;
CREATE INDEX idx_users_email ON users (email);
CREATE UNIQUE INDEX idx_users_email_provider_key ON users (email, provider_key) WHERE deleted_at IS NULL;
PRAGMA foreign_keys=ON;
-- +goose StatementEnd
