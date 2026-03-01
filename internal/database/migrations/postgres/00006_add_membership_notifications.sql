-- +goose Up
ALTER TABLE memberships ADD COLUMN upload_notifications BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE memberships ADD COLUMN download_notifications BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE memberships DROP COLUMN IF EXISTS upload_notifications;
ALTER TABLE memberships DROP COLUMN IF EXISTS download_notifications;
