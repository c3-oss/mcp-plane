package plane

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// newTestClient returns a Client wired to ts plus the recorded requests.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	c, err := NewClient(Config{
		BaseURL:   ts.URL,
		Workspace: "ws",
		APIToken:  "token",
	})
	require.NoError(t, err)
	return c, ts
}

func ptrStr(s string) *string { return &s }
func ptrInt(n int) *int       { return &n }
func ptrBool(b bool) *bool    { return &b }
func ptrSlice(s []string) *[]string {
	return &s
}

func TestNewClientRejectsMissingFields(t *testing.T) {
	_, err := NewClient(Config{Workspace: "ws", APIToken: "t"})
	require.Error(t, err)
	_, err = NewClient(Config{BaseURL: "https://x", Workspace: "ws"})
	require.Error(t, err)
	_, err = NewClient(Config{BaseURL: "https://x", APIToken: "t"})
	require.Error(t, err)
}

func TestNewClientTrimsBaseURL(t *testing.T) {
	c, err := NewClient(Config{BaseURL: "https://x/  ", Workspace: "ws", APIToken: "t"})
	require.NoError(t, err)
	require.Equal(t, "https://x", c.BaseURL())
}

func TestRequestSendsAuthHeaders(t *testing.T) {
	var seen *http.Request
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		seen = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	_, err := c.GetIssue(context.Background(), "P", "I")
	require.NoError(t, err)
	require.NotNil(t, seen)
	require.Equal(t, "token", seen.Header.Get("X-API-Key"))
	require.Equal(t, "application/json", seen.Header.Get("Accept"))
	require.Equal(t, "mcp-plane", seen.Header.Get("User-Agent"))
}

func TestErrorMappingNotFoundAndUnauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/workspaces/ws/projects/P/issues/MISSING/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"detail":"not found"}`, http.StatusNotFound)
	})
	mux.HandleFunc("/api/v1/workspaces/ws/projects/P/issues/FORBIDDEN/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"detail":"nope"}`, http.StatusForbidden)
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	c, err := NewClient(Config{BaseURL: ts.URL, Workspace: "ws", APIToken: "t"})
	require.NoError(t, err)

	_, err = c.GetIssue(context.Background(), "P", "MISSING")
	require.Error(t, err)
	require.True(t, IsNotFound(err))
	require.False(t, IsUnauthorized(err))

	_, err = c.GetIssue(context.Background(), "P", "FORBIDDEN")
	require.Error(t, err)
	require.True(t, IsUnauthorized(err))
}

func TestCreateIssueOmitsUnsetOptions(t *testing.T) {
	var captured map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/workspaces/ws/projects/P/issues/", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &captured))
		_, _ = w.Write([]byte(`{"id":"new"}`))
	})
	out, err := c.CreateIssue(context.Background(), "P", "Hello", IssueOptions{})
	require.NoError(t, err)
	require.Equal(t, "new", out["id"])
	require.Equal(t, map[string]any{"name": "Hello"}, captured)
}

func TestCreateIssueSendsAllProvidedFields(t *testing.T) {
	var captured map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &captured))
		_, _ = w.Write([]byte(`{}`))
	})
	_, err := c.CreateIssue(context.Background(), "P", "Hello", IssueOptions{
		DescriptionHTML: ptrStr("<p>x</p>"),
		Assignees:       ptrSlice([]string{"u1", "u2"}),
		Priority:        ptrStr("high"),
		State:           ptrStr("state-uuid"),
		Labels:          ptrSlice([]string{"l1"}),
		Parent:          ptrStr("parent-uuid"),
	})
	require.NoError(t, err)
	require.Equal(t, "Hello", captured["name"])
	require.Equal(t, "<p>x</p>", captured["description_html"])
	require.Equal(t, []any{"u1", "u2"}, captured["assignees"])
	require.Equal(t, "high", captured["priority"])
	require.Equal(t, "state-uuid", captured["state"])
	require.Equal(t, []any{"l1"}, captured["labels"])
	require.Equal(t, "parent-uuid", captured["parent"])
}

func TestUpdateIssueAllowsEmptyPayload(t *testing.T) {
	var captured map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPatch, r.Method)
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &captured))
		_, _ = w.Write([]byte(`{}`))
	})
	_, err := c.UpdateIssue(context.Background(), "P", "I", nil, IssueOptions{})
	require.NoError(t, err)
	require.Equal(t, map[string]any{}, captured)
}

func TestListIssuesQueryParamsAndArchivedStringification(t *testing.T) {
	var seenURL string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		seenURL = r.URL.String()
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	_, err := c.ListIssues(context.Background(), "P", ListIssuesOptions{
		PerPage:    ptrInt(25),
		Cursor:     ptrStr("abc"),
		Archived:   ptrBool(false),
		StateGroup: ptrStr("started"),
		Name:       ptrStr("hello world"),
		OrderBy:    ptrStr("-created_at"),
	})
	require.NoError(t, err)
	require.Contains(t, seenURL, "per_page=25")
	require.Contains(t, seenURL, "cursor=abc")
	require.Contains(t, seenURL, "archived=false")
	require.Contains(t, seenURL, "state_group=started")
	require.Contains(t, seenURL, "name=hello+world")
	require.Contains(t, seenURL, "order_by=-created_at")
}

