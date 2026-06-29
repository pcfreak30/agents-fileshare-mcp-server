package mcpserver

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/agent"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/config"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/osutil"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/share"
	"go.uber.org/zap"
)

type Server struct {
	store  filestore.FileStore
	agents *agent.Manager
	cfg    *config.Config
	log    *zap.Logger
	fs     osutil.FileSystem
}

func NewServer(store filestore.FileStore, agents *agent.Manager, cfg *config.Config, log *zap.Logger) *Server {
	return &Server{store: store, agents: agents, cfg: cfg, log: log, fs: osutil.RealFS{}}
}

func (s *Server) getAgentFromCtx(ctx context.Context) (*filestore.Agent, error) {
	sessionID, _ := ctx.Value(sessionIDKey{}).(string)
	if sessionID == "" {
		return nil, fmt.Errorf(model.ErrNotAuthenticated)
	}
	a, err := s.agents.GetBySession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("agent lookup: %w", err)
	}
	if a == nil {
		return nil, fmt.Errorf(model.ErrNotAuthenticated)
	}
	return a, nil
}

func (s *Server) getFile(fileID string, op string) (*filestore.FileMetadata, error) {
	f, err := s.store.GetFileByID(fileID)
	if err != nil {
		s.log.Error(op, zap.Error(err))
		return nil, fmt.Errorf("failed to %s", op)
	}
	if f == nil {
		return nil, fmt.Errorf(model.ErrFileNotFound)
	}
	return f, nil
}

func (s *Server) logAndFail(op string, err error) error {
	s.log.Error(op, zap.Error(err))
	return fmt.Errorf("failed to %s", op)
}

func ensureSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (s *Server) HandlePrepareUpload(ctx context.Context, in PrepareUploadInput) (*PrepareUploadOutput, error) {
	ag, err := s.getAgentFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	agentID := ag.AgentID

	filename := filestore.SanitizeFilename(in.Filename)
	if filename == "" {
		return nil, fmt.Errorf("filename is required")
	}

	contentType := in.ContentType
	if contentType == "" {
		return nil, fmt.Errorf("content_type is required")
	}

	visibility, err := filestore.ParseVisibility(in.Visibility)
	if err != nil {
		return nil, err
	}

	ttl, err := filestore.ParseTTL(in.TTL, s.cfg.DefaultTTL, s.cfg.MaxTTL)
	if err != nil {
		return nil, err
	}

	idBytes := make([]byte, 4)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, s.logAndFail("generate file id", err)
	}
	fileID := "f_" + hex.EncodeToString(idBytes)

	shareID := share.GenerateID(s.cfg.ShareIDLength)

	var downloadToken sql.NullString
	if visibility == model.VisToken {
		dtBytes := make([]byte, 16)
		if _, err := rand.Read(dtBytes); err != nil {
			return nil, s.logAndFail("generate download token", err)
		}
		downloadToken = sql.NullString{String: hex.EncodeToString(dtBytes), Valid: true}
	}

	expiresAt := time.Now().UTC().Add(ttl)

	if err := s.store.CreateFile(fileID, shareID, agentID, filename, contentType, visibility, int(ttl.Seconds()), downloadToken, expiresAt); err != nil {
		return nil, s.logAndFail("create file", err)
	}

	uploadURL := s.cfg.UploadURL(fileID)

	return &PrepareUploadOutput{
		FileID:       fileID,
		UploadURL:    uploadURL,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
		MaxSizeBytes: s.cfg.MaxFileSize,
	}, nil
}

func (s *Server) HandleListFiles(ctx context.Context, in ListFilesInput) (*ListFilesOutput, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	status := model.FileStatus(in.Status)
	if status == "" {
		status = model.StatusReady
	}

	agentFilter := in.Agent
	files, total, err := s.store.ListFiles(agentFilter, status, limit, in.Offset)
	if err != nil {
		return nil, s.logAndFail("list files", err)
	}

	return &ListFilesOutput{Files: ensureSlice(files), Total: total}, nil
}

func (s *Server) HandleGetFileInfo(ctx context.Context, in GetFileInfoInput) (*GetFileInfoOutput, error) {
	f, err := s.getFile(in.FileID, "get file info")
	if err != nil {
		return nil, err
	}

	out := &GetFileInfoOutput{
		FileID:      f.FileID,
		ShareID:     f.ShareID,
		Filename:    f.Filename,
		ContentType: f.ContentType,
		Size:        f.Size,
		SHA256:      f.SHA256,
		Visibility:  f.Visibility,
		Status:      f.Status,
		UploadedBy:  f.AgentID,
		ExpiresAt:   f.ExpiresAt.Format(time.RFC3339),
		DownloadURL: s.cfg.DownloadURL(f.ShareID),
	}

	if f.UploadedAt.Valid {
		out.UploadedAt = f.UploadedAt.Time.Format(time.RFC3339)
	}
	if f.Visibility == model.VisToken {
		out.DownloadToken = f.DownloadToken
	}

	return out, nil
}

func (s *Server) HandleGetFileLink(ctx context.Context, in GetFileLinkInput) (*GetFileLinkOutput, error) {
	f, err := s.getFile(in.FileID, "get file link")
	if err != nil {
		return nil, err
	}

	out := &GetFileLinkOutput{
		URL:       s.cfg.DownloadURL(f.ShareID),
		ExpiresAt: f.ExpiresAt.Format(time.RFC3339),
	}

	if f.Visibility == model.VisToken {
		out.Token = f.DownloadToken
	}

	return out, nil
}

func (s *Server) HandleDeleteFile(ctx context.Context, in DeleteFileInput) (*DeleteFileOutput, error) {
	ag, err := s.getAgentFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	agentID := ag.AgentID

	deleted, err := s.store.DeleteFile(in.FileID, agentID)
	if err != nil {
		return nil, s.logAndFail("delete file", err)
	}

	if deleted {
		filePath := s.cfg.FilePath(in.FileID)
		if err := s.fs.Remove(filePath); err != nil && !os.IsNotExist(err) {
			s.log.Warn("remove deleted file from disk", zap.String("file_id", in.FileID), zap.Error(err))
		}
	}

	return &DeleteFileOutput{Deleted: deleted, FileID: in.FileID}, nil
}

func (s *Server) HandleSearchFiles(ctx context.Context, in SearchFilesInput) (*SearchFilesOutput, error) {
	if in.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	files, err := s.store.SearchFiles(in.Query, in.Agent, limit)
	if err != nil {
		return nil, s.logAndFail("search files", err)
	}

	return &SearchFilesOutput{Files: ensureSlice(files)}, nil
}

func (s *Server) HandleListAgents(ctx context.Context) (*ListAgentsOutput, error) {
	agents, err := s.store.ListAgents()
	if err != nil {
		return nil, s.logAndFail("list agents", err)
	}

	return &ListAgentsOutput{Agents: ensureSlice(agents)}, nil
}

func (s *Server) HandleWhoami(ctx context.Context) (*WhoamiOutput, error) {
	sessionID, _ := ctx.Value(sessionIDKey{}).(string)
	if sessionID == "" {
		return nil, fmt.Errorf(model.ErrNotAuthenticated)
	}

	agentID, agentToken, err := s.agents.RegisterOrGet(sessionID)
	if err != nil {
		return nil, fmt.Errorf("agent registration: %w", err)
	}

	return &WhoamiOutput{
		AgentID:   agentID,
		SessionID: sessionID,
		Token:     agentToken,
	}, nil
}

type sessionIDKey struct{}
