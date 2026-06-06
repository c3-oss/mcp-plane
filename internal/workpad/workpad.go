// Package workpad renders an issue-status workpad and upserts it as a comment
// on a Plane issue, looking the existing comment up by marker.
package workpad

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/c3-oss/mcp-plane/internal/plane"
)

// Marker is the HTML/text token used to identify a workpad comment.
const Marker = "MCP Workpad"

// CommentClient is the minimal subset of *plane.Client we need. Defined as
// an interface so workpad can be unit-tested without a live HTTP server.
type CommentClient interface {
	ListComments(ctx context.Context, projectID, issueID string) ([]plane.Comment, error)
	AddComment(ctx context.Context, projectID, issueID, commentHTML, access string) (plane.Comment, error)
	UpdateComment(ctx context.Context, projectID, issueID, commentID, commentHTML, access string) (plane.Comment, error)
}

// RenderArgs are the inputs accepted by Render. Empty optional fields map to
// "-" or sensible defaults.
type RenderArgs struct {
	Issue            plane.Issue
	Summary          string
	DelegationStatus string
	RepoKey          string
	RepoPath         string
	PublishRemote    string
	Branch           string
	PRURL            string
	Validation       []string
	Blockers         []string
	Marker           string
}

// Render builds the workpad HTML payload.
func Render(args RenderArgs) string {
	marker := args.Marker
	if marker == "" {
		marker = Marker
	}
	validation := args.Validation
	if len(validation) == 0 {
		validation = []string{"Validation pending."}
	}
	blockers := args.Blockers
	if len(blockers) == 0 {
		blockers = []string{"None"}
	}
	delegationStatus := args.DelegationStatus
	if delegationStatus == "" {
		delegationStatus = "active"
	}

	artifacts := []string{
		"Issue: " + issueIdentifier(args.Issue),
		"Repo key: " + dashIfEmpty(args.RepoKey),
		"Repo Path: " + dashIfEmpty(args.RepoPath),
		"Publish Remote: " + dashIfEmpty(args.PublishRemote),
		"Branch: " + dashIfEmpty(args.Branch),
		"PR: " + dashIfEmpty(args.PRURL),
	}
	delegation := []string{
		"Status: " + delegationStatus,
		"Execution model: Native subagents coordinated by the parent agent.",
		"Progress updates: Synchronized back into this workpad by the parent agent.",
	}

	sections := []struct {
		title string
		body  string
	}{
		{"Summary", args.Summary},
		{"Status", "Issue state: " + issueState(args.Issue)},
		{"Delegation", bulletList(delegation)},
		{"Validation", bulletList(validation)},
		{"Artifacts", bulletList(artifacts)},
		{"Blockers", bulletList(blockers)},
	}

	var b strings.Builder
	fmt.Fprintf(&b, "<h2>%s</h2>", html.EscapeString(marker))
	for _, s := range sections {
		fmt.Fprintf(&b, "<h3>%s</h3>", html.EscapeString(s.title))
		fmt.Fprintf(&b, "<pre>%s</pre>", html.EscapeString(s.body))
	}
	return b.String()
}

// UpsertComment finds an existing workpad comment by marker and updates it,
// or adds a new comment when no marker is found. Returns the comment that
// was either updated or created.
func UpsertComment(ctx context.Context, c CommentClient, projectID, issueID, contentHTML, marker string) (plane.Comment, error) {
	if marker == "" {
		marker = Marker
	}
	comments, err := c.ListComments(ctx, projectID, issueID)
	if err != nil {
		return nil, err
	}
	for _, comment := range comments {
		if containsMarker(comment, marker) {
			id, _ := comment["id"].(string)
			if id != "" {
				return c.UpdateComment(ctx, projectID, issueID, id, contentHTML, "")
			}
		}
	}
	return c.AddComment(ctx, projectID, issueID, contentHTML, "")
}

func containsMarker(comment plane.Comment, marker string) bool {
	if comment == nil {
		return false
	}
	for _, key := range []string{"comment_html", "comment_text", "comment_stripped", "comment"} {
		if v, ok := comment[key].(string); ok && strings.Contains(v, marker) {
			return true
		}
	}
	return false
}

func issueIdentifier(issue plane.Issue) string {
	id := plane.ExtractIssueIdentifier(issue, "")
	if id == "" {
		return "-"
	}
	return id
}

func issueState(issue plane.Issue) string {
	if issue == nil {
		return "-"
	}
	if st, ok := issue["state"].(map[string]any); ok {
		if name, ok := st["name"].(string); ok && name != "" {
			return name
		}
	}
	if name, ok := issue["state"].(string); ok && strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return "-"
}

func dashIfEmpty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func bulletList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = "- " + item
	}
	return strings.Join(parts, "\n")
}
