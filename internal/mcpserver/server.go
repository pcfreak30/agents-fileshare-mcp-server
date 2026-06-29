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
		Name:        "whoami",
		Description: "Register or identify the current agent. Call this FIRST before any other tool. Returns an auth token on first call (new agents) that must be used as 'Authorization: Bearer <token>' in the PUT upload request. For existing agents the token is empty (already obtained previously).",
	}, ms.handleWhoami)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "prepare_upload",
		Description: "Create a pending upload slot and get the upload URL. Call whoami first to get your auth token. After preparing, PUT the file bytes to upload_url with 'Authorization: Bearer <token>' header. The file must be uploaded before the expires_at timestamp.",
	}, ms.handlePrepareUpload)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_files",
		Description: "List files, optionally filtered by agent or status. Returns files with metadata and a total count.",
	}, ms.handleListFiles)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_file_info",
		Description: "Get full metadata for a specific file by file_id, including download URL and download token (if token visibility).",
	}, ms.handleGetFileInfo)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_file_link",
		Description: "Get a shareable download link for a file. For token-visibility files, includes a download token that must be passed as a query parameter.",
	}, ms.handleGetFileLink)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_file",
		Description: "Delete a file. Only the uploading agent can delete their own files.",
	}, ms.handleDeleteFile)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_files",
		Description: "Search files by filename substring using FTS5 full-text search.",
	}, ms.handleSearchFiles)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_agents",
		Description: "List all agents that have uploaded files, with file counts and total sizes.",
	}, ms.handleListAgents)
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
