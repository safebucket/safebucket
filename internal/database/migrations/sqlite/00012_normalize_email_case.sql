-- +goose Up
UPDATE users   SET email = LOWER(email) WHERE email != LOWER(email);
UPDATE invites SET email = LOWER(email) WHERE email != LOWER(email);

-- +goose Down
