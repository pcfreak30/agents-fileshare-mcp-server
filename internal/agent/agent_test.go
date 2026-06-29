package agent

import (
	"errors"
	"testing"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
)

type mockStore struct {
	getAgentBySessionFn func(sessionID string) (*filestore.Agent, error)
	registerAgentFn     func(agentID, tokenHash, tokenLookup, sessionID string) error
	rotateAgentTokenFn  func(agentID, tokenHash, tokenLookup string) error
	verifyAnyTokenFn    func(token string) (*filestore.Agent, error)
}

func (m *mockStore) GetAgentBySession(sessionID string) (*filestore.Agent, error) {
	if m.getAgentBySessionFn != nil {
		return m.getAgentBySessionFn(sessionID)
	}
	return nil, nil
}

func (m *mockStore) RegisterAgent(agentID, tokenHash, tokenLookup, sessionID string) error {
	if m.registerAgentFn != nil {
		return m.registerAgentFn(agentID, tokenHash, tokenLookup, sessionID)
	}
	return nil
}

func (m *mockStore) RotateAgentToken(agentID, tokenHash, tokenLookup string) error {
	if m.rotateAgentTokenFn != nil {
		return m.rotateAgentTokenFn(agentID, tokenHash, tokenLookup)
	}
	return nil
}

func (m *mockStore) VerifyAnyToken(token string) (*filestore.Agent, error) {
	if m.verifyAnyTokenFn != nil {
		return m.verifyAnyTokenFn(token)
	}
	return nil, nil
}

func TestManager_RegisterOrGet_NewAgent(t *testing.T) {
	var registeredID, registeredHash, registeredLookup, registeredSession string
	store := &mockStore{
		registerAgentFn: func(agentID, tokenHash, tokenLookup, sessionID string) error {
			registeredID = agentID
			registeredHash = tokenHash
			registeredLookup = tokenLookup
			registeredSession = sessionID
			return nil
		},
	}

	m := NewManager(store)
	agentID, agentToken, err := m.RegisterOrGet("sess_new")
	if err != nil {
		t.Fatalf("RegisterOrGet: %v", err)
	}

	if agentID == "" {
		t.Error("expected non-empty agentID")
	}
	if agentToken == "" {
		t.Error("expected non-empty agentToken for new agent")
	}
	if registeredID != agentID {
		t.Errorf("registered agentID = %q, want %q", registeredID, agentID)
	}
	if registeredSession != "sess_new" {
		t.Errorf("registered session = %q, want %q", registeredSession, "sess_new")
	}
	if registeredHash == "" {
		t.Error("expected non-empty token hash")
	}
	if registeredLookup == "" {
		t.Error("expected non-empty token lookup")
	}
}

func TestManager_RegisterOrGet_ExistingAgent(t *testing.T) {
	var rotatedID, rotatedHash, rotatedLookup string
	store := &mockStore{
		getAgentBySessionFn: func(sessionID string) (*filestore.Agent, error) {
			return &filestore.Agent{AgentID: "agent_existing", SessionID: sessionID}, nil
		},
		rotateAgentTokenFn: func(agentID, tokenHash, tokenLookup string) error {
			rotatedID = agentID
			rotatedHash = tokenHash
			rotatedLookup = tokenLookup
			return nil
		},
	}

	m := NewManager(store)
	agentID, agentToken, err := m.RegisterOrGet("sess_existing")
	if err != nil {
		t.Fatalf("RegisterOrGet: %v", err)
	}

	if agentID != "agent_existing" {
		t.Errorf("agentID = %q, want %q", agentID, "agent_existing")
	}
	if agentToken == "" {
		t.Error("expected non-empty agentToken for existing agent (rotated)")
	}
	if rotatedID != "agent_existing" {
		t.Errorf("rotated agentID = %q, want %q", rotatedID, "agent_existing")
	}
	if rotatedHash == "" {
		t.Error("expected non-empty rotated token hash")
	}
	if rotatedLookup == "" {
		t.Error("expected non-empty rotated token lookup")
	}
}

func TestManager_RegisterOrGet_StoreLookupError(t *testing.T) {
	store := &mockStore{
		getAgentBySessionFn: func(sessionID string) (*filestore.Agent, error) {
			return nil, errors.New("db error")
		},
	}

	m := NewManager(store)
	_, _, err := m.RegisterOrGet("sess_err")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestManager_RegisterOrGet_RegisterError(t *testing.T) {
	store := &mockStore{
		registerAgentFn: func(agentID, tokenHash, tokenLookup, sessionID string) error {
			return errors.New("insert error")
		},
	}

	m := NewManager(store)
	_, _, err := m.RegisterOrGet("sess_reg_err")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestManager_RegisterOrGet_RotateError(t *testing.T) {
	store := &mockStore{
		getAgentBySessionFn: func(sessionID string) (*filestore.Agent, error) {
			return &filestore.Agent{AgentID: "agent_rot_err", SessionID: sessionID}, nil
		},
		rotateAgentTokenFn: func(agentID, tokenHash, tokenLookup string) error {
			return errors.New("rotate error")
		},
	}

	m := NewManager(store)
	_, _, err := m.RegisterOrGet("sess_rot_err")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestManager_VerifyToken(t *testing.T) {
	expectedAgent := &filestore.Agent{AgentID: "agent_v1", SessionID: "sess_v1"}
	store := &mockStore{
		verifyAnyTokenFn: func(token string) (*filestore.Agent, error) {
			if token == "valid_token" {
				return expectedAgent, nil
			}
			return nil, nil
		},
	}

	m := NewManager(store)

	a, err := m.VerifyToken("valid_token")
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if a == nil || a.AgentID != "agent_v1" {
		t.Errorf("expected agent_v1, got %+v", a)
	}

	a, err = m.VerifyToken("invalid")
	if err != nil {
		t.Fatalf("VerifyToken invalid: %v", err)
	}
	if a != nil {
		t.Errorf("expected nil for invalid token, got %+v", a)
	}
}

func TestManager_GetBySession(t *testing.T) {
	expectedAgent := &filestore.Agent{AgentID: "agent_gs", SessionID: "sess_gs"}
	store := &mockStore{
		getAgentBySessionFn: func(sessionID string) (*filestore.Agent, error) {
			if sessionID == "sess_gs" {
				return expectedAgent, nil
			}
			return nil, nil
		},
	}

	m := NewManager(store)

	a, err := m.GetBySession("sess_gs")
	if err != nil {
		t.Fatalf("GetBySession: %v", err)
	}
	if a == nil || a.AgentID != "agent_gs" {
		t.Errorf("expected agent_gs, got %+v", a)
	}

	a, err = m.GetBySession("nonexistent")
	if err != nil {
		t.Fatalf("GetBySession nonexistent: %v", err)
	}
	if a != nil {
		t.Errorf("expected nil, got %+v", a)
	}
}
