package agent

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
	"golang.org/x/crypto/bcrypt"
)

type Store interface {
	GetAgentBySession(sessionID string) (*filestore.Agent, error)
	RegisterAgent(agentID, tokenHash, tokenLookup, sessionID string) error
	VerifyAnyToken(token string) (*filestore.Agent, error)
}

type Manager struct {
	store Store
}

func NewManager(store Store) *Manager {
	return &Manager{store: store}
}

func (m *Manager) RegisterOrGet(sessionID string) (agentID, agentToken string, err error) {
	existing, err := m.store.GetAgentBySession(sessionID)
	if err != nil {
		return "", "", fmt.Errorf("lookup agent by session: %w", err)
	}
	if existing != nil {
		return existing.AgentID, "", nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	agentToken = hex.EncodeToString(tokenBytes)

	hash, err := bcrypt.GenerateFromPassword([]byte(agentToken), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("hash token: %w", err)
	}

	idBytes := make([]byte, 4)
	if _, err := rand.Read(idBytes); err != nil {
		return "", "", fmt.Errorf("generate id: %w", err)
	}
	agentID = "agent_" + hex.EncodeToString(idBytes)

	lookupHash := sha256.Sum256([]byte(agentToken))
	tokenLookup := hex.EncodeToString(lookupHash[:])

	if err := m.store.RegisterAgent(agentID, string(hash), tokenLookup, sessionID); err != nil {
		return "", "", fmt.Errorf("register agent: %w", err)
	}

	return agentID, agentToken, nil
}

func (m *Manager) VerifyToken(token string) (*filestore.Agent, error) {
	return m.store.VerifyAnyToken(token)
}

func (m *Manager) GetBySession(sessionID string) (*filestore.Agent, error) {
	return m.store.GetAgentBySession(sessionID)
}
