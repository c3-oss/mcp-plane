package plane

import (
	"context"
	"net/http"
)

// CommentAccessInternal and CommentAccessExternal match the values accepted
// by Plane's comment endpoint.
const (
	CommentAccessInternal = "INTERNAL"
	CommentAccessExternal = "EXTERNAL"
)

// ListComments returns every comment on an issue.
func (c *Client) ListComments(ctx context.Context, projectID, issueID string) ([]Comment, error) {
	raw, err := c.doRaw(ctx, http.MethodGet, c.issuePath(projectID, issueID, "comments"), nil, nil)
	if err != nil {
		return nil, err
	}
	items := extractResults(raw)
	out := make([]Comment, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, nil
}

// AddComment posts a new HTML comment to an issue. Pass an empty access
// string to default to INTERNAL (matches the Python client default).
func (c *Client) AddComment(ctx context.Context, projectID, issueID, commentHTML, access string) (Comment, error) {
	if access == "" {
		access = CommentAccessInternal
	}
	payload := map[string]any{
		"comment_html": commentHTML,
		"access":       access,
	}
	var out Comment
	if err := c.doJSON(ctx, http.MethodPost, c.issuePath(projectID, issueID, "comments"), nil, payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateComment patches an existing comment.
func (c *Client) UpdateComment(ctx context.Context, projectID, issueID, commentID, commentHTML, access string) (Comment, error) {
	if access == "" {
		access = CommentAccessInternal
	}
	payload := map[string]any{
		"comment_html": commentHTML,
		"access":       access,
	}
	var out Comment
	if err := c.doJSON(ctx, http.MethodPatch, c.issuePath(projectID, issueID, "comments", commentID), nil, payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteComment removes a comment from an issue.
func (c *Client) DeleteComment(ctx context.Context, projectID, issueID, commentID string) error {
	return c.doJSON(ctx, http.MethodDelete, c.issuePath(projectID, issueID, "comments", commentID), nil, nil, nil)
}
