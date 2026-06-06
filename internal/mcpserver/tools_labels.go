package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerLabelTools() {
	s.mcp.AddTool(mcp.NewTool("plane_label_list",
		mcp.WithDescription("List labels for a project."),
		mcp.WithString("project_id", mcp.Required()),
	), s.handleLabelList)
}

func (s *Server) handleLabelList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	labels, err := s.client.ListLabels(ctx, projectID)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(map[string]any{"results": labels})
}
