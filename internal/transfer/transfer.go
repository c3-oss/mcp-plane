// Package transfer implements the cross-project issue transfer flow.
//
// Org-context resolution (Slack ↔ Plane, project codes ↔ UUIDs) is out of
// scope; callers pass Plane-native UUIDs and project codes directly.
package transfer

import (
	"context"
	"fmt"
	"strings"

	"github.com/c3-oss/mcp-plane/internal/plane"
)

// API is the minimal slice of *plane.Client this package needs. Defined as
// an interface so the transfer flow can be unit-tested without a live HTTP
// server.
type API interface {
	BaseURL() string
	Workspace() string
	GetIssue(ctx context.Context, projectID, issueID string) (plane.Issue, error)
	CreateIssue(ctx context.Context, projectID, name string, opts plane.IssueOptions) (plane.Issue, error)
	UpdateIssue(ctx context.Context, projectID, issueID string, name *string, opts plane.IssueOptions) (plane.Issue, error)
	ListLabels(ctx context.Context, projectID string) ([]plane.Label, error)
	ListStates(ctx context.Context, projectID string) ([]plane.State, error)
	AddComment(ctx context.Context, projectID, issueID, commentHTML, access string) (plane.Comment, error)
}

// Params describes a transfer request.
type Params struct {
	SourceProjectID   string
	SourceIssueID     string
	SourceProjectCode string
	TargetProjectID   string
	TargetProjectCode string
	CloseSource       bool
}

// Outcome is the structured result of a transfer.
type Outcome struct {
	SourceIssue       plane.Issue `json:"source_issue"`
	TargetIssue       plane.Issue `json:"target_issue"`
	TargetProjectCode string      `json:"target_project_code"`
	SourceClosed      bool        `json:"source_closed"`
	CommentFailures   []string    `json:"comment_failures"`
	SkippedLabels     []string    `json:"skipped_labels"`
	StateFallbackUsed bool        `json:"state_fallback_used"`
}

// Transfer copies a source issue into the target project, translating labels
// and state by name. Comments cross-linking the two issues are added in both
// directions; failures are reported in Outcome.CommentFailures instead of
// aborting the transfer.
func Transfer(ctx context.Context, api API, p Params) (Outcome, error) {
	if p.SourceProjectID == "" || p.SourceIssueID == "" || p.TargetProjectID == "" {
		return Outcome{}, fmt.Errorf("transfer: missing required parameter")
	}

	source, err := api.GetIssue(ctx, p.SourceProjectID, p.SourceIssueID)
	if err != nil {
		return Outcome{}, fmt.Errorf("transfer: load source: %w", err)
	}

	targetLabels, skippedLabels, err := translateLabels(ctx, api, source, p.SourceProjectID, p.TargetProjectID)
	if err != nil {
		return Outcome{}, fmt.Errorf("transfer: translate labels: %w", err)
	}
	targetState, fallback, err := translateState(ctx, api, source, p.TargetProjectID)
	if err != nil {
		return Outcome{}, fmt.Errorf("transfer: translate state: %w", err)
	}

	opts := plane.IssueOptions{}
	if desc, ok := source["description_html"].(string); ok {
		opts.DescriptionHTML = &desc
	}
	if assignees := plane.ExtractAssigneeIDs(source["assignees"]); len(assignees) > 0 {
		opts.Assignees = &assignees
	}
	if pr, ok := source["priority"].(string); ok && pr != "" {
		opts.Priority = &pr
	}
	if targetState != "" {
		opts.State = &targetState
	}
	if len(targetLabels) > 0 {
		opts.Labels = &targetLabels
	}

	created, err := api.CreateIssue(ctx, p.TargetProjectID, plane.ExtractIssueName(source), opts)
	if err != nil {
		return Outcome{}, fmt.Errorf("transfer: create target: %w", err)
	}

	commentFailures := crossLinkComments(ctx, api, source, created, p)

	var sourceClosed bool
	if p.CloseSource {
		closed, err := closeSourceIssue(ctx, api, p.SourceProjectID, p.SourceIssueID)
		if err != nil {
			return Outcome{}, fmt.Errorf("transfer: close source: %w", err)
		}
		sourceClosed = closed
	}

	return Outcome{
		SourceIssue:       source,
		TargetIssue:       created,
		TargetProjectCode: p.TargetProjectCode,
		SourceClosed:      sourceClosed,
		CommentFailures:   commentFailures,
		SkippedLabels:     skippedLabels,
		StateFallbackUsed: fallback,
	}, nil
}

