package filestore

import (
	"database/sql"
	"time"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
)

type Agent struct {
	AgentID   string    `json:"agent_id"`
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
}

type AgentInfo struct {
	AgentID   string    `json:"agent_id"`
	FileCount int       `json:"file_count"`
	TotalSize int64     `json:"total_size"`
	LastSeen  time.Time `json:"last_seen"`
}

type FileMetadata struct {
	FileID        string           `json:"file_id"`
	ShareID       string           `json:"share_id"`
	AgentID       string           `json:"agent_id"`
	Filename      string           `json:"filename"`
	ContentType   string           `json:"content_type"`
	Size          int64            `json:"size"`
	SHA256        string           `json:"sha256"`
	Visibility    model.Visibility `json:"visibility"`
	DownloadToken string           `json:"download_token,omitempty"`
	Status        model.FileStatus `json:"status"`
	TTLSeconds    int              `json:"ttl_seconds"`
	UploadedAt    NullTime         `json:"uploaded_at"`
	ExpiresAt     time.Time        `json:"expires_at"`
	CreatedAt     time.Time        `json:"created_at"`
}

type NullTime struct {
	Time  time.Time
	Valid bool
}

func (nt NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	return nt.Time.MarshalJSON()
}

type AgentStore interface {
	RegisterAgent(agentID, tokenHash, tokenLookup, sessionID string) error
	RotateAgentToken(agentID, tokenHash, tokenLookup string) error
	VerifyAnyToken(token string) (*Agent, error)
	GetAgentBySession(sessionID string) (*Agent, error)
}

type FileStore interface {
	CreateFile(fileID, shareID, agentID, filename, contentType string, visibility model.Visibility, ttlSeconds int, downloadToken sql.NullString, expiresAt time.Time) error
	CompleteUpload(fileID, sha256 string, size int64) error
	RevertToPending(fileID string) error
	GetFileByID(fileID string) (*FileMetadata, error)
	GetFileByShareID(shareID string) (*FileMetadata, error)
	GetFileByDownloadToken(token string) (*FileMetadata, error)
	DeleteFile(fileID, agentID string) (bool, error)
	ExpireFiles() (int, error)
	ExpirePendingFiles(olderThan time.Duration) (int, error)
	GetExpiredFileIDs() ([]string, error)
	PurgeStaleAgents(olderThan time.Duration) (int, error)
	ListFiles(agentID string, status model.FileStatus, limit, offset int) ([]FileMetadata, int, error)
	SearchFiles(query, agentID string, limit int) ([]FileMetadata, error)
	ListAgents() ([]AgentInfo, error)
}
