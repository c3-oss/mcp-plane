package plane

import (
	"context"
	"net/http"
)

// ListStates returns the workflow states defined on a project.
func (c *Client) ListStates(ctx context.Context, projectID string) ([]State, error) {
	raw, err := c.doRaw(ctx, http.MethodGet, c.projectPath(projectID, "states"), nil, nil)
	if err != nil {
		return nil, err
	}
	items := extractResults(raw)
	out := make([]State, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, nil
}
