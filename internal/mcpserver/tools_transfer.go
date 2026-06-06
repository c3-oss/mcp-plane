package mcpserver

import (
	"context"

	"github.com/c3-oss/mcp-plane/internal/transfer"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerTransferTool() {
	s.mcp.AddTool(mcp.NewTool("plane_issue_transfer",
		mcp.WithDescription(
			"Transfer an issue to another project. Labels and state are translated by name "+
				"(case-insensitive); unmatched labels are reported in skipped_labels. "+
				"A cross-link comment is added on both issues. "+
				"Optionally closes the source by moving it to a state in the \"completed\" group.",
		),
		mcp.WithString("source_project_id", mcp.Required()),
		mcp.WithString("source_issue_id", mcp.Required()),
		mcp.WithString("source_project_code", mcp.Required(), mcp.Description(`Project code used to build cross-link identifiers like "SRC-7".`)),
		mcp.WithString("target_project_id", mcp.Required()),
		mcp.WithString("target_project_code", mcp.Required()),
		mcp.WithBoolean("close_source", mcp.Description("Move source issue to a completed-group state after transfer.")),
	), s.handleTransfer)
}

func (s *Server) handleTransfer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceProjectID, err := req.RequireString("source_project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("source_project_id", err), nil
	}
	sourceIssueID, err := req.RequireString("source_issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("source_issue_id", err), nil
	}
	sourceProjectCode, err := req.RequireString("source_project_code")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("source_project_code", err), nil
	}
	targetProjectID, err := req.RequireString("target_project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("target_project_id", err), nil
	}
	targetProjectCode, err := req.RequireString("target_project_code")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("target_project_code", err), nil
	}
	closeSource := req.GetBool("close_source", false)

	out, err := transfer.Transfer(ctx, s.client, transfer.Params{
		SourceProjectID:   sourceProjectID,
		SourceIssueID:     sourceIssueID,
		SourceProjectCode: sourceProjectCode,
		TargetProjectID:   targetProjectID,
		TargetProjectCode: targetProjectCode,
		CloseSource:       closeSource,
	})
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(out)
}
