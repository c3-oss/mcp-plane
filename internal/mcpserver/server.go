// Package mcpserver registers every Plane operation as an MCP tool and
// serves them over stdio.
package mcpserver

import (
	"context"
	"io"
	"os"

	"github.com/c3-oss/mcp-plane/internal/buildinfo"
	"github.com/c3-oss/mcp-plane/internal/plane"
	"github.com/mark3labs/mcp-go/server"
)

// Server bundles the MCP wiring with the Plane client. Construction is
// stateless: tools are registered once in New and run for the life of the
// process.
type Server struct {
	client *plane.Client
	mcp    *server.MCPServer
}

// New builds a Server with every Plane tool registered.
func New(client *plane.Client) *Server {
	mcp := server.NewMCPServer(
		"mcp-plane",
		buildinfo.Version,
		server.WithToolCapabilities(true),
	)
	s := &Server{client: client, mcp: mcp}
	s.registerAll()
	return s
}

// MCPServer exposes the underlying server.MCPServer for tests/in-process
// callers (e.g. server.NewInProcessClient).
func (s *Server) MCPServer() *server.MCPServer { return s.mcp }

// Serve runs the stdio transport until the context is cancelled or stdin
// is closed.
func (s *Server) Serve(ctx context.Context) error {
	stdio := server.NewStdioServer(s.mcp)
	return stdio.Listen(ctx, io.Reader(os.Stdin), io.Writer(os.Stdout))
}

func (s *Server) registerAll() {
	s.registerProjectTools()
	s.registerIssueTools()
	s.registerStateTools()
	s.registerLabelTools()
	s.registerCommentTools()
	s.registerActivityTools()
	s.registerAttachmentTools()
	s.registerTransferTool()
	s.registerWorkpadTool()
	s.registerWorkspaceTools()
}
