-- +goose Up
CREATE TABLE IF NOT EXISTS agents (
    agent_id    TEXT PRIMARY KEY,
    token_hash  TEXT NOT NULL,
    session_id  TEXT UNIQUE,
    created_at  DATETIME NOT NULL,
    last_seen   DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS files (
    file_id        TEXT PRIMARY KEY,
    share_id       TEXT UNIQUE NOT NULL,
    agent_id       TEXT NOT NULL REFERENCES agents(agent_id),
    filename       TEXT NOT NULL,
    content_type   TEXT NOT NULL,
    size           INTEGER NOT NULL DEFAULT 0,
    sha256         TEXT NOT NULL DEFAULT '',
    visibility     TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'agent', 'token')),
    download_token TEXT,
    status         TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'ready', 'expired', 'deleted')),
    ttl_seconds    INTEGER NOT NULL DEFAULT 259200,
    uploaded_at    DATETIME,
    expires_at     DATETIME NOT NULL,
    created_at     DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_files_agent     ON files(agent_id);
CREATE INDEX IF NOT EXISTS idx_files_status    ON files(status);
CREATE INDEX IF NOT EXISTS idx_files_expires   ON files(expires_at);
CREATE INDEX IF NOT EXISTS idx_files_share     ON files(share_id);
CREATE INDEX IF NOT EXISTS idx_agents_session  ON agents(session_id);

-- +goose Down
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS agents;
