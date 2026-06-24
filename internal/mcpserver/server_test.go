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
		"plane_project_list",
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

func TestIssueListFilterSchemaAcceptsArraysAndStrings(t *testing.T) {
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

	for _, tool := range list.Tools {
		if tool.Name == "plane_issue_list" {
			requireArrayOrStringFilterSchema(t, tool.InputSchema.Properties, "assignees")
			requireArrayOrStringFilterSchema(t, tool.InputSchema.Properties, "labels")
			compact, ok := tool.InputSchema.Properties["compact"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, "boolean", compact["type"])
			return
		}
	}
	require.Fail(t, "plane_issue_list should be registered")
}

func TestIssueListAcceptsArrayAndStringFilters(t *testing.T) {
	for _, tt := range []struct {
		name string
		args map[string]any
	}{
		{
			name: "arrays",
			args: map[string]any{
				"project_id": "P",
				"assignees":  []any{"u1", "u2"},
				"labels":     []any{"l1", "l2"},
			},
		},
		{
			name: "strings",
			args: map[string]any{
				"project_id": "P",
				"assignees":  "u1,u2",
				"labels":     "l1,l2",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/v1/workspaces/ws/projects/P/issues/", r.URL.Path)
				require.Equal(t, "u1,u2", r.URL.Query().Get("assignees"))
				require.Equal(t, "l1,l2", r.URL.Query().Get("labels"))
				_, _ = w.Write([]byte(`{"results":[],"count":0}`))
			})

			payload := callToolPayload(t, srv, "plane_issue_list", tt.args)
			require.Equal(t, float64(0), payload["count"])
		})
	}
}

