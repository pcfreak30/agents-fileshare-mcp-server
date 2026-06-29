package filestore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
	"github.com/pressly/goose/v3"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"go.uber.org/zap"
)

type scannable interface {
	Scan(dest ...any) error
}

type Store struct {
	db   *sql.DB
	log  *zap.Logger
	stmt struct {
		registerAgent         *sql.Stmt
		rotateAgentToken      *sql.Stmt
		verifyAgentToken      *sql.Stmt
		getAgentByTokenLookup  *sql.Stmt
		getAgentBySession     *sql.Stmt
		getAgentByID          *sql.Stmt
		allAgents             *sql.Stmt
		updateAgentSession *sql.Stmt
		touchAgent        *sql.Stmt
		createFile        *sql.Stmt
		completeUpload    *sql.Stmt
		revertToPending   *sql.Stmt
		getFileByID       *sql.Stmt
		getFileByShareID  *sql.Stmt
		getFileByDLToken  *sql.Stmt
		deleteFile        *sql.Stmt
		expireFiles       *sql.Stmt
		expirePendingFiles *sql.Stmt
		listAgents        *sql.Stmt
		expiredFileIDs    *sql.Stmt
		purgeStaleAgents  *sql.Stmt
	}
}

func NewStore(dbPath string, log *zap.Logger) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, MigrationsFS)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create migration provider: %w", err)
	}
	results, err := provider.Up(context.Background())
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	if len(results) > 0 {
		log.Info("migrations applied", zap.Int("count", len(results)))
	}

	s := &Store{db: db, log: log}

	prep := func(q string) *sql.Stmt {
		st, err := db.Prepare(q)
		if err != nil {
			log.Fatal("prepare statement", zap.String("query", q), zap.Error(err))
		}
		return st
	}

	s.stmt.registerAgent = prep("INSERT INTO agents (agent_id, token_hash, token_lookup, session_id, created_at, last_seen) VALUES (?, ?, ?, ?, ?, ?)")
	s.stmt.rotateAgentToken = prep("UPDATE agents SET token_hash = ?, token_lookup = ?, last_seen = ? WHERE agent_id = ?")
	s.stmt.verifyAgentToken = prep("SELECT token_hash FROM agents WHERE agent_id = ?")
	s.stmt.getAgentByTokenLookup = prep("SELECT agent_id, token_hash, session_id, created_at, last_seen FROM agents WHERE token_lookup = ?")
	s.stmt.getAgentBySession = prep("SELECT agent_id, session_id, created_at, last_seen FROM agents WHERE session_id = ?")
	s.stmt.getAgentByID = prep("SELECT agent_id, session_id, created_at, last_seen FROM agents WHERE agent_id = ?")
	s.stmt.allAgents = prep("SELECT agent_id, token_hash, session_id, created_at, last_seen FROM agents")
	s.stmt.updateAgentSession = prep("UPDATE agents SET session_id = ?, last_seen = ? WHERE agent_id = ?")
	s.stmt.touchAgent = prep("UPDATE agents SET last_seen = ? WHERE agent_id = ?")
	s.stmt.createFile = prep("INSERT INTO files (file_id, share_id, agent_id, filename, content_type, visibility, download_token, status, ttl_seconds, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	s.stmt.completeUpload = prep("UPDATE files SET status = ?, sha256 = ?, size = ?, uploaded_at = ? WHERE file_id = ?")
	s.stmt.revertToPending = prep("UPDATE files SET status = ?, sha256 = '', size = 0, uploaded_at = NULL WHERE file_id = ?")
	s.stmt.getFileByID = prep("SELECT " + fileColumns + " FROM files WHERE file_id = ?")
	s.stmt.getFileByShareID = prep("SELECT " + fileColumns + " FROM files WHERE share_id = ?")
	s.stmt.getFileByDLToken = prep("SELECT " + fileColumns + " FROM files WHERE download_token = ? AND visibility = ?")
	s.stmt.deleteFile = prep("UPDATE files SET status = ? WHERE file_id = ? AND agent_id = ? AND status != ?")
	s.stmt.expireFiles = prep("UPDATE files SET status = ? WHERE expires_at <= ? AND status NOT IN (?, ?)")
	s.stmt.expirePendingFiles = prep("UPDATE files SET status = ? WHERE status = ? AND created_at < ?")
	s.stmt.listAgents = prep("SELECT a.agent_id, COUNT(f.file_id), COALESCE(SUM(f.size), 0), a.last_seen FROM agents a LEFT JOIN files f ON a.agent_id = f.agent_id AND f.status = ? GROUP BY a.agent_id ORDER BY a.last_seen DESC")
	s.stmt.expiredFileIDs = prep("SELECT file_id FROM files WHERE status = ?")
	s.stmt.purgeStaleAgents = prep("DELETE FROM agents WHERE last_seen < ? AND agent_id NOT IN (SELECT DISTINCT agent_id FROM files WHERE status IN (?, ?))")

	return s, nil
}

