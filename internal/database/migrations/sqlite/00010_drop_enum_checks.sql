-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;

CREATE TABLE users_new
    (
        id TEXT PRIMARY KEY,
        first_name TEXT,
        last_name TEXT,
        email TEXT NOT NULL,
        hashed_password TEXT,
        is_initialized INTEGER NOT NULL DEFAULT 0,
        provider_type TEXT NOT NULL,
        provider_key TEXT NOT NULL,
        role TEXT NOT NULL DEFAULT 'user',
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

CREATE TABLE memberships_new
    (
        id TEXT PRIMARY KEY,
        user_id TEXT NOT NULL,
        bucket_id TEXT NOT NULL,
        "group" TEXT NOT NULL,
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
INSERT INTO memberships_new SELECT id, user_id, bucket_id, "group", upload_notifications,
    download_notifications, created_at, updated_at, deleted_at FROM memberships;
DROP TABLE memberships;
ALTER TABLE memberships_new RENAME TO memberships;
CREATE INDEX idx_memberships_user_id ON memberships (user_id);
CREATE INDEX idx_memberships_bucket_id ON memberships (bucket_id);
CREATE UNIQUE INDEX idx_memberships_user_bucket ON memberships (user_id, bucket_id) WHERE deleted_at IS NULL;

CREATE TABLE folders_new
    (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        status TEXT NOT NULL DEFAULT 'created',
        folder_id TEXT,
        bucket_id TEXT NOT NULL,
        deleted_by TEXT,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME,
        CONSTRAINT fk_folders_folder_id
            FOREIGN KEY (folder_id) REFERENCES folders_new (id) ON UPDATE CASCADE ON DELETE SET NULL,
        CONSTRAINT fk_folders_bucket_id
            FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_folders_deleted_by
            FOREIGN KEY (deleted_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL
    );
INSERT INTO folders_new SELECT id, name, status, folder_id, bucket_id, deleted_by,
    created_at, updated_at, deleted_at FROM folders;
DROP TABLE folders;
ALTER TABLE folders_new RENAME TO folders;
CREATE INDEX idx_folders_bucket_parent ON folders (bucket_id, folder_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_folders_deleted_by ON folders (deleted_by) WHERE deleted_by IS NOT NULL;
CREATE UNIQUE INDEX idx_folders_unique_name ON folders (bucket_id, COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'), name) WHERE deleted_at IS NULL;

CREATE TABLE files_new
    (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        extension TEXT,
        status TEXT,
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
INSERT INTO files_new SELECT id, name, extension, status, bucket_id, folder_id, size,
    deleted_by, expires_at, created_at, updated_at, deleted_at FROM files;
DROP TABLE files;
ALTER TABLE files_new RENAME TO files;
CREATE INDEX idx_files_bucket_folder ON files (bucket_id, folder_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_deleted_by ON files (deleted_by) WHERE deleted_by IS NOT NULL;
CREATE UNIQUE INDEX idx_files_unique_name ON files (bucket_id, COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'), name) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_expires_at ON files (expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE invites_new
    (
        id TEXT PRIMARY KEY,
        email TEXT NOT NULL,
        "group" TEXT NOT NULL,
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
INSERT INTO invites_new SELECT id, email, "group", bucket_id, created_by, created_at FROM invites;
DROP TABLE invites;
ALTER TABLE invites_new RENAME TO invites;
CREATE INDEX idx_invites_bucket_id ON invites (bucket_id);
CREATE INDEX idx_invites_email ON invites (email);

CREATE TABLE challenges_new
    (
        id TEXT PRIMARY KEY,
        type TEXT NOT NULL,
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
INSERT INTO challenges_new SELECT id, type, hashed_secret, attempts_left, expires_at,
    created_at, deleted_at, invite_id, user_id FROM challenges;
DROP TABLE challenges;
ALTER TABLE challenges_new RENAME TO challenges;
CREATE INDEX idx_challenges_expires_at ON challenges (expires_at);
CREATE UNIQUE INDEX idx_challenge_invite ON challenges (invite_id) WHERE invite_id IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX idx_challenge_user ON challenges (user_id) WHERE user_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE mfa_devices_new
    (
        id TEXT PRIMARY KEY,
        user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
        name VARCHAR(100) NOT NULL,
        type TEXT NOT NULL DEFAULT 'totp',
        encrypted_secret TEXT NOT NULL,
        is_default INTEGER NOT NULL DEFAULT 0,
        is_verified INTEGER NOT NULL DEFAULT 0,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        verified_at DATETIME,
        last_used_at DATETIME,
        CONSTRAINT unique_user_device_name UNIQUE (user_id, name)
    );
INSERT INTO mfa_devices_new SELECT id, user_id, name, type, encrypted_secret, is_default,
    is_verified, created_at, updated_at, verified_at, last_used_at FROM mfa_devices;
DROP TABLE mfa_devices;
ALTER TABLE mfa_devices_new RENAME TO mfa_devices;
CREATE UNIQUE INDEX idx_mfa_devices_one_default_per_user ON mfa_devices (user_id) WHERE is_default = 1;
CREATE INDEX idx_mfa_devices_user_id ON mfa_devices (user_id);
CREATE INDEX idx_mfa_devices_verified ON mfa_devices (user_id) WHERE is_verified = 1;

CREATE TABLE shares_new
    (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        bucket_id TEXT NOT NULL,
        folder_id TEXT,
        expires_at DATETIME,
        max_views INTEGER,
        current_views INTEGER NOT NULL DEFAULT 0,
        hashed_password TEXT,
        type TEXT NOT NULL,
        allow_upload INTEGER NOT NULL DEFAULT 0,
        max_uploads INTEGER,
        current_uploads INTEGER NOT NULL DEFAULT 0,
        max_upload_size INTEGER,
        created_by TEXT NOT NULL,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME,
        CONSTRAINT fk_shares_bucket_id
            FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_shares_folder_id
            FOREIGN KEY (folder_id) REFERENCES folders (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_shares_created_by
            FOREIGN KEY (created_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT chk_shares_upload_type
            CHECK (NOT (allow_upload = 1 AND type = 'files')),
        CONSTRAINT chk_shares_current_views
            CHECK (current_views >= 0),
        CONSTRAINT chk_shares_current_uploads
            CHECK (current_uploads >= 0)
    );
INSERT INTO shares_new SELECT id, name, bucket_id, folder_id, expires_at, max_views,
    current_views, hashed_password, type, allow_upload, max_uploads, current_uploads,
    max_upload_size, created_by, created_at, updated_at, deleted_at FROM shares;
DROP TABLE shares;
ALTER TABLE shares_new RENAME TO shares;
CREATE INDEX idx_shares_bucket_id ON shares (bucket_id) WHERE deleted_at IS NULL;

COMMIT;
PRAGMA foreign_keys=ON;
-- +goose StatementEnd
