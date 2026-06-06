package plane

import (
	"context"
	"net/http"
)

// ListActivities returns the activity entries for an issue.
func (c *Client) ListActivities(ctx context.Context, projectID, issueID string) ([]Activity, error) {
	raw, err := c.doRaw(ctx, http.MethodGet, c.issuePath(projectID, issueID, "activities"), nil, nil)
	if err != nil {
		return nil, err
	}
	items := extractResults(raw)
	out := make([]Activity, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, nil
}
