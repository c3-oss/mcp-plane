package workpad

import (
	"context"
	"strings"
	"testing"

	"github.com/c3-oss/mcp-plane/internal/plane"
	"github.com/stretchr/testify/require"
)

func TestRenderDefaultsForOptionalFields(t *testing.T) {
	out := Render(RenderArgs{Summary: "go!"})
	require.Contains(t, out, "<h2>MCP Workpad</h2>")
	require.Contains(t, out, "Validation pending.")
	require.Contains(t, out, "Blockers")
	require.Contains(t, out, "- None")
	require.Contains(t, out, "Status: active")
	require.Contains(t, out, "Repo key: -")
}

func TestRenderEscapesHTML(t *testing.T) {
	out := Render(RenderArgs{Summary: "<script>"})
	require.NotContains(t, out, "<script>x") // raw not present
	require.Contains(t, out, "&lt;script&gt;")
}

func TestRenderHonoursAllFields(t *testing.T) {
	out := Render(RenderArgs{
		Issue: map[string]any{
			"identifier": "PROJ-7",
			"state":      map[string]any{"name": "In Progress"},
		},
		Summary:          "summary line",
		DelegationStatus: "paused",
		RepoKey:          "core",
		RepoPath:         "/x/y",
		PublishRemote:    "origin",
		Branch:           "feat/x",
		PRURL:            "https://gh/x/y/pull/1",
		Validation:       []string{"ran tests", "linted"},
		Blockers:         []string{"waiting on design"},
	})
	require.Contains(t, out, "Issue: PROJ-7")
	require.Contains(t, out, "Issue state: In Progress")
	require.Contains(t, out, "Status: paused")
	require.Contains(t, out, "Repo key: core")
	require.Contains(t, out, "Branch: feat/x")
	require.Contains(t, out, "https://gh/x/y/pull/1")
	require.Contains(t, out, "- ran tests")
	require.Contains(t, out, "- waiting on design")
}

type fakeClient struct {
	comments  []plane.Comment
	addCalls  int
	updCalls  int
	lastAdd   string
	lastUpdID string
	lastUpd   string
}

func (f *fakeClient) ListComments(_ context.Context, _, _ string) ([]plane.Comment, error) {
	return f.comments, nil
}

func (f *fakeClient) AddComment(_ context.Context, _, _, html, _ string) (plane.Comment, error) {
	f.addCalls++
	f.lastAdd = html
	return map[string]any{"id": "new-id", "comment_html": html}, nil
}

func (f *fakeClient) UpdateComment(_ context.Context, _, _, commentID, html, _ string) (plane.Comment, error) {
	f.updCalls++
	f.lastUpdID = commentID
	f.lastUpd = html
	return map[string]any{"id": commentID, "comment_html": html}, nil
}

func TestUpsertCreatesWhenMarkerMissing(t *testing.T) {
	f := &fakeClient{comments: []plane.Comment{
		{"id": "c1", "comment_html": "<p>random</p>"},
	}}
	out, err := UpsertComment(context.Background(), f, "P", "I", "<p>workpad</p>", "")
	require.NoError(t, err)
	require.Equal(t, 1, f.addCalls)
	require.Equal(t, 0, f.updCalls)
	require.Equal(t, "<p>workpad</p>", f.lastAdd)
	require.Equal(t, "new-id", out["id"])
}

func TestUpsertUpdatesWhenMarkerPresent(t *testing.T) {
	html := "<h2>" + Marker + "</h2><p>old</p>"
	f := &fakeClient{comments: []plane.Comment{
		{"id": "old", "comment_html": html},
	}}
	out, err := UpsertComment(context.Background(), f, "P", "I", "<h2>"+Marker+"</h2><p>new</p>", "")
	require.NoError(t, err)
	require.Equal(t, 0, f.addCalls)
	require.Equal(t, 1, f.updCalls)
	require.Equal(t, "old", f.lastUpdID)
	require.True(t, strings.Contains(f.lastUpd, "new"))
	require.Equal(t, "old", out["id"])
}
