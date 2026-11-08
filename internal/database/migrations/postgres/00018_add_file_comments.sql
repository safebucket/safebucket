-- +goose Up
-- +goose StatementBegin

CREATE TABLE file_comments
    (
        id uuid
            PRIMARY KEY DEFAULT gen_random_uuid(),
        file_id uuid NOT NULL,
        author_id uuid,
        body TEXT NOT NULL,
        version INTEGER NOT NULL,
        edited_at TIMESTAMP,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

        CONSTRAINT fk_file_comments_file_id
            FOREIGN KEY (file_id) REFERENCES files (id) ON UPDATE CASCADE ON DELETE CASCADE,
        CONSTRAINT fk_file_comments_author_id
            FOREIGN KEY (author_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL
    );

CREATE INDEX idx_file_comments_file_id ON file_comments (file_id, created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS file_comments;

-- +goose StatementEnd