func TestIssueListCompactKeepsPaginationAndSummarizesIssues(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/workspaces/ws/projects/P/issues/", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"count": 3,
			"next_cursor": "cursor-2",
			"results": [
				{
					"id": "i1",
					"identifier": "TOOLS-1",
					"sequence_id": 1,
					"name": "Compact me",
					"description_html": "<p>large</p>",
					"priority": "high",
					"state": {"id": "s1", "name": "In Progress", "group": "started", "color": "#00f", "description": "hidden"},
					"labels": [{"id": "l1", "name": "Bug", "color": "#f00", "created_at": "hidden"}],
					"assignees": [{"id": "u1", "display_name": "Ada", "email": "ada@example.com"}]
				}
			]
		}`))
	})

	payload := callToolPayload(t, srv, "plane_issue_list", map[string]any{
		"project_id": "P",
		"compact":    true,
	})

	require.Equal(t, float64(3), payload["count"])
	require.Equal(t, "cursor-2", payload["next_cursor"])
	results, ok := payload["results"].([]any)
	require.True(t, ok)
	require.Len(t, results, 1)
	issue, ok := results[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "i1", issue["id"])
	require.Equal(t, "TOOLS-1", issue["identifier"])
	require.Equal(t, float64(1), issue["sequence_id"])
	require.Equal(t, "Compact me", issue["name"])
	require.Equal(t, "high", issue["priority"])
	require.NotContains(t, issue, "description_html")

	state, ok := issue["state"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "s1", state["id"])
	require.Equal(t, "In Progress", state["name"])
	require.Equal(t, "started", state["group"])
	require.Equal(t, "#00f", state["color"])
	require.NotContains(t, state, "description")

	labels, ok := issue["labels"].([]any)
	require.True(t, ok)
	require.Len(t, labels, 1)
	label, ok := labels[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "l1", label["id"])
	require.Equal(t, "Bug", label["name"])
	require.NotContains(t, label, "created_at")

	assignees, ok := issue["assignees"].([]any)
	require.True(t, ok)
	require.Len(t, assignees, 1)
	assignee, ok := assignees[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "u1", assignee["id"])
	require.Equal(t, "Ada", assignee["display_name"])
	require.NotContains(t, assignee, "email")
}

func TestIssueListFullResponseRemainsDefault(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/workspaces/ws/projects/P/issues/", r.URL.Path)
		_, _ = w.Write([]byte(`{"results":[{"id":"i1","description_html":"<p>large</p>"}],"count":1}`))
	})

	payload := callToolPayload(t, srv, "plane_issue_list", map[string]any{"project_id": "P"})
	results, ok := payload["results"].([]any)
	require.True(t, ok)
	require.Len(t, results, 1)
	issue, ok := results[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "<p>large</p>", issue["description_html"])
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

func requireArrayOrStringFilterSchema(t *testing.T, props map[string]any, name string) {
	t.Helper()
	prop, ok := props[name].(map[string]any)
	require.True(t, ok)
	oneOf, ok := prop["oneOf"].([]any)
	require.True(t, ok)
	require.Len(t, oneOf, 2)
	arraySchema, ok := oneOf[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "array", arraySchema["type"])
	items, ok := arraySchema["items"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "string", items["type"])
	stringSchema, ok := oneOf[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "string", stringSchema["type"])
}

func TestProjectListReturnsCompactResults(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/workspaces/ws/projects/", r.URL.Path)
		_, _ = w.Write([]byte(`{"results":[{"id":"p1","identifier":"TOOLS","name":"Tools","description":"hidden"},{"id":"p2","identifier":"STAFF","name":"Staff","extra":true}]}`))
	})

	payload := callToolPayload(t, srv, "plane_project_list", nil)
	results, ok := payload["results"].([]any)
	require.True(t, ok)
	require.Len(t, results, 2)
	first, ok := results[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, map[string]any{
		"id":         "p1",
		"identifier": "TOOLS",
		"name":       "Tools",
	}, first)
	require.NotContains(t, first, "description")
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

func TestAttachmentInitUploadSchemaWarnsAboutRawCredentials(t *testing.T) {
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

	for _, tool := range list.Tools {
		if tool.Name == "plane_attachment_init_upload" {
			require.Contains(t, tool.Description, "Prefer plane_attachment_upload")
			require.Contains(t, tool.Description, "temporary S3 form credentials")
			raw, ok := tool.InputSchema.Properties["raw"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, "boolean", raw["type"])
			require.Contains(t, raw["description"], "temporary S3 upload form credentials")
			return
		}
	}
	require.Fail(t, "plane_attachment_init_upload should be registered")
}

func TestAttachmentInitUploadDefaultsToCompactResult(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/workspaces/ws/projects/P/issues/I/issue-attachments/", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"id": "asset-1",
			"asset_url": "https://cdn.example.com/file.txt",
			"name": "file.txt",
			"upload_data": {
				"url": "https://uploads.example.com/",
				"fields": {
					"key": "tmp/file.txt",
					"policy": "secret-policy",
					"x-amz-signature": "secret-signature"
				}
			}
		}`))
	})

	payload := callToolPayload(t, srv, "plane_attachment_init_upload", map[string]any{
		"project_id": "P",
		"issue_id":   "I",
		"name":       "file.txt",
	})
	require.Equal(t, "asset-1", payload["asset_id"])
	require.Equal(t, "https://cdn.example.com/file.txt", payload["asset_url"])
	require.Equal(t, "file.txt", payload["name"])
	require.Equal(t, true, payload["raw_upload_credentials_omitted"])
	require.Equal(t, true, payload["upload_credentials_available_with_raw"])
	require.NotContains(t, payload, "upload_data")
}

func TestAttachmentInitUploadRawReturnsFullPayload(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/workspaces/ws/projects/P/issues/I/issue-attachments/", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"id": "asset-1",
			"upload_data": {
				"url": "https://uploads.example.com/",
				"fields": {"policy": "secret-policy"}
			}
		}`))
	})

	payload := callToolPayload(t, srv, "plane_attachment_init_upload", map[string]any{
		"project_id": "P",
		"issue_id":   "I",
		"name":       "file.txt",
		"raw":        true,
	})
	require.Equal(t, "asset-1", payload["id"])
	uploadData, ok := payload["upload_data"].(map[string]any)
	require.True(t, ok)
	fields, ok := uploadData["fields"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "secret-policy", fields["policy"])
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
