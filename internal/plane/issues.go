package plane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// applyIssueOptions copies non-nil pointer fields from opts into the payload.
func applyIssueOptions(payload map[string]any, opts IssueOptions) {
	if opts.DescriptionHTML != nil {
		payload["description_html"] = *opts.DescriptionHTML
	}
	if opts.Assignees != nil {
		payload["assignees"] = *opts.Assignees
	}
	if opts.Priority != nil {
		payload["priority"] = *opts.Priority
	}
	if opts.State != nil {
		payload["state"] = *opts.State
	}
	if opts.Labels != nil {
		payload["labels"] = *opts.Labels
	}
	if opts.Parent != nil {
		payload["parent"] = *opts.Parent
	}
}

// CreateIssue creates an issue in the given project.
func (c *Client) CreateIssue(ctx context.Context, projectID, name string, opts IssueOptions) (Issue, error) {
	payload := map[string]any{"name": name}
	applyIssueOptions(payload, opts)
	var out Issue
	if err := c.doJSON(ctx, http.MethodPost, c.projectPath(projectID, "issues"), nil, payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListIssues returns the paginated list of issues for a project.
func (c *Client) ListIssues(ctx context.Context, projectID string, opts ListIssuesOptions) (IssueList, error) {
	params := url.Values{}
	if opts.PerPage != nil {
		params.Set("per_page", strconv.Itoa(*opts.PerPage))
	}
	if opts.Cursor != nil {
		params.Set("cursor", *opts.Cursor)
	}
	if opts.Archived != nil {
		if *opts.Archived {
			params.Set("archived", "true")
		} else {
			params.Set("archived", "false")
		}
	}
	setPtrParam(params, "state_group", opts.StateGroup)
	setPtrParam(params, "assignees", opts.Assignees)
	setPtrParam(params, "labels", opts.Labels)
	setPtrParam(params, "priority", opts.Priority)
	setPtrParam(params, "created_at", opts.CreatedAt)
	setPtrParam(params, "target_date", opts.TargetDate)
	setPtrParam(params, "completed_at", opts.CompletedAt)
	setPtrParam(params, "name", opts.Name)
	setPtrParam(params, "order_by", opts.OrderBy)
	setPtrParam(params, "expand", opts.Expand)
	setPtrParam(params, "parent", opts.Parent)

	raw, err := c.doRaw(ctx, http.MethodGet, c.projectPath(projectID, "issues"), params, nil)
	if err != nil {
		return nil, err
	}
	return normalizeCollection(raw), nil
}

// GetIssue fetches a single issue by UUID.
func (c *Client) GetIssue(ctx context.Context, projectID, issueID string) (Issue, error) {
	var out Issue
	if err := c.doJSON(ctx, http.MethodGet, c.issuePath(projectID, issueID), nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetIssueByIdentifier fetches a single issue by its workspace identifier
// (e.g. "PROJ-123").
func (c *Client) GetIssueByIdentifier(ctx context.Context, identifier string) (Issue, error) {
	var out Issue
	if err := c.doJSON(ctx, http.MethodGet, c.workspacePath("issues", identifier), nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateIssue patches an issue. A nil name pointer means "do not change".
func (c *Client) UpdateIssue(ctx context.Context, projectID, issueID string, name *string, opts IssueOptions) (Issue, error) {
	payload := map[string]any{}
	if name != nil {
		payload["name"] = *name
	}
	applyIssueOptions(payload, opts)
	var out Issue
	if err := c.doJSON(ctx, http.MethodPatch, c.issuePath(projectID, issueID), nil, payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteIssue removes an issue.
func (c *Client) DeleteIssue(ctx context.Context, projectID, issueID string) error {
	return c.doJSON(ctx, http.MethodDelete, c.issuePath(projectID, issueID), nil, nil, nil)
}

// MarshalIssueOptions is a tiny helper used by tests to inspect the payload
// that CreateIssue/UpdateIssue would send.
func MarshalIssueOptions(name *string, opts IssueOptions) ([]byte, error) {
	payload := map[string]any{}
	if name != nil {
		payload["name"] = *name
	}
	applyIssueOptions(payload, opts)
	return json.Marshal(payload)
}
