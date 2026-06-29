package mcpserver

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/agent"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/testutil"
)

func newTestServer(store *testutil.MockFileStore) *Server {
	agentStore := &mockAgentStore{
		getAgentBySessionFn: func(sessionID string) (*filestore.Agent, error) {
			return &filestore.Agent{AgentID: "agent_test", SessionID: sessionID}, nil
		},
	}
	agents := agent.NewManager(agentStore)
	return &Server{
		store:  store,
		agents: agents,
		cfg:    testutil.NewTestConfig(),
		log:    testutil.NewTestLogger(),
		fs:     testutil.NewMemFS(),
	}
}

func ctxWithSession(sessionID string) context.Context {
	return context.WithValue(context.Background(), sessionIDKey{}, sessionID)
}

func TestServer_HandleListFiles(t *testing.T) {
	store := testutil.NewTestStore()
	store.ListFilesFn = func(agentID string, status model.FileStatus, limit, offset int) ([]filestore.FileMetadata, int, error) {
		return []filestore.FileMetadata{
			*testutil.ReadyFile("f_1", "s1", "a1", "test.txt", "text/plain", model.VisPublic),
		}, 1, nil
	}

	srv := newTestServer(store)
	out, err := srv.HandleListFiles(context.Background(), ListFilesInput{})
	if err != nil {
		t.Fatalf("HandleListFiles: %v", err)
	}
	if out.Total != 1 {
		t.Errorf("Total = %d, want 1", out.Total)
	}
	if len(out.Files) != 1 {
		t.Errorf("len(Files) = %d, want 1", len(out.Files))
	}
}

func TestServer_HandleListFiles_DefaultLimit(t *testing.T) {
	store := testutil.NewTestStore()
	var gotLimit int
	store.ListFilesFn = func(agentID string, status model.FileStatus, limit, offset int) ([]filestore.FileMetadata, int, error) {
		gotLimit = limit
		return nil, 0, nil
	}

	srv := newTestServer(store)
	srv.HandleListFiles(context.Background(), ListFilesInput{})
	if gotLimit != 50 {
		t.Errorf("default limit = %d, want 50", gotLimit)
	}
}

func TestServer_HandleListFiles_MaxLimit(t *testing.T) {
	store := testutil.NewTestStore()
	var gotLimit int
	store.ListFilesFn = func(agentID string, status model.FileStatus, limit, offset int) ([]filestore.FileMetadata, int, error) {
		gotLimit = limit
		return nil, 0, nil
	}

	srv := newTestServer(store)
	srv.HandleListFiles(context.Background(), ListFilesInput{Limit: 500})
	if gotLimit != 200 {
		t.Errorf("clamped limit = %d, want 200", gotLimit)
	}
}