func (s *Store) Close() error {
	st := &s.stmt
	for _, st := range []*sql.Stmt{
		st.registerAgent, st.rotateAgentToken, st.verifyAgentToken, st.getAgentByTokenLookup, st.getAgentBySession, st.getAgentByID, st.allAgents,
		st.updateAgentSession, st.touchAgent, st.createFile, st.completeUpload,
		st.revertToPending, st.getFileByID, st.getFileByShareID, st.getFileByDLToken,
		st.deleteFile, st.expireFiles, st.expirePendingFiles, st.listAgents, st.expiredFileIDs,
		st.purgeStaleAgents,
	} {
		st.Close()
	}
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) RegisterAgent(agentID, tokenHash, tokenLookup, sessionID string) error {
	now := time.Now().UTC()
	_, err := s.stmt.registerAgent.Exec(agentID, tokenHash, tokenLookup, sessionID, now, now)
	return err
}

func (s *Store) RotateAgentToken(agentID, tokenHash, tokenLookup string) error {
	_, err := s.stmt.rotateAgentToken.Exec(tokenHash, tokenLookup, time.Now().UTC(), agentID)
	return err
}

func (s *Store) VerifyAgentToken(agentID, token string) (bool, error) {
	var hash string
	err := s.stmt.verifyAgentToken.QueryRow(agentID).Scan(&hash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(token)) != nil {
		return false, nil
	}
	return true, nil
}

func (s *Store) VerifyAnyToken(token string) (*Agent, error) {
	lookup := tokenLookupHash(token)
	row := s.stmt.getAgentByTokenLookup.QueryRow(lookup)
	var a Agent
	var hash string
	if err := row.Scan(&a.AgentID, &hash, &a.SessionID, &a.CreatedAt, &a.LastSeen); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(token)) != nil {
		return nil, nil
	}
	return &a, nil
}

func tokenLookupHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *Store) GetAgentBySession(sessionID string) (*Agent, error) {
	return s.scanAgent(s.stmt.getAgentBySession.QueryRow(sessionID))
}

func (s *Store) GetAgentByID(agentID string) (*Agent, error) {
	return s.scanAgent(s.stmt.getAgentByID.QueryRow(agentID))
}

func (s *Store) scanAgent(row *sql.Row) (*Agent, error) {
	var a Agent
	err := row.Scan(&a.AgentID, &a.SessionID, &a.CreatedAt, &a.LastSeen)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) UpdateAgentSession(agentID, sessionID string) error {
	_, err := s.stmt.updateAgentSession.Exec(sessionID, time.Now().UTC(), agentID)
	return err
}

func (s *Store) TouchAgent(agentID string) error {
	_, err := s.stmt.touchAgent.Exec(time.Now().UTC(), agentID)
	return err
}

func (s *Store) CreateFile(fileID, shareID, agentID, filename, contentType string, visibility model.Visibility, ttlSeconds int, downloadToken sql.NullString, expiresAt time.Time) error {
	now := time.Now().UTC()
	var dlToken any
	if downloadToken.Valid {
		dlToken = downloadToken.String
	} else {
		dlToken = nil
	}
	_, err := s.stmt.createFile.Exec(fileID, shareID, agentID, filename, contentType, visibility, dlToken, model.StatusPending, ttlSeconds, expiresAt, now)
	return err
}

func (s *Store) CompleteUpload(fileID, sha256 string, size int64) error {
	now := time.Now().UTC()
	_, err := s.stmt.completeUpload.Exec(model.StatusReady, sha256, size, now, fileID)
	return err
}

func (s *Store) RevertToPending(fileID string) error {
	_, err := s.stmt.revertToPending.Exec(model.StatusPending, fileID)
	return err
}

