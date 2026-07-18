-- +goose Up
-- The friends feature looks users up by username (GetUserByUsername), relying on
-- Discord usernames being globally unique under the post-2023 (discriminator-less)
-- system. Enforce that assumption in the DB so a duplicate can never silently
-- resolve to the wrong account. A unique index is used (with IF NOT EXISTS) so the
-- migration no-ops on databases that already have it.
CREATE UNIQUE INDEX IF NOT EXISTS users_username_key ON users (username);

-- +goose Down
DROP INDEX IF EXISTS users_username_key;
