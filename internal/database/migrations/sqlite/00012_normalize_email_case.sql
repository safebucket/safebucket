-- +goose Up
UPDATE users SET email = LOWER(email)
WHERE email != LOWER(email)
  AND (provider_key, LOWER(email)) NOT IN (
    SELECT provider_key, LOWER(email) FROM users
    WHERE deleted_at IS NULL
    GROUP BY provider_key, LOWER(email)
    HAVING COUNT(*) > 1
  );

UPDATE invites SET email = LOWER(email)
WHERE email != LOWER(email)
  AND ("group", bucket_id, LOWER(email)) NOT IN (
    SELECT "group", bucket_id, LOWER(email) FROM invites
    GROUP BY "group", bucket_id, LOWER(email)
    HAVING COUNT(*) > 1
  );

INSERT OR IGNORE INTO memberships (id, user_id, bucket_id, "group", created_at, updated_at)
SELECT lower(hex(randomblob(16))),
       (SELECT u.id FROM users u
        WHERE u.email = LOWER(i.email) AND u.deleted_at IS NULL
        ORDER BY u.created_at ASC, u.id ASC
        LIMIT 1),
       i.bucket_id, i."group", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM invites i
WHERE EXISTS (
    SELECT 1 FROM users u
    WHERE u.email = LOWER(i.email) AND u.deleted_at IS NULL
);

DELETE FROM invites
WHERE EXISTS (
    SELECT 1 FROM users u
    WHERE u.email = LOWER(invites.email) AND u.deleted_at IS NULL
);

-- +goose Down