func TestListIssuesWrapsBareArrayResults(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"a"},{"id":"b"}]`))
	})
	out, err := c.ListIssues(context.Background(), "P", ListIssuesOptions{})
	require.NoError(t, err)
	results, ok := out["results"].([]any)
	require.True(t, ok)
	require.Len(t, results, 2)
}

func TestListIssuesPassesDictThrough(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"id":"a"}],"count":1,"next_cursor":"x"}`))
	})
	out, err := c.ListIssues(context.Background(), "P", ListIssuesOptions{})
	require.NoError(t, err)
	require.Equal(t, float64(1), out["count"])
	require.Equal(t, "x", out["next_cursor"])
}

func TestListProjectsUnwrapsResults(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/workspaces/ws/projects/", r.URL.Path)
		_, _ = w.Write([]byte(`{"results":[{"id":"p1","identifier":"TOOLS","name":"Tools"}]}`))
	})
	projects, err := c.ListProjects(context.Background())
	require.NoError(t, err)
	require.Len(t, projects, 1)
	require.Equal(t, "TOOLS", projects[0]["identifier"])
}

func TestListStatesUnwrapsResults(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"id":"s","group":"completed"}]}`))
	})
	states, err := c.ListStates(context.Background(), "P")
	require.NoError(t, err)
	require.Len(t, states, 1)
	require.Equal(t, "completed", states[0]["group"])
}

func TestAddCommentDefaultsToInternal(t *testing.T) {
	var captured map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &captured))
		_, _ = w.Write([]byte(`{"id":"c1"}`))
	})
	_, err := c.AddComment(context.Background(), "P", "I", "<p>hi</p>", "")
	require.NoError(t, err)
	require.Equal(t, "INTERNAL", captured["access"])
	require.Equal(t, "<p>hi</p>", captured["comment_html"])
}

func TestDeleteIssueSucceedsOn204(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	require.NoError(t, c.DeleteIssue(context.Background(), "P", "I"))
}

func TestDownloadAttachmentSameHostAddsAuth(t *testing.T) {
	var hadAuth bool
	c, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		hadAuth = r.Header.Get("X-API-Key") != ""
		_, _ = w.Write([]byte("hello"))
	})

	// Relative URL → same-host with auth.
	hadAuth = false
	body, _, err := c.DownloadAttachment(context.Background(), "/api/v1/assets/1/")
	require.NoError(t, err)
	require.Equal(t, "hello", string(body))
	require.True(t, hadAuth)

	// Same base URL → same-host with auth.
	hadAuth = false
	_, _, err = c.DownloadAttachment(context.Background(), ts.URL+"/api/v1/assets/1/")
	require.NoError(t, err)
	require.True(t, hadAuth)
}

func TestDownloadAttachmentExternalSkipsAuth(t *testing.T) {
	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.Header.Get("X-API-Key"))
		_, _ = w.Write([]byte("blob"))
	}))
	t.Cleanup(external.Close)

	c, _ := newTestClient(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("primary should not be hit")
	})
	body, _, err := c.DownloadAttachment(context.Background(), external.URL+"/blob")
	require.NoError(t, err)
	require.Equal(t, "blob", string(body))
}

func TestUploadAttachmentBlobAddsTrailingSlash(t *testing.T) {
	var contentType, gotURL string
	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		gotURL = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(ts.Close)

	c, err := NewClient(Config{BaseURL: ts.URL, Workspace: "ws", APIToken: "t"})
	require.NoError(t, err)
	require.NoError(t, c.UploadAttachmentBlob(
		context.Background(),
		ts.URL+"/upload", // no trailing slash on purpose
		map[string]string{"key": "value", "acl": "private"},
		"hello.txt",
		strings.NewReader("hi"),
		"text/plain",
	))
	require.Equal(t, "/upload/", gotURL)
	require.True(t, strings.HasPrefix(contentType, "multipart/form-data; boundary="))
	require.Contains(t, string(gotBody), `name="key"`)
	require.Contains(t, string(gotBody), `name="acl"`)
	require.Contains(t, string(gotBody), `filename="hello.txt"`)
	require.Contains(t, string(gotBody), "hi")
}

func TestCompleteAttachmentUpload204IsEmptyMap(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPatch, r.Method)
		w.WriteHeader(http.StatusNoContent)
	})
	out, err := c.CompleteAttachmentUpload(context.Background(), "P", "I", "asset")
	require.NoError(t, err)
	require.Equal(t, map[string]any{}, out)
}