func TestServer_HandleListFiles_StoreError(t *testing.T) {
	store := testutil.NewTestStore()
	store.ListFilesFn = func(agentID string, status model.FileStatus, limit, offset int) ([]filestore.FileMetadata, int, error) {
		return nil, 0, errors.New("db error")
	}

	srv := newTestServer(store)
	_, err := srv.HandleListFiles(context.Background(), ListFilesInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestServer_HandleGetFileInfo(t *testing.T) {
	store := testutil.NewTestStore()
	store.GetFileByIDFn = func(fileID string) (*filestore.FileMetadata, error) {
		return testutil.ReadyFile("f_1", "s1", "a1", "test.txt", "text/plain", model.VisPublic), nil
	}

	srv := newTestServer(store)
	out, err := srv.HandleGetFileInfo(context.Background(), GetFileInfoInput{FileID: "f_1"})
	if err != nil {
		t.Fatalf("HandleGetFileInfo: %v", err)
	}
	if out.FileID != "f_1" {
		t.Errorf("FileID = %q, want %q", out.FileID, "f_1")
	}
	if out.Filename != "test.txt" {
		t.Errorf("Filename = %q, want %q", out.Filename, "test.txt")
	}
}

func TestServer_HandleGetFileInfo_NotFound(t *testing.T) {
	store := testutil.NewTestStore()
	store.GetFileByIDFn = func(fileID string) (*filestore.FileMetadata, error) {
		return nil, nil
	}

	srv := newTestServer(store)
	_, err := srv.HandleGetFileInfo(context.Background(), GetFileInfoInput{FileID: "f_404"})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestServer_HandleGetFileLink(t *testing.T) {
	store := testutil.NewTestStore()
	store.GetFileByIDFn = func(fileID string) (*filestore.FileMetadata, error) {
		return testutil.ReadyFile("f_1", "s1", "a1", "test.txt", "text/plain", model.VisPublic), nil
	}

	srv := newTestServer(store)
	out, err := srv.HandleGetFileLink(context.Background(), GetFileLinkInput{FileID: "f_1"})
	if err != nil {
		t.Fatalf("HandleGetFileLink: %v", err)
	}
	if out.URL != "http://localhost:8080/f/s1" {
		t.Errorf("URL = %q, want http://localhost:8080/f/s1", out.URL)
	}
}

func TestServer_HandleGetFileLink_TokenVisibility(t *testing.T) {
	store := testutil.NewTestStore()
	store.GetFileByIDFn = func(fileID string) (*filestore.FileMetadata, error) {
		f := testutil.ReadyFile("f_1", "s1", "a1", "test.txt", "text/plain", model.VisToken)
		f.DownloadToken = "tok123"
		return f, nil
	}

	srv := newTestServer(store)
	out, err := srv.HandleGetFileLink(context.Background(), GetFileLinkInput{FileID: "f_1"})
	if err != nil {
		t.Fatalf("HandleGetFileLink: %v", err)
	}
	if out.Token != "tok123" {
		t.Errorf("Token = %q, want %q", out.Token, "tok123")
	}
}

func TestServer_HandleDeleteFile(t *testing.T) {
	store := testutil.NewTestStore()
	store.DeleteFileFn = func(fileID, agentID string) (bool, error) {
		return true, nil
	}

	srv := newTestServer(store)
	ctx := ctxWithSession("sess_del")
	out, err := srv.HandleDeleteFile(ctx, DeleteFileInput{FileID: "f_del"})
	if err != nil {
		t.Fatalf("HandleDeleteFile: %v", err)
	}
	if !out.Deleted {
		t.Error("expected Deleted=true")
	}
}

func TestServer_HandleDeleteFile_NotOwner(t *testing.T) {
	store := testutil.NewTestStore()
	store.DeleteFileFn = func(fileID, agentID string) (bool, error) {
		return false, nil
	}

	srv := newTestServer(store)
	ctx := ctxWithSession("sess_del2")
	out, err := srv.HandleDeleteFile(ctx, DeleteFileInput{FileID: "f_del2"})
	if err != nil {
		t.Fatalf("HandleDeleteFile: %v", err)
	}
	if out.Deleted {
		t.Error("expected Deleted=false for non-owner")
	}
}

func TestServer_HandleDeleteFile_NotAuthenticated(t *testing.T) {
	store := testutil.NewTestStore()
	srv := newTestServer(store)

	_, err := srv.HandleDeleteFile(context.Background(), DeleteFileInput{FileID: "f_del3"})
	if err == nil {
		t.Fatal("expected error for no auth")
	}
}

func TestServer_HandleSearchFiles(t *testing.T) {
	store := testutil.NewTestStore()
	store.SearchFilesFn = func(query, agentID string, limit int) ([]filestore.FileMetadata, error) {
		return []filestore.FileMetadata{
			*testutil.ReadyFile("f_s1", "ss1", "a1", "report.pdf", "application/pdf", model.VisPublic),
		}, nil
	}

	srv := newTestServer(store)
	out, err := srv.HandleSearchFiles(context.Background(), SearchFilesInput{Query: "report"})
	if err != nil {
		t.Fatalf("HandleSearchFiles: %v", err)
	}
	if len(out.Files) != 1 {
		t.Errorf("len(Files) = %d, want 1", len(out.Files))
	}
}

func TestServer_HandleSearchFiles_EmptyQuery(t *testing.T) {
	store := testutil.NewTestStore()
	srv := newTestServer(store)

	_, err := srv.HandleSearchFiles(context.Background(), SearchFilesInput{Query: ""})
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestServer_HandleSearchFiles_DefaultLimit(t *testing.T) {
	store := testutil.NewTestStore()
	var gotLimit int
	store.SearchFilesFn = func(query, agentID string, limit int) ([]filestore.FileMetadata, error) {
		gotLimit = limit
		return nil, nil
	}

	srv := newTestServer(store)
	srv.HandleSearchFiles(context.Background(), SearchFilesInput{Query: "test"})
	if gotLimit != 20 {
		t.Errorf("default limit = %d, want 20", gotLimit)
	}
}

func TestServer_HandleSearchFiles_MaxLimit(t *testing.T) {
	store := testutil.NewTestStore()
	var gotLimit int
	store.SearchFilesFn = func(query, agentID string, limit int) ([]filestore.FileMetadata, error) {
		gotLimit = limit
		return nil, nil
	}

	srv := newTestServer(store)
	srv.HandleSearchFiles(context.Background(), SearchFilesInput{Query: "test", Limit: 500})
	if gotLimit != 100 {
		t.Errorf("clamped limit = %d, want 100", gotLimit)
	}
}

func TestServer_HandleListAgents(t *testing.T) {
	store := testutil.NewTestStore()
	store.ListAgentsFn = func() ([]filestore.AgentInfo, error) {
		return []filestore.AgentInfo{
			{AgentID: "a1", FileCount: 5, TotalSize: 1000, LastSeen: time.Now()},
		}, nil
	}

	srv := newTestServer(store)
	out, err := srv.HandleListAgents(context.Background())
	if err != nil {
		t.Fatalf("HandleListAgents: %v", err)
	}
	if len(out.Agents) != 1 {
		t.Errorf("len(Agents) = %d, want 1", len(out.Agents))
	}
}

func TestServer_HandleWhoami_NotAuthenticated(t *testing.T) {
	store := testutil.NewTestStore()
	srv := newTestServer(store)

	_, err := srv.HandleWhoami(context.Background())
	if err == nil {
		t.Fatal("expected error for no auth")
	}
}

func TestServer_HandleWhoami_NewAgent(t *testing.T) {
	store := testutil.NewTestStore()
	var registered bool
	agentStore := &mockAgentStore{
		getAgentBySessionFn: func(sessionID string) (*filestore.Agent, error) {
			return nil, nil // new agent
		},
		registerAgentFn: func(agentID, tokenHash, tokenLookup, sessionID string) error {
			registered = true
			if agentID == "" {
				t.Error("expected non-empty agentID")
			}
			if tokenHash == "" {
				t.Error("expected non-empty tokenHash")
			}
			if tokenLookup == "" {
				t.Error("expected non-empty tokenLookup")
			}
			return nil
		},
	}
	agents := agent.NewManager(agentStore)
	srv := &Server{
		store:  store,
		agents: agents,
		cfg:    testutil.NewTestConfig(),
		log:    testutil.NewTestLogger(),
		fs:     testutil.NewMemFS(),
	}
	ctx := ctxWithSession("sess_new")

	out, err := srv.HandleWhoami(ctx)
	if err != nil {
		t.Fatalf("HandleWhoami: %v", err)
	}
	if !registered {
		t.Error("expected RegisterAgent to be called")
	}
	if out.AgentID == "" {
		t.Error("expected non-empty AgentID")
	}
	if out.SessionID != "sess_new" {
		t.Errorf("SessionID = %q, want %q", out.SessionID, "sess_new")
	}
	if out.Token == "" {
		t.Error("expected non-empty Token for new agent")
	}
}

func TestServer_HandleWhoami_ExistingAgent(t *testing.T) {
	store := testutil.NewTestStore()
	agentStore := &mockAgentStore{
		getAgentBySessionFn: func(sessionID string) (*filestore.Agent, error) {
			return &filestore.Agent{AgentID: "agent_existing", SessionID: sessionID}, nil
		},
	}
	agents := agent.NewManager(agentStore)
	srv := &Server{
		store:  store,
		agents: agents,
		cfg:    testutil.NewTestConfig(),
		log:    testutil.NewTestLogger(),
		fs:     testutil.NewMemFS(),
	}
	ctx := ctxWithSession("sess_existing")

	out, err := srv.HandleWhoami(ctx)
	if err != nil {
		t.Fatalf("HandleWhoami: %v", err)
	}
	if out.AgentID != "agent_existing" {
		t.Errorf("AgentID = %q, want %q", out.AgentID, "agent_existing")
	}
	if out.Token != "" {
		t.Errorf("expected empty Token for existing agent, got %q", out.Token)
	}
}

func TestServer_HandlePrepareUpload(t *testing.T) {
	store := testutil.NewTestStore()
	var createCalled bool
	store.CreateFileFn = func(fileID, shareID, agentID, filename, contentType string, visibility model.Visibility, ttlSeconds int, downloadToken sql.NullString, expiresAt time.Time) error {
		createCalled = true
		return nil
	}

	srv := newTestServer(store)
	ctx := ctxWithSession("sess_upload")

	out, err := srv.HandlePrepareUpload(ctx, PrepareUploadInput{
		Filename:    "test.txt",
		ContentType: "text/plain",
		Visibility:  "public",
		TTL:         "24h",
	})
	if err != nil {
		t.Fatalf("HandlePrepareUpload: %v", err)
	}
	if !createCalled {
		t.Error("expected CreateFile to be called")
	}
	if out.FileID == "" {
		t.Error("expected non-empty FileID")
	}
	if out.UploadURL == "" {
		t.Error("expected non-empty UploadURL")
	}
	if out.MaxSizeBytes != testutil.NewTestConfig().MaxFileSize {
		t.Errorf("MaxSizeBytes = %d, want %d", out.MaxSizeBytes, testutil.NewTestConfig().MaxFileSize)
	}
}

func TestServer_HandlePrepareUpload_NotAuthenticated(t *testing.T) {
	store := testutil.NewTestStore()
	srv := newTestServer(store)

	_, err := srv.HandlePrepareUpload(context.Background(), PrepareUploadInput{
		Filename:    "test.txt",
		ContentType: "text/plain",
	})
	if err == nil {
		t.Fatal("expected error for no auth")
	}
}

func TestServer_HandlePrepareUpload_EmptyFilename(t *testing.T) {
	store := testutil.NewTestStore()
	srv := newTestServer(store)
	ctx := ctxWithSession("sess_upload")

	_, err := srv.HandlePrepareUpload(ctx, PrepareUploadInput{
		Filename:    "",
		ContentType: "text/plain",
	})
	if err == nil {
		t.Fatal("expected error for empty filename")
	}
}

func TestServer_HandlePrepareUpload_EmptyContentType(t *testing.T) {
	store := testutil.NewTestStore()
	srv := newTestServer(store)
	ctx := ctxWithSession("sess_upload")

	_, err := srv.HandlePrepareUpload(ctx, PrepareUploadInput{
		Filename:    "test.txt",
		ContentType: "",
	})
	if err == nil {
		t.Fatal("expected error for empty content_type")
	}
}

func TestServer_HandlePrepareUpload_InvalidVisibility(t *testing.T) {
	store := testutil.NewTestStore()
	srv := newTestServer(store)
	ctx := ctxWithSession("sess_upload")

	_, err := srv.HandlePrepareUpload(ctx, PrepareUploadInput{
		Filename:    "test.txt",
		ContentType: "text/plain",
		Visibility:  "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid visibility")
	}
}

func TestServer_HandlePrepareUpload_InvalidTTL(t *testing.T) {
	store := testutil.NewTestStore()
	srv := newTestServer(store)
	ctx := ctxWithSession("sess_upload")

	_, err := srv.HandlePrepareUpload(ctx, PrepareUploadInput{
		Filename:    "test.txt",
		ContentType: "text/plain",
		TTL:         "notaduration",
	})
	if err == nil {
		t.Fatal("expected error for invalid TTL")
	}
}

func TestServer_HandlePrepareUpload_TokenVisibility(t *testing.T) {
	store := testutil.NewTestStore()
	var gotToken sql.NullString
	store.CreateFileFn = func(fileID, shareID, agentID, filename, contentType string, visibility model.Visibility, ttlSeconds int, downloadToken sql.NullString, expiresAt time.Time) error {
		gotToken = downloadToken
		return nil
	}

	srv := newTestServer(store)
	ctx := ctxWithSession("sess_upload")

	_, err := srv.HandlePrepareUpload(ctx, PrepareUploadInput{
		Filename:    "secret.txt",
		ContentType: "text/plain",
		Visibility:  "token",
	})
	if err != nil {
		t.Fatalf("HandlePrepareUpload token: %v", err)
	}
	if !gotToken.Valid || gotToken.String == "" {
		t.Error("expected non-empty download token for token visibility")
	}
}

func TestServer_HandlePrepareUpload_StoreError(t *testing.T) {
	store := testutil.NewTestStore()
	store.CreateFileFn = func(fileID, shareID, agentID, filename, contentType string, visibility model.Visibility, ttlSeconds int, downloadToken sql.NullString, expiresAt time.Time) error {
		return errors.New("db error")
	}

	srv := newTestServer(store)
	ctx := ctxWithSession("sess_upload")

	_, err := srv.HandlePrepareUpload(ctx, PrepareUploadInput{
		Filename:    "test.txt",
		ContentType: "text/plain",
	})
	if err == nil {
		t.Fatal("expected error for store failure")
	}
}

type mockAgentStore struct {
	getAgentBySessionFn func(sessionID string) (*filestore.Agent, error)
	registerAgentFn     func(agentID, tokenHash, tokenLookup, sessionID string) error
	verifyAnyTokenFn    func(token string) (*filestore.Agent, error)
}

func (m *mockAgentStore) GetAgentBySession(sessionID string) (*filestore.Agent, error) {
	if m.getAgentBySessionFn != nil {
		return m.getAgentBySessionFn(sessionID)
	}
	return nil, nil
}

func (m *mockAgentStore) RegisterAgent(agentID, tokenHash, tokenLookup, sessionID string) error {
	if m.registerAgentFn != nil {
		return m.registerAgentFn(agentID, tokenHash, tokenLookup, sessionID)
	}
	return nil
}

func (m *mockAgentStore) VerifyAnyToken(token string) (*filestore.Agent, error) {
	if m.verifyAnyTokenFn != nil {
		return m.verifyAnyTokenFn(token)
	}
	return nil, nil
}
