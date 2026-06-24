package mcpserver

import (
	"context"

	"github.com/c3-oss/mcp-plane/internal/plane"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerIssueTools() {
	s.mcp.AddTool(mcp.NewTool("plane_issue_create",
		mcp.WithDescription("Create an issue in a Plane project."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("Plane project UUID")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Issue title")),
		mcp.WithString("description_html", mcp.Description("HTML body")),
		mcp.WithArray("assignees", mcp.Description("Assignee Plane user UUIDs"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithString("priority", mcp.Description(`One of "urgent", "high", "medium", "low", "none".`)),
		mcp.WithString("state", mcp.Description("State UUID (use plane_state_list to discover).")),
		mcp.WithArray("labels", mcp.Description("Label UUIDs"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithString("parent", mcp.Description("Parent issue UUID")),
	), s.handleIssueCreate)

	s.mcp.AddTool(mcp.NewTool("plane_issue_list",
		mcp.WithDescription("List issues in a project with optional cursor pagination and filters."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithNumber("per_page", mcp.Description("Page size; server default applies if omitted.")),
		mcp.WithString("cursor"),
		mcp.WithBoolean("archived"),
		mcp.WithString("state_group", mcp.Description(`Plane state group: "backlog", "unstarted", "started", "completed", "cancelled".`)),
		mcp.WithAny("assignees", issueListIDsFilter("Assignee Plane user UUIDs. Use an array; comma-separated string is accepted.")),
		mcp.WithAny("labels", issueListIDsFilter("Label UUIDs. Use an array; comma-separated string is accepted.")),
		mcp.WithString("priority"),
		mcp.WithString("created_at", mcp.Description(`Range string, e.g. "2026-04-01;after,2026-04-30;before".`)),
		mcp.WithString("target_date"),
		mcp.WithString("completed_at"),
		mcp.WithString("name", mcp.Description("Substring search on issue name.")),
		mcp.WithString("order_by", mcp.Description(`e.g. "-created_at"`)),
		mcp.WithString("expand", mcp.Description(`Comma-separated fields to expand, e.g. "state,assignees".`)),
		mcp.WithString("parent"),
	), s.handleIssueList)

	s.mcp.AddTool(mcp.NewTool("plane_issue_get",
		mcp.WithDescription("Fetch a single issue by UUID."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
	), s.handleIssueGet)

	s.mcp.AddTool(mcp.NewTool("plane_issue_get_by_identifier",
		mcp.WithDescription(`Fetch an issue by its workspace identifier such as "PROJ-123".`),
		mcp.WithString("identifier", mcp.Required()),
	), s.handleIssueGetByIdentifier)

	s.mcp.AddTool(mcp.NewTool("plane_issue_update",
		mcp.WithDescription("Patch an issue. Only the fields supplied are sent to Plane."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
		mcp.WithString("name"),
		mcp.WithString("description_html"),
		mcp.WithArray("assignees", mcp.Items(map[string]any{"type": "string"})),
		mcp.WithString("priority"),
		mcp.WithString("state"),
		mcp.WithArray("labels", mcp.Items(map[string]any{"type": "string"})),
		mcp.WithString("parent"),
	), s.handleIssueUpdate)

	s.mcp.AddTool(mcp.NewTool("plane_issue_delete",
		mcp.WithDescription("Delete an issue."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
	), s.handleIssueDelete)
}

func (s *Server) handleIssueCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("name", err), nil
	}
	args := req.GetArguments()
	opts := issueOptionsFromArgs(args)
	out, err := s.client.CreateIssue(ctx, projectID, name, opts)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(out)
}

func (s *Server) handleIssueList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	args := req.GetArguments()
	opts := plane.ListIssuesOptions{
		PerPage:     argIntPtr(args, "per_page"),
		Cursor:      argStringPtr(args, "cursor"),
		Archived:    argBoolPtr(args, "archived"),
		StateGroup:  argStringPtr(args, "state_group"),
		Assignees:   argStringOrStringSliceCSVPtr(args, "assignees"),
		Labels:      argStringOrStringSliceCSVPtr(args, "labels"),
		Priority:    argStringPtr(args, "priority"),
		CreatedAt:   argStringPtr(args, "created_at"),
		TargetDate:  argStringPtr(args, "target_date"),
		CompletedAt: argStringPtr(args, "completed_at"),
		Name:        argStringPtr(args, "name"),
		OrderBy:     argStringPtr(args, "order_by"),
		Expand:      argStringPtr(args, "expand"),
		Parent:      argStringPtr(args, "parent"),
	}
	out, err := s.client.ListIssues(ctx, projectID, opts)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(out)
}

func (s *Server) handleIssueGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	out, err := s.client.GetIssue(ctx, projectID, issueID)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(out)
}

func (s *Server) handleIssueGetByIdentifier(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	identifier, err := req.RequireString("identifier")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("identifier", err), nil
	}
	out, err := s.client.GetIssueByIdentifier(ctx, identifier)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(out)
}

func (s *Server) handleIssueUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	args := req.GetArguments()
	name := argStringPtr(args, "name")
	opts := issueOptionsFromArgs(args)
	out, err := s.client.UpdateIssue(ctx, projectID, issueID, name, opts)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(out)
}

func (s *Server) handleIssueDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	if err := s.client.DeleteIssue(ctx, projectID, issueID); err != nil {
		return toolError(err), nil
	}
	return asTextResult(map[string]any{"deleted": true, "issue_id": issueID})
}

func issueOptionsFromArgs(args map[string]any) plane.IssueOptions {
	return plane.IssueOptions{
		DescriptionHTML: argStringPtr(args, "description_html"),
		Assignees:       argStringSlicePtr(args, "assignees"),
		Priority:        argStringPtr(args, "priority"),
		State:           argStringPtr(args, "state"),
		Labels:          argStringSlicePtr(args, "labels"),
		Parent:          argStringPtr(args, "parent"),
	}
}

func issueListIDsFilter(description string) mcp.PropertyOption {
	return func(schema map[string]any) {
		schema["description"] = description
		schema["oneOf"] = []any{
			map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
			map[string]any{"type": "string"},
		}
	}
}
