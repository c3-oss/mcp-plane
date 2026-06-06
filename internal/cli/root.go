// Package cli wires the Cobra command tree for mcp-plane.
package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/c3-oss/mcp-plane/internal/config"
	"github.com/c3-oss/mcp-plane/internal/logging"
	"github.com/c3-oss/mcp-plane/internal/mcpserver"
	"github.com/c3-oss/mcp-plane/internal/plane"
)

// serveFunc is the default action of the root command; tests override it.
var serveFunc = defaultServe

func newRootCmd() *cobra.Command {
	var logLevel string

	cmd := &cobra.Command{
		Use:           "mcp-plane",
		Short:         "MCP server (stdio) for the Plane REST API",
		Long: "mcp-plane runs an MCP server over stdio that exposes the Plane REST API as tools.\n" +
			"Configure with the PLANE_BASE_URL, PLANE_API_TOKEN, and PLANE_WORKSPACE environment\n" +
			"variables, then register the binary with your MCP client.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return logging.Configure(logLevel)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return serveFunc(cmd)
		},
	}

	cmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	cmd.AddCommand(newVersionCmd())

	return cmd
}

func defaultServe(cmd *cobra.Command) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client, err := plane.NewClient(plane.Config{
		BaseURL:   cfg.BaseURL,
		Workspace: cfg.Workspace,
		APIToken:  cfg.APIToken,
	})
	if err != nil {
		return err
	}
	return mcpserver.New(client).Serve(cmd.Context())
}

// Execute runs the root command and returns any error encountered.
func Execute() error {
	return newRootCmd().Execute()
}

// ExecuteContext runs the root command bound to ctx so signal cancellation
// propagates into long-running commands like the stdio server.
func ExecuteContext(ctx context.Context) error {
	return newRootCmd().ExecuteContext(ctx)
}
