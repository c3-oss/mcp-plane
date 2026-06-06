package mcpserver

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/c3-oss/mcp-plane/internal/plane"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerAttachmentTools() {
	s.mcp.AddTool(mcp.NewTool("plane_attachment_list",
		mcp.WithDescription("List attachments on an issue."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
	), s.handleAttachmentList)

	s.mcp.AddTool(mcp.NewTool("plane_attachment_init_upload",
		mcp.WithDescription("Low-level: request S3 upload credentials for a new attachment."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
		mcp.WithString("name", mcp.Required()),
		mcp.WithString("file_type"),
		mcp.WithNumber("size"),
	), s.handleAttachmentInitUpload)

	s.mcp.AddTool(mcp.NewTool("plane_attachment_complete_upload",
		mcp.WithDescription("Low-level: tell Plane the S3 upload has completed."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
		mcp.WithString("asset_id", mcp.Required()),
	), s.handleAttachmentCompleteUpload)

	s.mcp.AddTool(mcp.NewTool("plane_attachment_upload",
		mcp.WithDescription("High-level: upload a local file as an issue attachment (init → multipart POST → complete)."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to the file on the host running this MCP server.")),
		mcp.WithString("display_name", mcp.Description("Optional display name; defaults to the file basename.")),
	), s.handleAttachmentUpload)

	s.mcp.AddTool(mcp.NewTool("plane_attachment_read",
		mcp.WithDescription("Read an attachment. Text is returned inline; images are returned base64-encoded; other types return metadata only."),
		mcp.WithString("project_id", mcp.Required()),
		mcp.WithString("issue_id", mcp.Required()),
		mcp.WithString("attachment_id", mcp.Required()),
	), s.handleAttachmentRead)
}

func (s *Server) handleAttachmentList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	out, err := s.client.ListAttachments(ctx, projectID, issueID)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(map[string]any{"results": out})
}

func (s *Server) handleAttachmentInitUpload(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("name", err), nil
	}
	args := req.GetArguments()
	fileType := argStringPtr(args, "file_type")
	size := argInt64Ptr(args, "size")
	out, err := s.client.InitAttachmentUpload(ctx, projectID, issueID, name, fileType, size)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(out)
}

func (s *Server) handleAttachmentCompleteUpload(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	assetID, err := req.RequireString("asset_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("asset_id", err), nil
	}
	out, err := s.client.CompleteAttachmentUpload(ctx, projectID, issueID, assetID)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(out)
}

func (s *Server) handleAttachmentUpload(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	path, err := req.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("file_path", err), nil
	}
	displayName := req.GetString("display_name", "")
	if displayName == "" {
		displayName = filepath.Base(path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return mcp.NewToolResultErrorf("file_path: %v", err), nil
	}
	size := info.Size()

	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	init, err := s.client.InitAttachmentUpload(ctx, projectID, issueID, displayName, &mimeType, &size)
	if err != nil {
		return toolError(err), nil
	}
	assetID, uploadURL, uploadFields, err := parseInitPayload(init)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("init payload", err), nil
	}

	f, err := os.Open(path) //nolint:gosec // path is supplied by an MCP caller with file-system trust.
	if err != nil {
		return mcp.NewToolResultErrorf("open file: %v", err), nil
	}
	defer f.Close()

	if err := s.client.UploadAttachmentBlob(ctx, uploadURL, uploadFields, displayName, f, mimeType); err != nil {
		return toolError(err), nil
	}
	result, err := s.client.CompleteAttachmentUpload(ctx, projectID, issueID, assetID)
	if err != nil {
		return toolError(err), nil
	}
	return asTextResult(map[string]any{
		"asset_id": assetID,
		"name":     displayName,
		"size":     size,
		"result":   result,
	})
}

func (s *Server) handleAttachmentRead(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("project_id", err), nil
	}
	issueID, err := req.RequireString("issue_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("issue_id", err), nil
	}
	attachmentID, err := req.RequireString("attachment_id")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("attachment_id", err), nil
	}

	attachments, err := s.client.ListAttachments(ctx, projectID, issueID)
	if err != nil {
		return toolError(err), nil
	}
	var found plane.Attachment
	for _, a := range attachments {
		if id, _ := a["id"].(string); id == attachmentID {
			found = a
			break
		}
	}
	if found == nil {
		return mcp.NewToolResultErrorf("attachment %q not found on issue %q", attachmentID, issueID), nil
	}

	name := plane.AttachmentName(found)
	kind := plane.AttachmentKind(name)
	out := map[string]any{
		"id":   attachmentID,
		"name": name,
		"kind": kind,
		"size": plane.AttachmentSize(found),
	}
	if kind == "other" {
		return asTextResult(out)
	}
	url := attachmentURL(found, s.client.AttachmentDetailURL(projectID, issueID, attachmentID))
	body, contentType, err := s.client.DownloadAttachment(ctx, url)
	if err != nil {
		return toolError(err), nil
	}
	if contentType != "" {
		out["mime_type"] = contentType
	}
	switch kind {
	case "text":
		out["content"] = makeUTF8(body)
	case "image":
		out["data_base64"] = base64.StdEncoding.EncodeToString(body)
	}
	return asTextResult(out)
}

func parseInitPayload(init map[string]any) (assetID, uploadURL string, fields map[string]string, err error) {
	if v, ok := init["id"].(string); ok && v != "" {
		assetID = v
	} else if v, ok := init["asset_id"].(string); ok && v != "" {
		assetID = v
	} else if att, ok := init["attachment"].(map[string]any); ok {
		if v, ok := att["id"].(string); ok && v != "" {
			assetID = v
		}
	}
	if assetID == "" {
		return "", "", nil, fmt.Errorf("missing asset id")
	}
	uploadData, ok := init["upload_data"].(map[string]any)
	if !ok {
		return "", "", nil, fmt.Errorf("missing upload_data")
	}
	uploadURL, _ = uploadData["url"].(string)
	if uploadURL == "" {
		return "", "", nil, fmt.Errorf("missing upload_data.url")
	}
	fields = map[string]string{}
	if raw, ok := uploadData["fields"].(map[string]any); ok {
		for k, v := range raw {
			if s, ok := v.(string); ok {
				fields[k] = s
			}
		}
	}
	return assetID, uploadURL, fields, nil
}

func attachmentURL(att plane.Attachment, fallback string) string {
	if attrs, ok := att["attributes"].(map[string]any); ok {
		if u, ok := attrs["url"].(string); ok && u != "" {
			return u
		}
	}
	if u, ok := att["asset"].(string); ok && u != "" {
		return u
	}
	if u, ok := att["url"].(string); ok && u != "" {
		return u
	}
	return fallback
}

func makeUTF8(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	// Replace invalid sequences with U+FFFD, matching the Python errors="replace" behaviour.
	out := make([]rune, 0, len(b))
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if r == utf8.RuneError && size == 1 {
			out = append(out, '�')
			i++
			continue
		}
		out = append(out, r)
		i += size
	}
	return string(out)
}
