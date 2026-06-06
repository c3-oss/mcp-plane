package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerWorkspaceTools() {
	s.mcp.AddTool(mcp.NewTool("plane_workspace_info",
		mcp.WithDescription("Return the configured Plane base URL and workspace slug. The API token is never echoed."),
	), s.handleWorkspaceInfo)

	s.mcp.AddTool(mcp.NewTool("plane_health",
		mcp.WithDescription("Round-trip a cheap GET against the configured workspace using the given project to verify credentials."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("Any project UUID the configured token can read.")),
	), s.handleHealth)
}

func (s *Server) handleWorkspaceInfo(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return asTextResult(map[string]any{
		"base_url":  s.client.BaseURL(),
		"workspace": s.client.Workspace(),
	})
}

func (s *Server) handleHealth(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	if _, err := s.client.ListStates(ctx, projectID); err != nil {
		return toolError(err), nil
	}
	return asTextResult(map[string]any{"ok": true})
}
