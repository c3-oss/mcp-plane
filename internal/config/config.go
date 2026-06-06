// Package config loads Plane connection settings from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
)

var placeholderTokens = map[string]struct{}{
	"changeme":    {},
	"change-me":   {},
	"placeholder": {},
	"replace-me":  {},
	"todo":        {},
	"tbd":         {},
}

// Settings is the resolved Plane connection configuration.
type Settings struct {
	BaseURL   string
	Workspace string
	APIToken  string
}

// String redacts the API token so Settings is safe to log.
func (s Settings) String() string {
	return fmt.Sprintf("config.Settings{BaseURL:%q Workspace:%q APIToken:***}", s.BaseURL, s.Workspace)
}

// Missing returns the names of required fields that are absent or placeholders.
func (s Settings) Missing() []string {
	var missing []string
	if strings.TrimSpace(s.BaseURL) == "" {
		missing = append(missing, "PLANE_BASE_URL")
	}
	if !hasRealValue(s.APIToken) {
		missing = append(missing, "PLANE_API_TOKEN")
	}
	if strings.TrimSpace(s.Workspace) == "" {
		missing = append(missing, "PLANE_WORKSPACE")
	}
	return missing
}

// Validate returns a single error describing every missing/invalid field, or
// nil if the configuration is complete.
func (s Settings) Validate() error {
	missing := s.Missing()
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing Plane settings: %s", strings.Join(missing, ", "))
}

// Load reads PLANE_BASE_URL / PLANE_API_TOKEN / PLANE_WORKSPACE from the
// process environment and returns a validated Settings value.
func Load() (Settings, error) {
	return LoadFrom(envLookup(os.Environ()))
}

// LoadFrom builds Settings from an arbitrary key/value map. Useful for tests.
func LoadFrom(env map[string]string) (Settings, error) {
	s := Settings{
		BaseURL:   strings.TrimSpace(env["PLANE_BASE_URL"]),
		Workspace: strings.TrimSpace(env["PLANE_WORKSPACE"]),
		APIToken:  strings.TrimSpace(env["PLANE_API_TOKEN"]),
	}
	if err := s.Validate(); err != nil {
		return Settings{}, err
	}
	return s, nil
}

func hasRealValue(token string) bool {
	t := strings.TrimSpace(token)
	if t == "" {
		return false
	}
	if _, isPlaceholder := placeholderTokens[strings.ToLower(t)]; isPlaceholder {
		return false
	}
	return true
}

func envLookup(environ []string) map[string]string {
	out := make(map[string]string, len(environ))
	for _, kv := range environ {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			out[kv[:i]] = kv[i+1:]
		}
	}
	return out
}
