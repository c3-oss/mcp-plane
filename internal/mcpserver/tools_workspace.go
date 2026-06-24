package mcpserver

import (
	"context"

	"github.com/c3-oss/mcp-plane/internal/plane"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerWorkspaceTools() {
	s.mcp.AddTool(mcp.NewTool("plane_workspace_info",
		mcp.WithDescription("Return the configured Plane base URL and workspace slug. The API token is never echoed."),
	), s.handleWorkspaceInfo)

	s.mcp.AddTool(mcp.NewTool("plane_health",
		mcp.WithDescription("Probe project access by reading the states and issues endpoints, returning structured diagnostics."),
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

	states, statesErr := s.client.ListStates(ctx, projectID)
	perPage := 1
	issues, issuesErr := s.client.ListIssues(ctx, projectID, plane.ListIssuesOptions{PerPage: &perPage})

	stateCheck := endpointCheck(statesErr)
	if statesErr == nil {
		stateCheck["count"] = len(states)
	}
	issueCheck := endpointCheck(issuesErr)
	if issuesErr == nil {
		issueCheck["result_count"] = issueResultCount(issues)
	}

	warnings := []string{}
	if statesErr == nil && len(states) == 0 {
		warnings = append(warnings, "states endpoint returned no states")
	}
	if issuesErr == nil && issueResultCount(issues) == 0 {
		warnings = append(warnings, "issues endpoint returned no issues for per_page=1 probe")
	}

	return asTextResult(map[string]any{
		"ok":         statesErr == nil && issuesErr == nil && len(states) > 0,
		"project_id": projectID,
		"checks": map[string]any{
			"states_endpoint": stateCheck,
			"issues_endpoint": issueCheck,
		},
		"warnings": warnings,
	})
}

func endpointCheck(err error) map[string]any {
	out := map[string]any{"ok": err == nil}
	if err != nil {
		out["error"] = err.Error()
	}
	return out
}

func issueResultCount(issues plane.IssueList) int {
	results, _ := issues["results"].([]any)
	return len(results)
}
