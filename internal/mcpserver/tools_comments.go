package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerCommentTools() {
	s.mcp.AddTool(mcp.NewTool("plane_comment_list",
		mcp.WithDescription("List comments on an issue."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
	), s.handleCommentList)

	s.mcp.AddTool(mcp.NewTool("plane_comment_add",
		mcp.WithDescription("Add an HTML comment to an issue."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
		mcp.WithString("comment_html", mcp.Required()),
		mcp.WithString("access", mcp.Description(`Visibility: "INTERNAL" (default) or "EXTERNAL".`), mcp.DefaultString("INTERNAL")),
	), s.handleCommentAdd)

	s.mcp.AddTool(mcp.NewTool("plane_comment_update",
		mcp.WithDescription("Update an existing comment's HTML body."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
		mcp.WithString("comment_id", mcp.Required()),
		mcp.WithString("comment_html", mcp.Required()),
		mcp.WithString("access", mcp.DefaultString("INTERNAL")),
	), s.handleCommentUpdate)

	s.mcp.AddTool(mcp.NewTool("plane_comment_delete",
		mcp.WithDescription("Delete a comment from an issue."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
		mcp.WithString("comment_id", mcp.Required()),
	), s.handleCommentDelete)
}

func (s *Server) handleCommentList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	out, err := s.client.ListComments(ctx, projectID, issueID)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(map[string]any{"results": out})
}

func (s *Server) handleCommentAdd(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	commentHTML, err := req.RequireString("comment_html")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("comment_html", err), nil
	}
	access := req.GetString("access", "INTERNAL")
	out, err := s.client.AddComment(ctx, projectID, issueID, commentHTML, access)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(out)
}

func (s *Server) handleCommentUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	commentID, err := req.RequireString("comment_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("comment_id", err), nil
	}
	commentHTML, err := req.RequireString("comment_html")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("comment_html", err), nil
	}
	access := req.GetString("access", "INTERNAL")
	out, err := s.client.UpdateComment(ctx, projectID, issueID, commentID, commentHTML, access)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(out)
}

func (s *Server) handleCommentDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	commentID, err := req.RequireString("comment_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("comment_id", err), nil
	}
	if err := s.client.DeleteComment(ctx, projectID, issueID, commentID); err != nil {
		return toolError(err), nil
	}
	return asTextResult(map[string]any{"deleted": true, "comment_id": commentID})
}
