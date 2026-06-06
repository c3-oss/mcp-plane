package mcpserver

import (
	"context"

	"github.com/c3-oss/mcp-plane/internal/plane"
	"github.com/c3-oss/mcp-plane/internal/workpad"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerWorkpadTool() {
	s.mcp.AddTool(mcp.NewTool("plane_issue_workpad_upsert",
		mcp.WithDescription(
			"Render the workpad and upsert it as a comment on the issue. "+
				"Finds an existing workpad comment by marker text; updates it if present, otherwise adds a new one.",
		),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
		mcp.WithString("summary", mcp.Required(), mcp.Description("Workpad summary line.")),
		mcp.WithString("delegation_status"),
		mcp.WithString("repo_key"),
		mcp.WithString("repo_path"),
		mcp.WithString("publish_remote"),
		mcp.WithString("branch"),
		mcp.WithString("pr_url"),
		mcp.WithArray("validation", mcp.Items(map[string]any{"type": "string"})),
		mcp.WithArray("blockers", mcp.Items(map[string]any{"type": "string"})),
	), s.handleWorkpadUpsert)
}

func (s *Server) handleWorkpadUpsert(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	summary, err := req.RequireString("summary")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("summary", err), nil
	}
	args := req.GetArguments()

	issue, err := s.client.GetIssue(ctx, projectID, issueID)
	if err != nil {
		return toolError(err), nil
	}

	html := workpad.Render(workpad.RenderArgs{
		Issue:            issue,
		Summary:          summary,
		DelegationStatus: req.GetString("delegation_status", ""),
		RepoKey:          req.GetString("repo_key", ""),
		RepoPath:         req.GetString("repo_path", ""),
		PublishRemote:    req.GetString("publish_remote", ""),
		Branch:           req.GetString("branch", ""),
		PRURL:            req.GetString("pr_url", ""),
		Validation:       argStringSlice(args, "validation"),
		Blockers:         argStringSlice(args, "blockers"),
	})

	comment, err := workpad.UpsertComment(ctx, s.client, projectID, issueID, html, "")
	if err != nil {
		return toolError(err), nil
	}

	commentID, _ := comment["id"].(string)
	return asTextResult(map[string]any{
		"status":     "workpad_upserted",
		"comment_id": commentID,
		"issue":      plane.Issue(issue),
	})
}
