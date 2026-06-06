// Command mcp-plane runs an MCP server (stdio) that exposes the Plane
// REST API as MCP tools. Configure with PLANE_BASE_URL, PLANE_API_TOKEN,
// and PLANE_WORKSPACE environment variables.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/c3-oss/mcp-plane/internal/cli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if err := cli.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
