package mcpserver

import (
	"errors"

	"github.com/c3-oss/mcp-plane/internal/plane"
	"github.com/mark3labs/mcp-go/mcp"
)

// toolError converts a Plane error into an MCP tool error result. The token
// is never included (the Plane client only sets the path and status in the
// error type), so this stays safe to surface to callers.
func toolError(err error) *mcp.CallToolResult {
	var api *plane.APIError
	if errors.As(err, &api) {
		switch {
		case plane.IsNotFound(err):
			return mcp.NewToolResultErrorf("plane not found: %s %s", api.Method, api.URL)
		case plane.IsUnauthorized(err):
			return mcp.NewToolResultErrorf("plane unauthorized: %s %s (check PLANE_API_TOKEN)", api.Method, api.URL)
		case plane.IsBadRequest(err):
			return mcp.NewToolResultErrorf("plane bad request: %s %s: %s", api.Method, api.URL, string(api.Body))
		}
		return mcp.NewToolResultErrorf("plane error %d: %s %s: %s", api.StatusCode, api.Method, api.URL, string(api.Body))
	}
	return mcp.NewToolResultErrorFromErr("tool error", err)
}
