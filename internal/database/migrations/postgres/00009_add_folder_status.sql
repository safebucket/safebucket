-- +goose Up
CREATE TYPE folder_status AS ENUM ('ready', 'deleted', 'restoring');
ALTER TABLE folders
    ALTER COLUMN status SET DEFAULT 'ready'::folder_status,
    ALTER COLUMN status TYPE folder_status USING (CASE WHEN status::text IN ('', 'uploading', 'uploaded') OR status IS NULL THEN 'ready' ELSE status::text END)::folder_status,
    ALTER COLUMN status SET NOT NULL;

-- +goose Down
ALTER TABLE folders
    ALTER COLUMN status DROP NOT NULL,
    ALTER COLUMN status TYPE file_status USING CASE WHEN status::text = 'ready' THEN NULL ELSE status::text::file_status END,
    ALTER COLUMN status SET DEFAULT NULL;
DROP TYPE folder_status;