func (s *Store) GetExpiredFileIDs() ([]string, error) {
	rows, err := s.stmt.expiredFileIDs.Query(model.StatusExpired)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

const fileColumns = "file_id, share_id, agent_id, filename, content_type, size, sha256, visibility, download_token, status, ttl_seconds, uploaded_at, expires_at, created_at"

func prefixedColumns(alias string) string {
	parts := strings.Split(fileColumns, ", ")
	for i := range parts {
		parts[i] = alias + "." + parts[i]
	}
	return strings.Join(parts, ", ")
}

func (s *Store) GetFileByID(fileID string) (*FileMetadata, error) {
	return s.scanFile(s.stmt.getFileByID.QueryRow(fileID))
}

func (s *Store) GetFileByShareID(shareID string) (*FileMetadata, error) {
	return s.scanFile(s.stmt.getFileByShareID.QueryRow(shareID))
}

func (s *Store) GetFileByDownloadToken(token string) (*FileMetadata, error) {
	return s.scanFile(s.stmt.getFileByDLToken.QueryRow(token, model.VisToken))
}

func (s *Store) scanFile(row *sql.Row) (*FileMetadata, error) {
	f, err := scanFileFrom(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Store) ListFiles(agentID string, status model.FileStatus, limit, offset int) ([]FileMetadata, int, error) {
	var args []any
	var where []string

	if agentID != "" {
		where = append(where, "agent_id = ?")
		args = append(args, agentID)
	}
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	countQ := "SELECT COUNT(*) FROM files " + whereClause
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := s.db.QueryRow(countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := "SELECT " + fileColumns + " FROM files " + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	queryArgs := make([]any, 0, len(args)+2)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := s.db.Query(q, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	fileRows, _, err := scanFileRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return fileRows, total, nil
}

func (s *Store) SearchFiles(query, agentID string, limit int) ([]FileMetadata, error) {
	ftsQuery := sanitizeFTSQuery(query)

	var args []any
	var conds []string

	conds = append(conds, "f.status = ?")
	args = append(args, model.StatusReady)

	if ftsQuery != "" {
		conds = append(conds, "fts MATCH ?")
		args = append(args, ftsQuery)
	}
	if agentID != "" {
		conds = append(conds, "f.agent_id = ?")
		args = append(args, agentID)
	}

	from := "files f"
	if ftsQuery != "" {
		from = "files f JOIN files_fts fts ON f.rowid = fts.rowid"
	}

	where := "WHERE " + strings.Join(conds, " AND ")
	orderBy := "ORDER BY f.created_at DESC"
	if ftsQuery != "" {
		orderBy = "ORDER BY fts.rank"
	}

	q := "SELECT " + prefixedColumns("f") + " FROM " + from + " " + where + " " + orderBy + " LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files, _, err := scanFileRows(rows)
	return files, err
}

func scanFileFrom(s scannable) (*FileMetadata, error) {
	var f FileMetadata
	var uploadedAt sql.NullTime
	var downloadToken sql.NullString
	var visibility string
	var status string
	if err := s.Scan(
		&f.FileID, &f.ShareID, &f.AgentID, &f.Filename, &f.ContentType,
		&f.Size, &f.SHA256, &visibility, &downloadToken, &status,
		&f.TTLSeconds, &uploadedAt, &f.ExpiresAt, &f.CreatedAt,
	); err != nil {
		return nil, err
	}
	f.Visibility = model.Visibility(visibility)
	f.Status = model.FileStatus(status)
	if uploadedAt.Valid {
		f.UploadedAt = NullTime{Time: uploadedAt.Time, Valid: true}
	}
	if downloadToken.Valid {
		f.DownloadToken = downloadToken.String
	}
	return &f, nil
}

func scanFileRows(rows *sql.Rows) ([]FileMetadata, int, error) {
	var files []FileMetadata
	for rows.Next() {
		f, err := scanFileFrom(rows)
		if err != nil {
			return nil, 0, err
		}
		files = append(files, *f)
	}
	return files, len(files), nil
}

func sanitizeFTSQuery(q string) string {
	if q == "" {
		return ""
	}
	parts := strings.Fields(q)
	for i, p := range parts {
		parts[i] = strings.Trim(p, `"'*:+-^(){}[]`)
		if parts[i] != "" {
			parts[i] += "*"
		}
	}
	return strings.Join(parts, " ")
}

func (s *Store) DeleteFile(fileID, agentID string) (bool, error) {
	result, err := s.stmt.deleteFile.Exec(model.StatusDeleted, fileID, agentID, model.StatusDeleted)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) ExpireFiles() (int, error) {
	result, err := s.stmt.expireFiles.Exec(model.StatusExpired, time.Now().UTC(), model.StatusExpired, model.StatusDeleted)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

func (s *Store) ExpirePendingFiles(olderThan time.Duration) (int, error) {
	cutoff := time.Now().UTC().Add(-olderThan)
	result, err := s.stmt.expirePendingFiles.Exec(model.StatusExpired, model.StatusPending, cutoff)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

func (s *Store) PurgeStaleAgents(olderThan time.Duration) (int, error) {
	cutoff := time.Now().UTC().Add(-olderThan)
	result, err := s.stmt.purgeStaleAgents.Exec(cutoff, model.StatusReady, model.StatusPending)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

func (s *Store) ListAgents() ([]AgentInfo, error) {
	rows, err := s.stmt.listAgents.Query(model.StatusReady)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []AgentInfo
	for rows.Next() {
		var a AgentInfo
		if err := rows.Scan(&a.AgentID, &a.FileCount, &a.TotalSize, &a.LastSeen); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

func SanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "..", "_")
	if len(name) > 255 {
		name = name[:255]
	}
	return name
}

func ParseTTL(ttlStr string, defaultTTL, maxTTL time.Duration) (time.Duration, error) {
	if ttlStr == "" {
		return defaultTTL, nil
	}
	d, err := time.ParseDuration(ttlStr)
	if err != nil {
		return 0, fmt.Errorf("invalid TTL: %w", err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("TTL must be positive")
	}
	if d > maxTTL {
		return 0, fmt.Errorf("TTL exceeds maximum of %s", maxTTL)
	}
	return d, nil
}

func ParseVisibility(v string) (model.Visibility, error) {
	switch v {
	case "", string(model.VisPublic):
		return model.VisPublic, nil
	case string(model.VisAgent):
		return model.VisAgent, nil
	case string(model.VisToken):
		return model.VisToken, nil
	default:
		return "", fmt.Errorf("invalid visibility: %s (must be %s, %s, or %s)", v, model.VisPublic, model.VisAgent, model.VisToken)
	}
}
