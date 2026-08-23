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

-- +goose Down
