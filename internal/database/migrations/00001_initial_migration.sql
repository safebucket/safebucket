-- +goose Up
-- +goose StatementBegin
select 'up SQL query';

create table if not exists users
(
    id              UUID primary key                  default gen_random_uuid(),
    first_name      TEXT                              default null,
    last_name       TEXT                              default null,
    email           TEXT unique              not null default null,
    hashed_password TEXT                              default null,
    is_external     BOOLEAN                  not null default false,
    created_at      timestamp with time zone not null default now(),
    updated_at      timestamp with time zone not null default now(),
    deleted_at      timestamp with time zone,
    check (hashed_password is not null or is_external = true)
);

create index if not exists idx_users_deleted_at on users (deleted_at);

create table if not exists buckets
(
    id         UUID primary key                  default gen_random_uuid(),
    name       TEXT                     not null default null,
    created_by UUID                     not null,
    created_at timestamp with time zone not null default now(),
    updated_at timestamp with time zone not null default now(),
    deleted_at timestamp with time zone
);

create index if not exists idx_buckets_deleted_at on buckets (deleted_at);

create table if not exists files
(
    id         UUID primary key                  default gen_random_uuid(),
    name       TEXT                     not null default null,
    extension  TEXT                              default null,
    uploaded   BOOLEAN                  not null default false,
    bucket_id  UUID                     references buckets (id) on delete set null,
    path       TEXT                     not null default '/',
    type       TEXT                     not null default null,
    size       integer                           default null,
    created_at timestamp with time zone not null default now(),
    updated_at timestamp with time zone not null default now(),
    deleted_at timestamp with time zone
);

create index if not exists idx_files_deleted_at on files (deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
select 'down SQL query';
-- +goose StatementEnd
