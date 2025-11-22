-- +goose Up
-- +goose StatementBegin

SELECT 'up SQL query';

-- Create custom ENUM types
CREATE TYPE file_status AS ENUM ('uploading', 'uploaded', 'deleting', 'trashed', 'restoring');
CREATE TYPE provider_type AS ENUM ('local', 'oidc');
CREATE TYPE challenge_type AS ENUM ('invite', 'password_reset');
CREATE TYPE group_type AS ENUM ('owner', 'contributor', 'viewer');
CREATE TYPE role_type AS ENUM ('admin', 'user', 'guest');

-- Users table
CREATE TABLE users
(
    id              uuid
        PRIMARY KEY                        DEFAULT gen_random_uuid(),
    first_name      TEXT,
    last_name       TEXT email TEXT NOT NULL,
    hashed_password TEXT,
    is_initialized  BOOLEAN       NOT NULL DEFAULT FALSE,
    provider_type   provider_type NOT NULL,
    provider_key    TEXT          NOT NULL,
    role            role_type     NOT NULL DEFAULT 'user',
    created_at      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP
);

-- Indexes for Users
CREATE INDEX idx_users_email ON users (email);
-- Partial unique index for active users only (soft-delete aware)
CREATE UNIQUE INDEX idx_users_email_provider_key ON users (email, provider_key) WHERE deleted_at IS NULL;

