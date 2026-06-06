package plane

import (
	"context"
	"net/http"
)

// ListLabels returns the labels defined on a project.
func (c *Client) ListLabels(ctx context.Context, projectID string) ([]Label, error) {
	raw, err := c.doRaw(ctx, http.MethodGet, c.projectPath(projectID, "labels"), nil, nil)
	if err != nil {
		return nil, err
	}
	items := extractResults(raw)
	out := make([]Label, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, nil
}
