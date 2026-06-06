package plane

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildIssueWebURL(t *testing.T) {
	require.Equal(t, "https://plane.example.com/ws/browse/PROJ-1",
		BuildIssueWebURL("https://plane.example.com/", "ws", "PROJ-1"))
	require.Equal(t, "", BuildIssueWebURL("", "ws", "PROJ-1"))
	require.Equal(t, "", BuildIssueWebURL("https://x", "", "PROJ-1"))
	require.Equal(t, "", BuildIssueWebURL("https://x", "ws", ""))
}

func TestExtractIssueIdentifierPrefersExplicitField(t *testing.T) {
	require.Equal(t, "PROJ-7", ExtractIssueIdentifier(map[string]any{"identifier": "PROJ-7"}, ""))
}

func TestExtractIssueIdentifierFallsBackToSequenceAndCode(t *testing.T) {
	require.Equal(t, "PROJ-9", ExtractIssueIdentifier(map[string]any{"sequence_id": float64(9)}, "PROJ"))
	require.Equal(t, "PROJ-9", ExtractIssueIdentifier(
		map[string]any{"sequence_id": float64(9), "project": map[string]any{"identifier": "PROJ"}},
		"",
	))
	require.Equal(t, "9", ExtractIssueIdentifier(map[string]any{"sequence_id": float64(9)}, ""))
}

func TestExtractAssigneeIDsHandlesMixedShapes(t *testing.T) {
	ids := ExtractAssigneeIDs([]any{
		"u1",
		map[string]any{"id": "u2"},
		map[string]any{"display_name": "no-id"},
		"",
	})
	require.Equal(t, []string{"u1", "u2"}, ids)
}

func TestAttachmentKindClassification(t *testing.T) {
	require.Equal(t, "image", AttachmentKind("photo.PNG"))
	require.Equal(t, "image", AttachmentKind("photo.jpeg"))
	require.Equal(t, "text", AttachmentKind("README.md"))
	require.Equal(t, "text", AttachmentKind("notes.txt"))
	require.Equal(t, "other", AttachmentKind("data.json"))
	require.Equal(t, "other", AttachmentKind("noext"))
}
