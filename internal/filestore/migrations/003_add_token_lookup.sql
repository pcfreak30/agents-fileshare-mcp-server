-- +goose Up
ALTER TABLE agents ADD COLUMN token_lookup TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_agents_token_lookup ON agents(token_lookup);

-- +goose Down
DROP INDEX IF EXISTS idx_agents_token_lookup;
-- SQLite doesn't support DROP COLUMN before 3.35; leave the column on downgrade.
