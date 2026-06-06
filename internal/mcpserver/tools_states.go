package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerStateTools() {
	s.mcp.AddTool(mcp.NewTool("plane_state_list",
		mcp.WithDescription("List workflow states for a project. Returns id, name, group, color."),
		mcp.WithString("project_id", mcp.Required()),
	), s.handleStateList)
}

func (s *Server) handleStateList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	states, err := s.client.ListStates(ctx, projectID)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(map[string]any{"results": states})
}
