-- +goose Up
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_buckets_updated_at ON buckets;
DROP TRIGGER IF EXISTS update_memberships_updated_at ON memberships;
DROP TRIGGER IF EXISTS update_folders_updated_at ON folders;
DROP TRIGGER IF EXISTS update_files_updated_at ON files;
DROP FUNCTION IF EXISTS update_updated_at_column();
-- +goose Down
-- no-op: triggers are not recreated
