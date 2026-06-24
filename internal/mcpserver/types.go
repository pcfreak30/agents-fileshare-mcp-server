package mcpserver

import (
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
)

type PrepareUploadInput struct {
	Filename    string `json:"filename" jsonschema:"required,description=Name of the file to upload (max 255 chars)"`
	ContentType string `json:"content_type" jsonschema:"required,description=MIME type of the file"`
	TTL         string `json:"ttl" jsonschema:"description=Time-to-live duration string (e.g. 24h, 72h). Default: 72h, Max: 168h"`
	Visibility  string `json:"visibility" jsonschema:"description=Access visibility: public, agent, or token. Default: public"`
}

type PrepareUploadOutput struct {
	FileID       string `json:"file_id"`
	UploadURL    string `json:"upload_url"`
	ExpiresAt    string `json:"expires_at"`
	MaxSizeBytes int64  `json:"max_size_bytes"`
}

type ListFilesInput struct {
	Agent  string `json:"agent" jsonschema:"description=Filter by agent_id"`
	Status string `json:"status" jsonschema:"description=Filter by status: ready, pending, or expired. Default: ready"`
	Limit  int    `json:"limit" jsonschema:"description=Maximum results. Default: 50, Max: 200"`
	Offset int    `json:"offset" jsonschema:"description=Result offset. Default: 0"`
}

type ListFilesOutput struct {
	Files []filestore.FileMetadata `json:"files"`
	Total int                      `json:"total"`
}

type GetFileInfoInput struct {
	FileID string `json:"file_id" jsonschema:"required,description=The file ID to get info for"`
}

type GetFileInfoOutput struct {
	FileID        string `json:"file_id"`
	ShareID       string `json:"share_id"`
	Filename      string `json:"filename"`
	ContentType   string `json:"content_type"`
	Size          int64  `json:"size"`
	SHA256        string `json:"sha256"`
	Visibility    model.Visibility `json:"visibility"`
	Status        model.FileStatus `json:"status"`
	UploadedBy    string `json:"uploaded_by"`
	UploadedAt    string `json:"uploaded_at"`
	ExpiresAt     string `json:"expires_at"`
	DownloadURL   string `json:"download_url"`
	DownloadToken string `json:"download_token,omitempty"`
}

type GetFileLinkInput struct {
	FileID string `json:"file_id" jsonschema:"required,description=The file ID to get a link for"`
}

type GetFileLinkOutput struct {
	URL       string `json:"url"`
	Token     string `json:"token,omitempty"`
	ExpiresAt string `json:"expires_at"`
}

type DeleteFileInput struct {
	FileID string `json:"file_id" jsonschema:"required,description=The file ID to delete"`
}

type DeleteFileOutput struct {
	Deleted bool   `json:"deleted"`
	FileID  string `json:"file_id"`
}

type SearchFilesInput struct {
	Query string `json:"query" jsonschema:"required,description=Filename substring to search for"`
	Agent string `json:"agent" jsonschema:"description=Filter by agent_id"`
	Limit int    `json:"limit" jsonschema:"description=Maximum results. Default: 20, Max: 100"`
}

type SearchFilesOutput struct {
	Files []filestore.FileMetadata `json:"files"`
}

type ListAgentsOutput struct {
	Agents []filestore.AgentInfo `json:"agents"`
}

type WhoamiOutput struct {
	AgentID   string `json:"agent_id"`
	SessionID string `json:"session_id"`
}
