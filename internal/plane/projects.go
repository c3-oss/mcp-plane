package plane

import (
	"context"
	"net/http"
)

// ListProjects returns the projects visible in the configured workspace.
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	raw, err := c.doRaw(ctx, http.MethodGet, c.workspacePath("projects"), nil, nil)
	if err != nil {
		return nil, err
	}
	items := extractResults(raw)
	out := make([]Project, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, nil
}
