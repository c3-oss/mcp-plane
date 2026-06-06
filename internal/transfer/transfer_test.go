package transfer

import (
	"context"
	"strings"
	"testing"

	"github.com/c3-oss/mcp-plane/internal/plane"
	"github.com/stretchr/testify/require"
)

type fakeAPI struct {
	baseURL   string
	workspace string

	issues map[string]plane.Issue // key = projectID + ":" + issueID

	labelsByProject map[string][]plane.Label
	statesByProject map[string][]plane.State

	createdIssues []plane.Issue
	addedComments []addedComment
	updatedIssues []updatedIssue
}

type addedComment struct {
	projectID, issueID, html string
}
type updatedIssue struct {
	projectID, issueID string
	opts               plane.IssueOptions
}

func newFakeAPI() *fakeAPI {
	return &fakeAPI{
		baseURL:         "https://plane.example.com",
		workspace:       "ws",
		issues:          map[string]plane.Issue{},
		labelsByProject: map[string][]plane.Label{},
		statesByProject: map[string][]plane.State{},
	}
}

func (f *fakeAPI) BaseURL() string   { return f.baseURL }
func (f *fakeAPI) Workspace() string { return f.workspace }

func (f *fakeAPI) GetIssue(_ context.Context, projectID, issueID string) (plane.Issue, error) {
	return f.issues[projectID+":"+issueID], nil
}

func (f *fakeAPI) CreateIssue(_ context.Context, projectID, name string, opts plane.IssueOptions) (plane.Issue, error) {
	issue := plane.Issue{
		"id":         "target-uuid",
		"identifier": "TGT-1",
		"name":       name,
	}
	if opts.State != nil {
		issue["state"] = map[string]any{"id": *opts.State}
	}
	if opts.Labels != nil {
		issue["labels"] = *opts.Labels
	}
	f.createdIssues = append(f.createdIssues, issue)
	f.issues[projectID+":target-uuid"] = issue
	return issue, nil
}

func (f *fakeAPI) UpdateIssue(_ context.Context, projectID, issueID string, _ *string, opts plane.IssueOptions) (plane.Issue, error) {
	f.updatedIssues = append(f.updatedIssues, updatedIssue{projectID, issueID, opts})
	return f.issues[projectID+":"+issueID], nil
}

func (f *fakeAPI) ListLabels(_ context.Context, projectID string) ([]plane.Label, error) {
	return f.labelsByProject[projectID], nil
}

func (f *fakeAPI) ListStates(_ context.Context, projectID string) ([]plane.State, error) {
	return f.statesByProject[projectID], nil
}

func (f *fakeAPI) AddComment(_ context.Context, projectID, issueID, html, _ string) (plane.Comment, error) {
	f.addedComments = append(f.addedComments, addedComment{projectID, issueID, html})
	return plane.Comment{"id": "c"}, nil
}

func TestTransferTranslatesLabelsCaseInsensitively(t *testing.T) {
	f := newFakeAPI()
	f.issues["P-SRC:I-SRC"] = plane.Issue{
		"id":         "I-SRC",
		"identifier": "SRC-7",
		"name":       "do the thing",
		"labels":     []any{"src-bug-id", "src-perf-id"},
	}
	f.labelsByProject["P-SRC"] = []plane.Label{
		{"id": "src-bug-id", "name": "Bug"},
		{"id": "src-perf-id", "name": "Perf"},
	}
	f.labelsByProject["P-TGT"] = []plane.Label{
		{"id": "tgt-bug-id", "name": "bug"}, // case-insensitive match
		// No "Perf" label → should be skipped.
	}

	out, err := Transfer(context.Background(), f, Params{
		SourceProjectID:   "P-SRC",
		SourceIssueID:     "I-SRC",
		SourceProjectCode: "SRC",
		TargetProjectID:   "P-TGT",
		TargetProjectCode: "TGT",
	})
	require.NoError(t, err)
	require.Len(t, f.createdIssues, 1)
	require.Equal(t, []string{"Perf"}, out.SkippedLabels)
	labels, _ := f.createdIssues[0]["labels"].([]string)
	require.Equal(t, []string{"tgt-bug-id"}, labels)
}

func TestTransferStateFallbackWhenTargetMissing(t *testing.T) {
	f := newFakeAPI()
	f.issues["P-SRC:I-SRC"] = plane.Issue{
		"id":   "I-SRC",
		"name": "x",
		"state": map[string]any{
			"id":   "src-state",
			"name": "Doing",
		},
	}
	f.statesByProject["P-TGT"] = []plane.State{
		{"id": "tgt-todo", "name": "Todo"},
	}
	out, err := Transfer(context.Background(), f, Params{
		SourceProjectID: "P-SRC", SourceIssueID: "I-SRC",
		TargetProjectID: "P-TGT", TargetProjectCode: "TGT",
	})
	require.NoError(t, err)
	require.True(t, out.StateFallbackUsed)
	require.Nil(t, f.createdIssues[0]["state"])
}

func TestTransferCloseSourceFindsCompletedState(t *testing.T) {
	f := newFakeAPI()
	f.issues["P-SRC:I-SRC"] = plane.Issue{"id": "I-SRC", "name": "x"}
	f.statesByProject["P-SRC"] = []plane.State{
		{"id": "src-todo", "name": "Todo", "group": "unstarted"},
		{"id": "src-done", "name": "Done", "group": "completed"},
	}

	out, err := Transfer(context.Background(), f, Params{
		SourceProjectID: "P-SRC", SourceIssueID: "I-SRC",
		TargetProjectID: "P-TGT", CloseSource: true,
	})
	require.NoError(t, err)
	require.True(t, out.SourceClosed)
	require.Len(t, f.updatedIssues, 1)
	require.Equal(t, "src-done", *f.updatedIssues[0].opts.State)
}

func TestTransferCrossLinkComments(t *testing.T) {
	f := newFakeAPI()
	f.issues["P-SRC:I-SRC"] = plane.Issue{
		"id": "I-SRC", "name": "x", "identifier": "SRC-7",
	}
	out, err := Transfer(context.Background(), f, Params{
		SourceProjectID: "P-SRC", SourceIssueID: "I-SRC", SourceProjectCode: "SRC",
		TargetProjectID: "P-TGT", TargetProjectCode: "TGT",
	})
	require.NoError(t, err)
	require.Empty(t, out.CommentFailures)
	require.Len(t, f.addedComments, 2)
	require.True(t, strings.Contains(f.addedComments[0].html, "Transferred to"))
	require.True(t, strings.Contains(f.addedComments[1].html, "Transferred from"))
	require.True(t, strings.Contains(f.addedComments[0].html, "TGT-1"))
	require.True(t, strings.Contains(f.addedComments[1].html, "SRC-7"))
}
