package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerActivityTools() {
	s.mcp.AddTool(mcp.NewTool("plane_activity_list",
		mcp.WithDescription("List activity entries for an issue (read-only audit log)."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
	), s.handleActivityList)
}

func (s *Server) handleActivityList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	activities, err := s.client.ListActivities(ctx, projectID, issueID)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(map[string]any{"results": activities})
}
