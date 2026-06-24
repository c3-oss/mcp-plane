package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/c3-oss/mcp-plane/internal/plane"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *Server {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	c, err := plane.NewClient(plane.Config{
		BaseURL:   ts.URL,
		Workspace: "ws",
		APIToken:  "token",
	})
	require.NoError(t, err)
	return New(c)
}

func TestToolsListIncludesEveryRegisteredTool(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	c, err := client.NewInProcessClient(srv.MCPServer())
	require.NoError(t, err)
	require.NoError(t, c.Start(context.Background()))
	t.Cleanup(func() { _ = c.Close() })

	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{})
	require.NoError(t, err)
	list, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)

	names := map[string]bool{}
	for _, tool := range list.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{
		"plane_issue_create", "plane_issue_list", "plane_issue_get", "plane_issue_get_by_identifier",
		"plane_issue_update", "plane_issue_delete",
		"plane_state_list", "plane_label_list",
		"plane_comment_list", "plane_comment_add", "plane_comment_update", "plane_comment_delete",
		"plane_activity_list",
		"plane_attachment_list", "plane_attachment_init_upload", "plane_attachment_complete_upload",
		"plane_attachment_upload", "plane_attachment_read",
		"plane_issue_transfer", "plane_issue_workpad_upsert",
		"plane_workspace_info", "plane_health",
	} {
		require.True(t, names[want], "tool %q should be registered", want)
	}
}

func TestStateListRoundTripsThroughInProcessClient(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/workspaces/ws/projects/P/states/", r.URL.Path)
		_, _ = w.Write([]byte(`{"results":[{"id":"s","name":"Todo","group":"unstarted"}]}`))
	})
	c, err := client.NewInProcessClient(srv.MCPServer())
	require.NoError(t, err)
	require.NoError(t, c.Start(context.Background()))
	t.Cleanup(func() { _ = c.Close() })

	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{})
	require.NoError(t, err)

	req := mcp.CallToolRequest{}
	req.Params.Name = "plane_state_list"
	req.Params.Arguments = map[string]any{"project_id": "P"}
	res, err := c.CallTool(context.Background(), req)
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Len(t, res.Content, 1)
	text, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(text.Text), &payload))
	results, _ := payload["results"].([]any)
	require.Len(t, results, 1)
}

func TestNotFoundMapsToToolError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"detail":"missing"}`, http.StatusNotFound)
	})
	c, err := client.NewInProcessClient(srv.MCPServer())
	require.NoError(t, err)
	require.NoError(t, c.Start(context.Background()))
	t.Cleanup(func() { _ = c.Close() })
	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{})
	require.NoError(t, err)

	req := mcp.CallToolRequest{}
	req.Params.Name = "plane_issue_get"
	req.Params.Arguments = map[string]any{"project_id": "P", "issue_id": "I"}
	res, err := c.CallTool(context.Background(), req)
	require.NoError(t, err)
	require.True(t, res.IsError, "404 should produce an isError tool result")
	require.Len(t, res.Content, 1)
	text, _ := res.Content[0].(mcp.TextContent)
	require.True(t, strings.Contains(text.Text, "not found"))
}

func TestWorkspaceInfoDoesNotEchoToken(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	c, err := client.NewInProcessClient(srv.MCPServer())
	require.NoError(t, err)
	require.NoError(t, c.Start(context.Background()))
	t.Cleanup(func() { _ = c.Close() })
	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{})
	require.NoError(t, err)

	req := mcp.CallToolRequest{}
	req.Params.Name = "plane_workspace_info"
	res, err := c.CallTool(context.Background(), req)
	require.NoError(t, err)
	require.False(t, res.IsError)
	text, _ := res.Content[0].(mcp.TextContent)
	require.NotContains(t, text.Text, "token")
	require.NotContains(t, text.Text, "X-API-Key")
	require.Contains(t, text.Text, "ws")
}

func TestAttachmentMimeTypeStripsParameters(t *testing.T) {
	require.Equal(t, "text/plain", attachmentMimeType("notes.txt"))
	require.Equal(t, "application/octet-stream", attachmentMimeType("archive.unknownext"))
}
