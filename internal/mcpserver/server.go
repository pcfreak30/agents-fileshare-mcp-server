package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/agent"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/config"
	"github.com/pcfreak30/agents-fileshare-mcp-server/internal/filestore"
	"go.uber.org/zap"
)

type MCPServer struct {
	srv    *Server
	logger *zap.Logger
	agents *agent.Manager
}

func NewMCPServer(store filestore.FileStore, agents *agent.Manager, cfg *config.Config, log *zap.Logger) *MCPServer {
	srv := NewServer(store, agents, cfg, log)
	return &MCPServer{srv: srv, logger: log, agents: agents}
}

func (ms *MCPServer) RegisterTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "prepare_upload",
		Description: "Create a pending upload slot and get upload credentials",
	}, ms.handlePrepareUpload)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_files",
		Description: "List files, optionally filtered by agent or status",
	}, ms.handleListFiles)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_file_info",
		Description: "Get full metadata for a specific file",
	}, ms.handleGetFileInfo)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_file_link",
		Description: "Get a shareable download link for a file",
	}, ms.handleGetFileLink)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_file",
		Description: "Delete a file. Only the uploading agent can delete their own files",
	}, ms.handleDeleteFile)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_files",
		Description: "Search files by filename substring",
	}, ms.handleSearchFiles)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_agents",
		Description: "List all agents that have uploaded files",
	}, ms.handleListAgents)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "whoami",
		Description: "Get the current agent's identity and token",
	}, ms.handleWhoami)
}

func handleTool[I any, O any](ctx context.Context, req *mcp.CallToolRequest, in I, fn func(context.Context, I) (*O, error)) (*mcp.CallToolResult, O, error) {
	var zero O
	ctx = context.WithValue(ctx, sessionIDKey{}, req.Session.ID())
	out, err := fn(ctx, in)
	if err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, zero, nil
	}
	return &mcp.CallToolResult{}, *out, nil
}

func (ms *MCPServer) handlePrepareUpload(ctx context.Context, req *mcp.CallToolRequest, in PrepareUploadInput) (*mcp.CallToolResult, PrepareUploadOutput, error) {
	return handleTool(ctx, req, in, ms.srv.HandlePrepareUpload)
}

func (ms *MCPServer) handleListFiles(ctx context.Context, req *mcp.CallToolRequest, in ListFilesInput) (*mcp.CallToolResult, ListFilesOutput, error) {
	return handleTool(ctx, req, in, ms.srv.HandleListFiles)
}

func (ms *MCPServer) handleGetFileInfo(ctx context.Context, req *mcp.CallToolRequest, in GetFileInfoInput) (*mcp.CallToolResult, GetFileInfoOutput, error) {
	return handleTool(ctx, req, in, ms.srv.HandleGetFileInfo)
}

func (ms *MCPServer) handleGetFileLink(ctx context.Context, req *mcp.CallToolRequest, in GetFileLinkInput) (*mcp.CallToolResult, GetFileLinkOutput, error) {
	return handleTool(ctx, req, in, ms.srv.HandleGetFileLink)
}

func (ms *MCPServer) handleDeleteFile(ctx context.Context, req *mcp.CallToolRequest, in DeleteFileInput) (*mcp.CallToolResult, DeleteFileOutput, error) {
	return handleTool(ctx, req, in, ms.srv.HandleDeleteFile)
}

func (ms *MCPServer) handleSearchFiles(ctx context.Context, req *mcp.CallToolRequest, in SearchFilesInput) (*mcp.CallToolResult, SearchFilesOutput, error) {
	return handleTool(ctx, req, in, ms.srv.HandleSearchFiles)
}

func (ms *MCPServer) handleListAgents(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, ListAgentsOutput, error) {
	return handleTool(ctx, req, in, func(ctx context.Context, _ struct{}) (*ListAgentsOutput, error) {
		return ms.srv.HandleListAgents(ctx)
	})
}

func (ms *MCPServer) handleWhoami(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, WhoamiOutput, error) {
	return handleTool(ctx, req, in, func(ctx context.Context, _ struct{}) (*WhoamiOutput, error) {
		return ms.srv.HandleWhoami(ctx)
	})
}

type AgentRegistrator interface {
	RegisterOrGet(sessionID string) (agentID, agentToken string, err error)
}

func InitializedHandler(agents AgentRegistrator, log *zap.Logger) func(context.Context, *mcp.InitializedRequest) {
	return func(ctx context.Context, req *mcp.InitializedRequest) {
		sessionID := req.Session.ID()
		if sessionID == "" {
			log.Warn("initialized without session ID")
			return
		}

		agentID, agentToken, err := agents.RegisterOrGet(sessionID)
		if err != nil {
			log.Error("register agent on init", zap.Error(err), zap.String("session_id", sessionID))
			return
		}

		log.Info("agent initialized",
			zap.String("agent_id", agentID),
			zap.String("session_id", sessionID),
			zap.Bool("new_agent", agentToken != ""),
		)
	}
}
