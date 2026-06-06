package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadFromRequiresBaseURL(t *testing.T) {
	_, err := LoadFrom(map[string]string{
		"PLANE_API_TOKEN": "real-token",
		"PLANE_WORKSPACE": "acme",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "PLANE_BASE_URL")
}

func TestLoadFromTrimsWhitespace(t *testing.T) {
	s, err := LoadFrom(map[string]string{
		"PLANE_BASE_URL":  "  https://plane.example.com  ",
		"PLANE_API_TOKEN": "  real-token\n",
		"PLANE_WORKSPACE": "\tacme ",
	})
	require.NoError(t, err)
	require.Equal(t, "https://plane.example.com", s.BaseURL)
	require.Equal(t, "real-token", s.APIToken)
	require.Equal(t, "acme", s.Workspace)
}

func TestLoadFromRejectsPlaceholderToken(t *testing.T) {
	for _, placeholder := range []string{"", "changeme", "CHANGE-ME", "placeholder", "TBD"} {
		t.Run(placeholder, func(t *testing.T) {
			_, err := LoadFrom(map[string]string{
				"PLANE_BASE_URL":  "https://plane.example.com",
				"PLANE_API_TOKEN": placeholder,
				"PLANE_WORKSPACE": "acme",
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "PLANE_API_TOKEN")
		})
	}
}

func TestLoadFromRequiresWorkspace(t *testing.T) {
	_, err := LoadFrom(map[string]string{
		"PLANE_BASE_URL":  "https://plane.example.com",
		"PLANE_API_TOKEN": "real",
		"PLANE_WORKSPACE": "",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "PLANE_WORKSPACE")
}

func TestSettingsStringRedactsToken(t *testing.T) {
	s := Settings{BaseURL: "https://x", Workspace: "ws", APIToken: "secret-XYZ"}
	str := s.String()
	require.NotContains(t, str, "secret-XYZ")
	require.True(t, strings.Contains(str, "***"))
}
