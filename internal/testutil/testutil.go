package testutil

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/config"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/osutil"
	"go.uber.org/zap"
)

type MockFileStore struct {
	CreateFileFn             func(fileID, shareID, agentID, filename, contentType string, visibility model.Visibility, ttlSeconds int, downloadToken sql.NullString, expiresAt time.Time) error
	CompleteUploadFn        func(fileID, sha256 string, size int64) error
	RevertToPendingFn       func(fileID string) error
	GetFileByIDFn           func(fileID string) (*filestore.FileMetadata, error)
	GetFileByShareIDFn      func(shareID string) (*filestore.FileMetadata, error)
	GetFileByDownloadTokenFn func(token string) (*filestore.FileMetadata, error)
	DeleteFileFn            func(fileID, agentID string) (bool, error)
	ExpireFilesFn           func() (int, error)
	GetExpiredFileIDsFn     func() ([]string, error)
	ListFilesFn             func(agentID string, status model.FileStatus, limit, offset int) ([]filestore.FileMetadata, int, error)
	SearchFilesFn           func(query, agentID string, limit int) ([]filestore.FileMetadata, error)
	ListAgentsFn            func() ([]filestore.AgentInfo, error)
}

func (m *MockFileStore) CreateFile(fileID, shareID, agentID, filename, contentType string, visibility model.Visibility, ttlSeconds int, downloadToken sql.NullString, expiresAt time.Time) error {
	if m.CreateFileFn != nil {
		return m.CreateFileFn(fileID, shareID, agentID, filename, contentType, visibility, ttlSeconds, downloadToken, expiresAt)
	}
	return nil
}

func (m *MockFileStore) CompleteUpload(fileID, sha256 string, size int64) error {
	if m.CompleteUploadFn != nil {
		return m.CompleteUploadFn(fileID, sha256, size)
	}
	return nil
}

func (m *MockFileStore) RevertToPending(fileID string) error {
	if m.RevertToPendingFn != nil {
		return m.RevertToPendingFn(fileID)
	}
	return nil
}

func (m *MockFileStore) GetFileByID(fileID string) (*filestore.FileMetadata, error) {
	if m.GetFileByIDFn != nil {
		return m.GetFileByIDFn(fileID)
	}
	return nil, nil
}

func (m *MockFileStore) GetFileByShareID(shareID string) (*filestore.FileMetadata, error) {
	if m.GetFileByShareIDFn != nil {
		return m.GetFileByShareIDFn(shareID)
	}
	return nil, nil
}

func (m *MockFileStore) GetFileByDownloadToken(token string) (*filestore.FileMetadata, error) {
	if m.GetFileByDownloadTokenFn != nil {
		return m.GetFileByDownloadTokenFn(token)
	}
	return nil, nil
}

func (m *MockFileStore) DeleteFile(fileID, agentID string) (bool, error) {
	if m.DeleteFileFn != nil {
		return m.DeleteFileFn(fileID, agentID)
	}
	return false, nil
}

func (m *MockFileStore) ExpireFiles() (int, error) {
	if m.ExpireFilesFn != nil {
		return m.ExpireFilesFn()
	}
	return 0, nil
}

func (m *MockFileStore) GetExpiredFileIDs() ([]string, error) {
	if m.GetExpiredFileIDsFn != nil {
		return m.GetExpiredFileIDsFn()
	}
	return nil, nil
}

func (m *MockFileStore) ListFiles(agentID string, status model.FileStatus, limit, offset int) ([]filestore.FileMetadata, int, error) {
	if m.ListFilesFn != nil {
		return m.ListFilesFn(agentID, status, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockFileStore) SearchFiles(query, agentID string, limit int) ([]filestore.FileMetadata, error) {
	if m.SearchFilesFn != nil {
		return m.SearchFilesFn(query, agentID, limit)
	}
	return nil, nil
}

func (m *MockFileStore) ListAgents() ([]filestore.AgentInfo, error) {
	if m.ListAgentsFn != nil {
		return m.ListAgentsFn()
	}
	return nil, nil
}

func NewTestConfig() *config.Config {
	return &config.Config{
		ListenAddr:      ":8080",
		DataDir:         "/tmp/test-data",
		MaxFileSize:     1024 * 1024,
		DefaultTTL:      72 * time.Hour,
		MaxTTL:          168 * time.Hour,
		CleanupInterval: 1 * time.Minute,
		ShareIDLength:   8,
		BaseURL:         "http://localhost:8080",
		UploadRateLimit: 10,
	}
}

func NewTestLogger() *zap.Logger {
	return zap.NewNop()
}

func NewTestStore() *MockFileStore {
	return &MockFileStore{}
}

func ReadyFile(fileID, shareID, agentID, filename, contentType string, visibility model.Visibility) *filestore.FileMetadata {
	now := time.Now().UTC()
	return &filestore.FileMetadata{
		FileID:      fileID,
		ShareID:     shareID,
		AgentID:     agentID,
		Filename:    filename,
		ContentType: contentType,
		Size:        100,
		SHA256:      "abc123",
		Visibility:  visibility,
		Status:      model.StatusReady,
		TTLSeconds:  259200,
		UploadedAt:  filestore.NullTime{Time: now, Valid: true},
		ExpiresAt:   now.Add(72 * time.Hour),
		CreatedAt:   now.Add(-1 * time.Minute),
	}
}

func PendingFile(fileID, shareID, agentID, filename, contentType string, visibility model.Visibility) *filestore.FileMetadata {
	now := time.Now().UTC()
	return &filestore.FileMetadata{
		FileID:      fileID,
		ShareID:     shareID,
		AgentID:     agentID,
		Filename:    filename,
		ContentType: contentType,
		Visibility:  visibility,
		Status:      model.StatusPending,
		TTLSeconds:  259200,
		ExpiresAt:   now.Add(72 * time.Hour),
		CreatedAt:   now,
	}
}

type MemFS struct {
	Files    map[string][]byte
	StatErr  map[string]error
	RemoveFn func(name string) error
}

func NewMemFS() *MemFS {
	return &MemFS{
		Files:   make(map[string][]byte),
		StatErr: make(map[string]error),
	}
}

func (m *MemFS) Create(name string) (*os.File, error)       { return nil, nil }
func (m *MemFS) MkdirAll(path string, perm os.FileMode) error { return nil }
func (m *MemFS) Open(name string) (*os.File, error)         { return nil, nil }

func (m *MemFS) Remove(name string) error {
	if m.RemoveFn != nil {
		return m.RemoveFn(name)
	}
	delete(m.Files, name)
	return nil
}

func (m *MemFS) Rename(oldpath, newpath string) error {
	if data, ok := m.Files[oldpath]; ok {
		m.Files[newpath] = data
		delete(m.Files, oldpath)
	}
	return nil
}

func (m *MemFS) Stat(name string) (os.FileInfo, error) {
	if err, ok := m.StatErr[name]; ok {
		return nil, err
	}
	if _, ok := m.Files[name]; ok {
		return &mockFileInfo{name: name, size: int64(len(m.Files[name]))}, nil
	}
	return nil, os.ErrNotExist
}

type mockFileInfo struct {
	name string
	size int64
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() any            { return nil }

var _ osutil.FileSystem = (*MemFS)(nil)
var _ filestore.FileStore = (*MockFileStore)(nil)

func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}