-- Buckets table
CREATE TABLE buckets
(
    id         uuid
        PRIMARY KEY               DEFAULT gen_random_uuid(),
    name       TEXT      NOT NULL,
    created_by uuid      NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- Foreign Keys
    CONSTRAINT fk_buckets_created_by
        FOREIGN KEY (created_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE
);

-- Indexes for Buckets
CREATE INDEX idx_buckets_created_by ON buckets (created_by);

-- Memberships table
CREATE TABLE memberships
(
    id         uuid
        PRIMARY KEY                DEFAULT gen_random_uuid(),
    user_id    uuid       NOT NULL,
    bucket_id  uuid       NOT NULL,
    "group"    group_type NOT NULL,
    created_at TIMESTAMP  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- Foreign Keys
    CONSTRAINT fk_memberships_user_id
        FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_memberships_bucket_id
        FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE
);

-- Indexes for Memberships
CREATE INDEX idx_memberships_user_id ON memberships (user_id);
CREATE INDEX idx_memberships_bucket_id ON memberships (bucket_id);
-- Partial unique index for active memberships only (soft-delete aware)
CREATE UNIQUE INDEX idx_memberships_user_bucket ON memberships (user_id, bucket_id) WHERE deleted_at IS NULL;

-- Folders table
CREATE TABLE folders
(
    id         uuid
        PRIMARY KEY               DEFAULT gen_random_uuid(),
    name       TEXT      NOT NULL,
    status     file_status,
    folder_id  uuid,
    bucket_id  uuid      NOT NULL,
    trashed_at TIMESTAMP,
    trashed_by uuid,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

-- Foreign Keys
    CONSTRAINT fk_folders_folder_id
        FOREIGN KEY (folder_id) REFERENCES folders (id) ON UPDATE CASCADE ON DELETE SET NULL,
    CONSTRAINT fk_folders_bucket_id
        FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_folders_trashed_by
        FOREIGN KEY (trashed_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL
);

-- Indexes for Folders
-- Composite index for folder browsing (list folders in bucket at specific parent)
CREATE INDEX idx_folders_bucket_parent ON folders (bucket_id, folder_id) WHERE deleted_at IS NULL;
-- Composite index for trash view (list trashed folders in bucket)
CREATE INDEX idx_folders_bucket_trashed ON folders (bucket_id, trashed_at) WHERE trashed_at IS NOT NULL;
-- Unique index for folder name uniqueness within same parent (using COALESCE to handle NULL folder_id)
CREATE UNIQUE INDEX idx_folders_unique_name
    ON folders (bucket_id, COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'::uuid), name) WHERE deleted_at IS NULL;

-- Files table
CREATE TABLE files
(
    id         uuid
        PRIMARY KEY               DEFAULT gen_random_uuid(),
    name       TEXT      NOT NULL,
    extension  TEXT,
    status     file_status,
    bucket_id  uuid      NOT NULL,
    folder_id  uuid,
    size       BIGINT,
    trashed_at TIMESTAMP,
    trashed_by uuid,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- Foreign Keys
    CONSTRAINT fk_files_bucket_id
        FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_files_folder_id
        FOREIGN KEY (folder_id) REFERENCES folders (id) ON UPDATE CASCADE ON DELETE SET NULL,
    CONSTRAINT fk_files_trashed_by
        FOREIGN KEY (trashed_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL,

    -- Constraints
    CONSTRAINT chk_files_size_positive
        CHECK (size IS NULL OR size >= 0
)
    );

-- Indexes for Files
-- Composite index for file browsing (list files in bucket at specific folder)
CREATE INDEX idx_files_bucket_folder ON files (bucket_id, folder_id) WHERE deleted_at IS NULL;
-- Composite index for trash view (list trashed files in bucket)
CREATE INDEX idx_files_bucket_trashed ON files (bucket_id, trashed_at) WHERE trashed_at IS NOT NULL;
-- Unique index for file name uniqueness within same folder (using COALESCE to handle NULL folder_id)
CREATE UNIQUE INDEX idx_files_unique_name
    ON files (bucket_id, COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'::uuid), name) WHERE deleted_at IS NULL;

-- Invites table
CREATE TABLE invites
(
    id         uuid
        PRIMARY KEY                DEFAULT gen_random_uuid(),
    email      TEXT       NOT NULL,
    "group"    group_type NOT NULL,
    bucket_id  uuid       NOT NULL,
    created_by uuid       NOT NULL,
    created_at TIMESTAMP  NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Foreign Keys
    CONSTRAINT fk_invites_bucket_id
        FOREIGN KEY (bucket_id) REFERENCES buckets (id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_invites_created_by
        FOREIGN KEY (created_by) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,

    -- Constraints
    CONSTRAINT idx_invite_unique
        UNIQUE (email, "group", bucket_id)
);

-- Indexes for Invites
CREATE INDEX idx_invites_bucket_id ON invites (bucket_id);
CREATE INDEX idx_invites_email ON invites (email);

-- Challenges table
CREATE TABLE challenges
(
    id            uuid
        PRIMARY KEY                       DEFAULT gen_random_uuid(),
    type          challenge_type NOT NULL,
    hashed_secret TEXT           NOT NULL,
    attempts_left INTEGER        NOT NULL DEFAULT 3,
    expires_at    TIMESTAMP,
    created_at    TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    TIMESTAMP,
    invite_id     uuid,
    user_id       uuid,

    -- Foreign Keys
    CONSTRAINT fk_challenges_invite_id
        FOREIGN KEY (invite_id) REFERENCES invites (id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_challenges_user_id
        FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,

    -- Constraints
    CONSTRAINT chk_challenges_attempts_left
        CHECK (attempts_left >= 0),
    CONSTRAINT chk_challenges_mutual_exclusive
        CHECK ( (invite_id IS NOT NULL AND user_id IS NULL) OR (invite_id IS NULL AND user_id IS NOT NULL) )
);

-- Indexes for Challenges
CREATE INDEX idx_challenges_expires_at ON challenges (expires_at);
CREATE UNIQUE INDEX idx_challenge_invite ON challenges (invite_id) WHERE invite_id IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX idx_challenge_user ON challenges (user_id) WHERE user_id IS NOT NULL AND deleted_at IS NULL;

-- Trigger to update updated_at timestamp automatically
CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS TRIGGER AS
$$
BEGIN
new.updated_at
= CURRENT_TIMESTAMP;
RETURN new;
END;
$$ LANGUAGE 'plpgsql';

-- Apply triggers to tables with updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE
    ON users
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_buckets_updated_at
    BEFORE UPDATE
    ON buckets
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_memberships_updated_at
    BEFORE UPDATE
    ON memberships
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_folders_updated_at
    BEFORE UPDATE
    ON folders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_files_updated_at
    BEFORE UPDATE
    ON files
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop triggers
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_buckets_updated_at ON buckets;
DROP TRIGGER IF EXISTS update_memberships_updated_at ON memberships;
DROP TRIGGER IF EXISTS update_folders_updated_at ON folders;
DROP TRIGGER IF EXISTS update_files_updated_at ON files;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign key dependencies)
DROP TABLE IF EXISTS challenges;
DROP TABLE IF EXISTS invites;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS folders;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS buckets;
DROP TABLE IF EXISTS users;

-- Drop custom types
DROP TYPE IF EXISTS role_type;
DROP TYPE IF EXISTS group_type;
DROP TYPE IF EXISTS challenge_type;
DROP TYPE IF EXISTS provider_type;
DROP TYPE IF EXISTS file_status;

-- +goose StatementEnd
