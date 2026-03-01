-- +goose Up
-- +goose StatementBegin

CREATE TABLE users
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

CREATE INDEX idx_users_email ON users (email);
CREATE UNIQUE INDEX idx_users_email_provider_key ON users (email, provider_key) WHERE deleted_at IS NULL;

CREATE TABLE buckets
    (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        created_by TEXT NOT NULL,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME,

        CONSTRAINT fk_buckets_created_by
            FOREIGN KEY (created_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE
    );

CREATE INDEX idx_buckets_created_by ON buckets (created_by);

CREATE TABLE memberships
    (
        id TEXT PRIMARY KEY,
        user_id TEXT NOT NULL,
        bucket_id TEXT NOT NULL,
        "group" TEXT NOT NULL CHECK("group" IN ('owner', 'contributor', 'viewer')),
        upload_notifications INTEGER NOT NULL DEFAULT 1,
        download_notifications INTEGER NOT NULL DEFAULT 0,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME,

        CONSTRAINT fk_memberships_user_id
            FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_memberships_bucket_id
            FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE
    );

CREATE INDEX idx_memberships_user_id ON memberships (user_id);
CREATE INDEX idx_memberships_bucket_id ON memberships (bucket_id);
CREATE UNIQUE INDEX idx_memberships_user_bucket ON memberships (user_id, bucket_id) WHERE deleted_at IS NULL;

CREATE TABLE folders
    (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        status TEXT CHECK(status IN ('uploading', 'uploaded', 'deleting', 'deleted', 'restoring')),
        folder_id TEXT,
        bucket_id TEXT NOT NULL,
        deleted_by TEXT,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME,

        CONSTRAINT fk_folders_folder_id
            FOREIGN KEY (folder_id) REFERENCES folders (id) ON UPDATE CASCADE ON DELETE SET NULL,
        CONSTRAINT fk_folders_bucket_id
            FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_folders_deleted_by
            FOREIGN KEY (deleted_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL
    );

CREATE INDEX idx_folders_bucket_parent ON folders (bucket_id, folder_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_folders_deleted_by ON folders (deleted_by) WHERE deleted_by IS NOT NULL;
CREATE UNIQUE INDEX idx_folders_unique_name ON folders (bucket_id,
                                                        COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'),
                                                        name) WHERE deleted_at IS NULL;

CREATE TABLE files
    (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        extension TEXT,
        status TEXT CHECK(status IN ('uploading', 'uploaded', 'deleting', 'deleted', 'restoring')),
        bucket_id TEXT NOT NULL,
        folder_id TEXT,
        size INTEGER NOT NULL DEFAULT 0,
        deleted_by TEXT,
        expires_at DATETIME,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME,

        CONSTRAINT fk_files_bucket_id
            FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_files_folder_id
            FOREIGN KEY (folder_id) REFERENCES folders (id) ON UPDATE CASCADE ON DELETE SET NULL,
        CONSTRAINT fk_files_deleted_by
            FOREIGN KEY (deleted_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL,

        CONSTRAINT chk_files_size_positive
            CHECK (size >= 0)
    );

CREATE INDEX idx_files_bucket_folder ON files (bucket_id, folder_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_deleted_by ON files (deleted_by) WHERE deleted_by IS NOT NULL;
CREATE UNIQUE INDEX idx_files_unique_name ON files (bucket_id,
                                                    COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'),
                                                    name) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_expires_at ON files (expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE invites
    (
        id TEXT PRIMARY KEY,
        email TEXT NOT NULL,
        "group" TEXT NOT NULL CHECK("group" IN ('owner', 'contributor', 'viewer')),
        bucket_id TEXT NOT NULL,
        created_by TEXT NOT NULL,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

        CONSTRAINT fk_invites_bucket_id
            FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_invites_created_by
            FOREIGN KEY (created_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,

        CONSTRAINT idx_invite_unique
            UNIQUE (email, "group", bucket_id)
    );

CREATE INDEX idx_invites_bucket_id ON invites (bucket_id);
CREATE INDEX idx_invites_email ON invites (email);

CREATE TABLE challenges
    (
        id TEXT PRIMARY KEY,
        type TEXT NOT NULL CHECK(type IN ('invite', 'password_reset', 'mfa_reset')),
        hashed_secret TEXT NOT NULL,
        attempts_left INTEGER NOT NULL DEFAULT 3,
        expires_at DATETIME,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME,
        invite_id TEXT,
        user_id TEXT,

        CONSTRAINT fk_challenges_invite_id
            FOREIGN KEY (invite_id) REFERENCES invites (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_challenges_user_id
            FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,

        CONSTRAINT chk_challenges_attempts_left
            CHECK (attempts_left >= 0),
        CONSTRAINT chk_challenges_mutual_exclusive
            CHECK ( (invite_id IS NOT NULL AND user_id IS NULL) OR (invite_id IS NULL AND user_id IS NOT NULL) )
    );

CREATE INDEX idx_challenges_expires_at ON challenges (expires_at);
CREATE UNIQUE INDEX idx_challenge_invite ON challenges (invite_id) WHERE invite_id IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX idx_challenge_user ON challenges (user_id) WHERE user_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE mfa_devices (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type TEXT NOT NULL DEFAULT 'totp' CHECK(type IN ('totp')),
    encrypted_secret TEXT NOT NULL,
    is_default INTEGER NOT NULL DEFAULT 0,
    is_verified INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verified_at DATETIME,
    last_used_at DATETIME,

    CONSTRAINT unique_user_device_name UNIQUE (user_id, name)
);

CREATE UNIQUE INDEX idx_mfa_devices_one_default_per_user
    ON mfa_devices (user_id)
    WHERE is_default = 1;

CREATE INDEX idx_mfa_devices_user_id ON mfa_devices (user_id);

CREATE INDEX idx_mfa_devices_verified
    ON mfa_devices (user_id)
    WHERE is_verified = 1;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS mfa_devices;
DROP TABLE IF EXISTS challenges;
DROP TABLE IF EXISTS invites;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS folders;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS buckets;
DROP TABLE IF EXISTS users;

-- +goose StatementEnd
