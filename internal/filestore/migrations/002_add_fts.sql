-- +goose Up
CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
    filename,
    content='files',
    content_rowid='rowid',
    tokenize='unicode61'
);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS files_fts_insert AFTER INSERT ON files BEGIN
    INSERT INTO files_fts(rowid, filename) VALUES (new.rowid, new.filename);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS files_fts_delete AFTER DELETE ON files BEGIN
    INSERT INTO files_fts(files_fts, rowid, filename) VALUES ('delete', old.rowid, old.filename);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS files_fts_update AFTER UPDATE ON files BEGIN
    INSERT INTO files_fts(files_fts, rowid, filename) VALUES ('delete', old.rowid, old.filename);
    INSERT INTO files_fts(rowid, filename) VALUES (new.rowid, new.filename);
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS files_fts_update;
DROP TRIGGER IF EXISTS files_fts_delete;
DROP TRIGGER IF EXISTS files_fts_insert;
DROP TABLE IF EXISTS files_fts;