func translateLabels(ctx context.Context, api API, source plane.Issue, sourceProjectID, targetProjectID string) ([]string, []string, error) {
	sourceLabelIDs := extractLabelIDs(source["labels"])
	if len(sourceLabelIDs) == 0 {
		return nil, nil, nil
	}
	sourceLabels, err := api.ListLabels(ctx, sourceProjectID)
	if err != nil {
		return nil, nil, fmt.Errorf("list source labels: %w", err)
	}
	sourceLookup := labelIDToName(sourceLabels)
	var sourceNames []string
	for _, id := range sourceLabelIDs {
		if name, ok := sourceLookup[id]; ok && name != "" {
			sourceNames = append(sourceNames, name)
		}
	}
	if len(sourceNames) == 0 {
		return nil, nil, nil
	}

	targetLabels, err := api.ListLabels(ctx, targetProjectID)
	if err != nil {
		return nil, nil, fmt.Errorf("list target labels: %w", err)
	}
	targetByName := map[string]string{}
	for _, l := range targetLabels {
		name, _ := l["name"].(string)
		id, _ := l["id"].(string)
		if name != "" && id != "" {
			targetByName[strings.ToLower(name)] = id
		}
	}

	var resolved, skipped []string
	for _, name := range sourceNames {
		if id, ok := targetByName[strings.ToLower(name)]; ok {
			resolved = append(resolved, id)
		} else {
			skipped = append(skipped, name)
		}
	}
	return resolved, skipped, nil
}

func translateState(ctx context.Context, api API, source plane.Issue, targetProjectID string) (string, bool, error) {
	st, ok := source["state"].(map[string]any)
	if !ok {
		return "", false, nil
	}
	name, ok := st["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return "", false, nil
	}
	target, err := api.ListStates(ctx, targetProjectID)
	if err != nil {
		return "", false, fmt.Errorf("list target states: %w", err)
	}
	for _, ts := range target {
		tn, _ := ts["name"].(string)
		if strings.EqualFold(tn, name) {
			id, _ := ts["id"].(string)
			return id, false, nil
		}
	}
	return "", true, nil
}

func crossLinkComments(ctx context.Context, api API, source, created plane.Issue, p Params) []string {
	srcIdent := plane.ExtractIssueIdentifier(source, p.SourceProjectCode)
	tgtIdent := plane.ExtractIssueIdentifier(created, p.TargetProjectCode)
	srcURL := plane.BuildIssueWebURL(api.BaseURL(), api.Workspace(), srcIdent)
	tgtURL := plane.BuildIssueWebURL(api.BaseURL(), api.Workspace(), tgtIdent)

	var failures []string

	sourceComment := fmt.Sprintf("<p>Transferred to %s.</p>", anchor(tgtIdent, tgtURL, plane.ExtractIssueUUID(created)))
	if _, err := api.AddComment(ctx, p.SourceProjectID, p.SourceIssueID, sourceComment, ""); err != nil {
		failures = append(failures, "source")
	}

	targetUUID := plane.ExtractIssueUUID(created)
	targetComment := fmt.Sprintf("<p>Transferred from %s.</p>", anchor(srcIdent, srcURL, p.SourceIssueID))
	if _, err := api.AddComment(ctx, p.TargetProjectID, targetUUID, targetComment, ""); err != nil {
		failures = append(failures, "target")
	}
	return failures
}

func anchor(label, url, fallback string) string {
	if label == "" {
		label = fallback
	}
	if url == "" {
		return label
	}
	return fmt.Sprintf(`<a href="%s">%s</a>`, url, label)
}

func closeSourceIssue(ctx context.Context, api API, projectID, issueID string) (bool, error) {
	states, err := api.ListStates(ctx, projectID)
	if err != nil {
		return false, fmt.Errorf("list source states: %w", err)
	}
	for _, st := range states {
		group, _ := st["group"].(string)
		id, _ := st["id"].(string)
		if group == "completed" && id != "" {
			if _, err := api.UpdateIssue(ctx, projectID, issueID, nil, plane.IssueOptions{State: &id}); err != nil {
				return false, fmt.Errorf("patch source state: %w", err)
			}
			return true, nil
		}
	}
	return false, nil
}

func extractLabelIDs(raw any) []string {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	var ids []string
	for _, item := range list {
		switch x := item.(type) {
		case string:
			if x != "" {
				ids = append(ids, x)
			}
		case map[string]any:
			if id, ok := x["id"].(string); ok && id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func labelIDToName(labels []plane.Label) map[string]string {
	m := make(map[string]string, len(labels))
	for _, l := range labels {
		id, _ := l["id"].(string)
		name, _ := l["name"].(string)
		if id != "" {
			m[id] = name
		}
	}
	return m
}
