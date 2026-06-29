package upload

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/agent"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore/model"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/testutil"
)

type mockAgentStore struct {
	getAgentBySessionFn func(sessionID string) (*filestore.Agent, error)
	registerAgentFn     func(agentID, tokenHash, tokenLookup, sessionID string) error
	rotateAgentTokenFn  func(agentID, tokenHash, tokenLookup string) error
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

func (m *mockAgentStore) RotateAgentToken(agentID, tokenHash, tokenLookup string) error {
	if m.rotateAgentTokenFn != nil {
		return m.rotateAgentTokenFn(agentID, tokenHash, tokenLookup)
	}
	return nil
}

func (m *mockAgentStore) VerifyAnyToken(token string) (*filestore.Agent, error) {
	if m.verifyAnyTokenFn != nil {
		return m.verifyAnyTokenFn(token)
	}
	return nil, nil
}

func TestHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	store := testutil.NewTestStore()
	agents := agent.NewManager(&mockAgentStore{})
	h := NewHandler(store, agents, testutil.NewTestConfig(), testutil.NewTestLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/upload/f_1", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_ServeHTTP_MissingAuth(t *testing.T) {
	store := testutil.NewTestStore()
	agents := agent.NewManager(&mockAgentStore{})
	h := NewHandler(store, agents, testutil.NewTestConfig(), testutil.NewTestLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/upload/f_1", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandler_ServeHTTP_InvalidToken(t *testing.T) {
	store := testutil.NewTestStore()
	agentStore := &mockAgentStore{
		verifyAnyTokenFn: func(token string) (*filestore.Agent, error) {
			return nil, nil
		},
	}
	agents := agent.NewManager(agentStore)
	h := NewHandler(store, agents, testutil.NewTestConfig(), testutil.NewTestLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/upload/f_1", nil)
	r.Header.Set("Authorization", "Bearer badtoken")
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandler_ServeHTTP_MissingFileID(t *testing.T) {
	store := testutil.NewTestStore()
	agentStore := &mockAgentStore{
		verifyAnyTokenFn: func(token string) (*filestore.Agent, error) {
			return &filestore.Agent{AgentID: "agent_1", SessionID: "sess_1"}, nil
		},
	}
	agents := agent.NewManager(agentStore)
	h := NewHandler(store, agents, testutil.NewTestConfig(), testutil.NewTestLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/upload/", nil)
	r.Header.Set("Authorization", "Bearer token1")
	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_ServeHTTP_FileNotFound(t *testing.T) {
	store := testutil.NewTestStore()
	store.GetFileByIDFn = func(fileID string) (*filestore.FileMetadata, error) {
		return nil, nil
	}
	agentStore := &mockAgentStore{
		verifyAnyTokenFn: func(token string) (*filestore.Agent, error) {
			return &filestore.Agent{AgentID: "agent_1", SessionID: "sess_1"}, nil
		},
	}
	agents := agent.NewManager(agentStore)
	h := NewHandler(store, agents, testutil.NewTestConfig(), testutil.NewTestLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/upload/f_404", nil)
	r.Header.Set("Authorization", "Bearer token1")
	h.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_ServeHTTP_FileAlreadyUploaded(t *testing.T) {
	store := testutil.NewTestStore()
	store.GetFileByIDFn = func(fileID string) (*filestore.FileMetadata, error) {
		return testutil.ReadyFile("f_1", "s1", "agent_1", "test.txt", "text/plain", model.VisPublic), nil
	}
	agentStore := &mockAgentStore{
		verifyAnyTokenFn: func(token string) (*filestore.Agent, error) {
			return &filestore.Agent{AgentID: "agent_1", SessionID: "sess_1"}, nil
		},
	}
	agents := agent.NewManager(agentStore)
	h := NewHandler(store, agents, testutil.NewTestConfig(), testutil.NewTestLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/upload/f_1", nil)
	r.Header.Set("Authorization", "Bearer token1")
	h.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandler_ServeHTTP_WrongAgent(t *testing.T) {
	store := testutil.NewTestStore()
	store.GetFileByIDFn = func(fileID string) (*filestore.FileMetadata, error) {
		return testutil.PendingFile("f_1", "s1", "agent_other", "test.txt", "text/plain", model.VisPublic), nil
	}
	agentStore := &mockAgentStore{
		verifyAnyTokenFn: func(token string) (*filestore.Agent, error) {
			return &filestore.Agent{AgentID: "agent_1", SessionID: "sess_1"}, nil
		},
	}
	agents := agent.NewManager(agentStore)
	h := NewHandler(store, agents, testutil.NewTestConfig(), testutil.NewTestLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/upload/f_1", nil)
	r.Header.Set("Authorization", "Bearer token1")
	h.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandler_ServeHTTP_StoreLookupError(t *testing.T) {
	store := testutil.NewTestStore()
	store.GetFileByIDFn = func(fileID string) (*filestore.FileMetadata, error) {
		return nil, errors.New("db error")
	}
	agentStore := &mockAgentStore{
		verifyAnyTokenFn: func(token string) (*filestore.Agent, error) {
			return &filestore.Agent{AgentID: "agent_1", SessionID: "sess_1"}, nil
		},
	}
	agents := agent.NewManager(agentStore)
	h := NewHandler(store, agents, testutil.NewTestConfig(), testutil.NewTestLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/upload/f_1", nil)
	r.Header.Set("Authorization", "Bearer token1")
	h.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandler_ServeHTTP_TokenVerificationError(t *testing.T) {
	store := testutil.NewTestStore()
	agentStore := &mockAgentStore{
		verifyAnyTokenFn: func(token string) (*filestore.Agent, error) {
			return nil, errors.New("auth error")
		},
	}
	agents := agent.NewManager(agentStore)
	h := NewHandler(store, agents, testutil.NewTestConfig(), testutil.NewTestLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/upload/f_1", strings.NewReader("data"))
	r.Header.Set("Authorization", "Bearer token1")
	h.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
