-- +goose Up
-- +goose StatementBegin

select 'up SQL query';

-- Create custom ENUM types
create type file_status as ENUM ('uploading', 'uploaded', 'deleting');
create type file_type as ENUM ('file', 'folder');
create type provider_type as ENUM ('local', 'oidc');
create type challenge_type as ENUM ('invite', 'password_reset');
create type group_type as ENUM ('owner', 'contributor', 'viewer');

-- Users table
create table users
(
    id              UUID primary key       default gen_random_uuid(),
    first_name      VARCHAR(255),
    last_name       VARCHAR(255),
    email           VARCHAR(255)  not null,
    hashed_password VARCHAR(255),
    is_initialized  BOOLEAN       not null default false,
    provider_type   provider_type not null,
    provider_key    VARCHAR(255)  not null,
    created_at      TIMESTAMP     not null default current_timestamp,
    updated_at      TIMESTAMP     not null default current_timestamp,
    deleted_at      TIMESTAMP,

    -- Constraints
    constraint idx_email_provider_key unique (email, provider_key)
);

-- Indexes for Users
create index idx_users_deleted_at on users (deleted_at);
create index idx_users_email on users (email);
create index idx_users_provider_type on users (provider_type);

-- Buckets table
create table buckets
(
    id         UUID primary key      default gen_random_uuid(),
    name       VARCHAR(255) not null,
    created_by UUID         not null,
    created_at TIMESTAMP    not null default current_timestamp,
    updated_at TIMESTAMP    not null default current_timestamp,
    deleted_at TIMESTAMP,

    -- Foreign Keys
    constraint fk_buckets_created_by foreign key (created_by) references users (id) on update cascade on delete cascade
);

-- Indexes for Buckets
create index idx_buckets_deleted_at on buckets (deleted_at);
create index idx_buckets_created_by on buckets (created_by);
create index idx_buckets_name on buckets (name);

-- Files table
create table files
(
    id         UUID primary key       default gen_random_uuid(),
    name       VARCHAR(255)  not null,
    extension  VARCHAR(50),
    status     file_status,
    bucket_id  UUID          not null,
    path       VARCHAR(1024) not null default '/',
    type       file_type     not null,
    size       BIGINT,
    created_at TIMESTAMP     not null default current_timestamp,
    updated_at TIMESTAMP     not null default current_timestamp,
    deleted_at TIMESTAMP,

    -- Foreign Keys
    constraint fk_files_bucket_id foreign key (bucket_id) references buckets (id) on update cascade on delete cascade,

    -- Constraints
    constraint chk_files_size_positive check (size is null or size >= 0)
);

-- Indexes for Files
create index idx_files_deleted_at on files (deleted_at);
create index idx_files_bucket_id on files (bucket_id);
create index idx_files_path on files (path);
create index idx_files_status on files (status);
create index idx_files_type on files (type);
-- Composite index for file uniqueness within bucket path
create unique index idx_files_unique_path on files (bucket_id, path, name) where deleted_at is null;

-- Invites table
create table invites
(
    id         UUID primary key      default gen_random_uuid(),
    email      VARCHAR(255) not null,
    "group"    group_type not null,
    bucket_id  UUID         not null,
    created_by UUID         not null,
    created_at TIMESTAMP    not null default current_timestamp,

    -- Foreign Keys
    constraint fk_invites_bucket_id foreign key (bucket_id) references buckets (id) on update cascade on delete cascade,
    constraint fk_invites_created_by foreign key (created_by) references users (id) on update cascade on delete cascade,

    -- Constraints
    constraint idx_invite_unique unique (email, "group", bucket_id)
);

-- Indexes for Invites
create index idx_invites_bucket_id on invites (bucket_id);
create index idx_invites_created_by on invites (created_by);
create index idx_invites_email on invites (email);

-- Challenges table
create table challenges
(
    id            UUID primary key        default gen_random_uuid(),
    type          challenge_type not null,
    hashed_secret VARCHAR(255)   not null,
    attempts_left INTEGER        not null default 3,
    expires_at    TIMESTAMP,
    created_at    TIMESTAMP      not null default current_timestamp,
    deleted_at    TIMESTAMP,
    invite_id     UUID,
    user_id       UUID,

    -- Foreign Keys
    constraint fk_challenges_invite_id foreign key (invite_id) references invites (id) on update cascade on delete cascade,
    constraint fk_challenges_user_id foreign key (user_id) references users (id) on update cascade on delete cascade,

    -- Constraints
    constraint chk_challenges_attempts_left check (attempts_left >= 0),
    constraint chk_challenges_mutual_exclusive check ( (invite_id is not null and user_id is null) or
                                                       (invite_id is null and user_id is not null) )
);

-- Indexes for Challenges
create index idx_challenges_deleted_at on challenges (deleted_at);
create index idx_challenges_type on challenges (type);
create index idx_challenges_expires_at on challenges (expires_at);
create unique index idx_challenge_invite on challenges (invite_id) where invite_id is not null and deleted_at is null;
create unique index idx_challenge_user on challenges (user_id) where user_id is not null and deleted_at is null;

-- Policies table (for Casbin RBAC)
create table policies
(
    id         SERIAL primary key,
    ptype      VARCHAR(512),
    v0         VARCHAR(512),
    v1         VARCHAR(512),
    v2         VARCHAR(512),
    v3         VARCHAR(512),
    v4         VARCHAR(512),
    v5         VARCHAR(512),
    created_at TIMESTAMP not null default current_timestamp,
    updated_at TIMESTAMP not null default current_timestamp
);

-- Composite unique index for Casbin policies
create unique index idx_policies_unique on policies (ptype, v0, v1, v2, v3, v4, v5);
-- Individual indexes for common query patterns
create index idx_policies_ptype on policies (ptype);
create index idx_policies_v0 on policies (v0);
create index idx_policies_v1 on policies (v1);

-- Trigger to update updated_at timestamp automatically
create or replace function update_updated_at_column() returns TRIGGER as
$$
begin
    NEW.updated_at = current_timestamp;
    return NEW;
end;
$$ language 'plpgsql';

-- Apply triggers to tables with updated_at
create trigger update_users_updated_at
    before update
    on users
    for each row
execute function update_updated_at_column();

create trigger update_buckets_updated_at
    before update
    on buckets
    for each row
execute function update_updated_at_column();

create trigger update_files_updated_at
    before update
    on files
    for each row
execute function update_updated_at_column();

create trigger update_policies_updated_at
    before update
    on policies
    for each row
execute function update_updated_at_column();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop triggers
drop trigger if exists update_users_updated_at on users;
drop trigger if exists update_buckets_updated_at on buckets;
drop trigger if exists update_files_updated_at on files;
drop trigger if exists update_policies_updated_at on policies;

-- Drop trigger function
drop function if exists update_updated_at_column();

-- Drop tables in reverse order (respecting foreign key dependencies)
drop table if exists policies;
drop table if exists challenges;
drop table if exists invites;
drop table if exists files;
drop table if exists buckets;
drop table if exists users;

-- Drop custom types
drop type if exists group_type;
drop type if exists challenge_type;
drop type if exists provider_type;
drop type if exists file_type;
drop type if exists file_status;

-- +goose StatementEnd
