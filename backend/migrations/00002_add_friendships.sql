-- +goose Up
-- App-internal friendships. Discord OAuth cannot expose a user's real Discord
-- friends list (the relationships.read scope is restricted), so friendship is a
-- request -> accept relationship between two users that have both logged in.
-- A single row represents the relationship in one direction (requester ->
-- addressee); status flips from 'pending' to 'accepted'. The DAO guards against
-- creating a reverse duplicate.
CREATE TABLE IF NOT EXISTS friendships (
    requester_id TEXT NOT NULL, -- discord_id that sent the request
    addressee_id TEXT NOT NULL, -- discord_id that received it
    status       TEXT NOT NULL DEFAULT 'pending', -- 'pending' | 'accepted'
    created_at   TIMESTAMP DEFAULT now(),
    PRIMARY KEY (requester_id, addressee_id),
    CHECK (requester_id <> addressee_id)
);

-- encounter_users links friends to an encounter as shared-edit members. It is in
-- the start.sql baseline, but recreate it defensively so databases that predate
-- it (or that lost it) converge instead of drifting.
CREATE TABLE IF NOT EXISTS encounter_users (
    encounter_id INTEGER REFERENCES encounters(id) ON DELETE CASCADE,
    user_id      TEXT NOT NULL, -- Discord user ID
    PRIMARY KEY (encounter_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS friendships;
