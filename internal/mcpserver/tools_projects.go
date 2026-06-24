package mcpserver

import (
	"context"

	"github.com/c3-oss/mcp-plane/internal/plane"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerProjectTools() {
	s.mcp.AddTool(mcp.NewTool("plane_project_list",
		mcp.WithDescription("List visible Plane projects with compact id, identifier, and name fields."),
	), s.handleProjectList)
}

func (s *Server) handleProjectList(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projects, err := s.client.ListProjects(ctx)
	if err != nil {
		return toolError(err), nil
	}
	out := make([]map[string]any, 0, len(projects))
	for _, project := range projects {
		out = append(out, compactProject(project))
	}
	return asTextResult(map[string]any{"results": out})
}

func compactProject(project plane.Project) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"id", "identifier", "name"} {
		if value, ok := project[key].(string); ok && value != "" {
			out[key] = value
		}
	}
	return out
}
