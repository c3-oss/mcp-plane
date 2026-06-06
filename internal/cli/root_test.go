package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRootDefaultActionInvokesServeFunc(t *testing.T) {
	called := false
	t.Cleanup(func() { serveFunc = defaultServe })
	serveFunc = func(*cobra.Command) error {
		called = true
		return nil
	}

	cmd := newRootCmd()
	cmd.SetArgs(nil)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	require.NoError(t, cmd.Execute())
	require.True(t, called, "default action should call serveFunc")
}

func TestRootHelpMentionsPlane(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--help"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	require.NoError(t, cmd.Execute())
	require.True(t, strings.Contains(out.String(), "Plane"))
}
