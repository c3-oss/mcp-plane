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

func callToolPayload(t *testing.T, srv *Server, name string, args map[string]any) map[string]any {
	t.Helper()
	c, err := client.NewInProcessClient(srv.MCPServer())
	require.NoError(t, err)
	require.NoError(t, c.Start(context.Background()))
	t.Cleanup(func() { _ = c.Close() })
	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{})
	require.NoError(t, err)

	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	res, err := c.CallTool(context.Background(), req)
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Len(t, res.Content, 1)
	text, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(text.Text), &payload))
	return payload
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

func TestHealthReportsEndpointDiagnostics(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/workspaces/ws/projects/P/states/":
			_, _ = w.Write([]byte(`{"results":[{"id":"s","name":"Todo"}]}`))
		case "/api/v1/workspaces/ws/projects/P/issues/":
			require.Equal(t, "1", r.URL.Query().Get("per_page"))
			_, _ = w.Write([]byte(`{"results":[{"id":"i","name":"Issue"}],"count":3}`))
		default:
			http.NotFound(w, r)
		}
	})

	payload := callToolPayload(t, srv, "plane_health", map[string]any{"project_id": "P"})
	require.Equal(t, true, payload["ok"])
	checks, ok := payload["checks"].(map[string]any)
	require.True(t, ok)
	states, ok := checks["states_endpoint"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, states["ok"])
	require.Equal(t, float64(1), states["count"])
	issues, ok := checks["issues_endpoint"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, issues["ok"])
	require.Equal(t, float64(1), issues["result_count"])
}

func TestHealthFlagsSuspiciousEmptyProject(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/workspaces/ws/projects/P/states/":
			_, _ = w.Write([]byte(`{"results":[]}`))
		case "/api/v1/workspaces/ws/projects/P/issues/":
			_, _ = w.Write([]byte(`{"results":[],"count":0}`))
		default:
			http.NotFound(w, r)
		}
	})

	payload := callToolPayload(t, srv, "plane_health", map[string]any{"project_id": "P"})
	require.Equal(t, false, payload["ok"])
	require.Contains(t, payload["warnings"], "states endpoint returned no states")
	require.Contains(t, payload["warnings"], "issues endpoint returned no issues for per_page=1 probe")
}

func TestHealthReturnsStructuredEndpointErrors(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/workspaces/ws/projects/P/states/":
			_, _ = w.Write([]byte(`{"results":[{"id":"s","name":"Todo"}]}`))
		case "/api/v1/workspaces/ws/projects/P/issues/":
			http.Error(w, `{"detail":"issues unavailable"}`, http.StatusForbidden)
		default:
			http.NotFound(w, r)
		}
	})

	payload := callToolPayload(t, srv, "plane_health", map[string]any{"project_id": "P"})
	require.Equal(t, false, payload["ok"])
	checks, ok := payload["checks"].(map[string]any)
	require.True(t, ok)
	issues, ok := checks["issues_endpoint"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, false, issues["ok"])
	require.Contains(t, issues["error"], "403")
}

func TestAttachmentMimeTypeStripsParameters(t *testing.T) {
	require.Equal(t, "text/plain", attachmentMimeType("notes.txt"))
	require.Equal(t, "application/octet-stream", attachmentMimeType("archive.unknownext"))
}

func TestAttachmentURLFallsBackForRelativeAssetKey(t *testing.T) {
	fallback := "/api/v1/workspaces/ws/projects/P/issues/I/issue-attachments/A"
	got := attachmentURL(plane.Attachment{
		"id":    "A",
		"asset": "uploads/metadata-probe.txt",
		"attributes": map[string]any{
			"name": "metadata-probe.txt",
			"size": 11,
			"type": "text/plain",
		},
	}, fallback)
	require.Equal(t, fallback, got)
}

func TestAttachmentURLUsesDownloadableCandidates(t *testing.T) {
	fallback := "/fallback"
	require.Equal(t, "https://cdn.example.com/file.txt", attachmentURL(plane.Attachment{
		"asset": "https://cdn.example.com/file.txt",
	}, fallback))
	require.Equal(t, "/api/assets/file.txt", attachmentURL(plane.Attachment{
		"url": "/api/assets/file.txt",
	}, fallback))
	require.Equal(t, "https://attrs.example.com/file.txt", attachmentURL(plane.Attachment{
		"attributes": map[string]any{"url": "https://attrs.example.com/file.txt"},
		"asset":      "uploads/file.txt",
	}, fallback))
}
