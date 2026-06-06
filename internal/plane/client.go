package plane

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultRequestTimeout  = 30 * time.Second
	defaultDownloadTimeout = 60 * time.Second
)

// Config describes how to connect to a Plane workspace.
type Config struct {
	BaseURL    string
	Workspace  string
	APIToken   string
	HTTPClient *http.Client // optional; default = http.Client{Timeout: 30s}
	UserAgent  string       // optional; default = "mcp-plane"
}

// Client is a thin wrapper around net/http that talks to the Plane REST API.
//
// All paths are joined with c.baseURL; auth and content-type headers are
// injected automatically. Downloads and S3 uploads use a longer-timeout
// secondary client so a slow blob does not strand the main pool.
type Client struct {
	baseURL   string
	workspace string
	apiToken  string
	userAgent string

	http     *http.Client
	download *http.Client
}

// NewClient returns a Client validated against the configuration. Errors are
// returned for missing fields or placeholder API tokens.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIToken) == "" {
		return nil, errors.New("plane: missing APIToken")
	}
	if strings.TrimSpace(cfg.Workspace) == "" {
		return nil, errors.New("plane: missing Workspace")
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return nil, errors.New("plane: missing BaseURL")
	}
	primary := cfg.HTTPClient
	if primary == nil {
		primary = &http.Client{Timeout: defaultRequestTimeout}
	}
	download := *primary
	download.Timeout = defaultDownloadTimeout
	ua := cfg.UserAgent
	if ua == "" {
		ua = "mcp-plane"
	}
	return &Client{
		baseURL:   base,
		workspace: cfg.Workspace,
		apiToken:  cfg.APIToken,
		userAgent: ua,
		http:      primary,
		download:  &download,
	}, nil
}

// BaseURL returns the configured Plane base URL.
func (c *Client) BaseURL() string { return c.baseURL }

// Workspace returns the configured workspace slug.
func (c *Client) Workspace() string { return c.workspace }

// workspacePath joins fragments under /api/v1/workspaces/{workspace}.
func (c *Client) workspacePath(fragments ...string) string {
	parts := make([]string, 0, len(fragments)+3)
	parts = append(parts, "/api/v1/workspaces", c.workspace)
	parts = append(parts, fragments...)
	return strings.Join(parts, "/") + "/"
}

// projectPath joins fragments under
// /api/v1/workspaces/{workspace}/projects/{projectID}.
func (c *Client) projectPath(projectID string, fragments ...string) string {
	parts := make([]string, 0, len(fragments)+5)
	parts = append(parts, "/api/v1/workspaces", c.workspace, "projects", projectID)
	parts = append(parts, fragments...)
	return strings.Join(parts, "/") + "/"
}

// issuePath joins fragments under .../projects/{projectID}/issues/{issueID}.
func (c *Client) issuePath(projectID, issueID string, fragments ...string) string {
	parts := make([]string, 0, len(fragments)+7)
	parts = append(parts, "/api/v1/workspaces", c.workspace, "projects", projectID, "issues", issueID)
	parts = append(parts, fragments...)
	return strings.Join(parts, "/") + "/"
}

// doJSON performs a JSON request, decoding any 2xx response body into out
// (pointer to slice/map). out may be nil to discard the body.
func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	raw, err := c.doRaw(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	if out == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("plane: decode %s %s: %w", method, path, err)
	}
	return nil
}

// doRaw performs the request and returns the raw response bytes for callers
// that need to normalize the shape (collections).
func (c *Client) doRaw(ctx context.Context, method, path string, query url.Values, body any) ([]byte, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u = u + "?" + query.Encode()
	}

	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("plane: encode %s %s: %w", method, path, err)
		}
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, buf)
	if err != nil {
		return nil, fmt.Errorf("plane: build %s %s: %w", method, path, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiToken)
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		slog.Warn("plane request failed", "method", method, "path", path, "error", err)
		return nil, fmt.Errorf("plane: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024*1024))
	slog.Debug("plane request",
		"method", method,
		"path", path,
		"status", resp.StatusCode,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, nil
	}
	trimmed := respBody
	if len(trimmed) > maxErrorBodySize {
		trimmed = trimmed[:maxErrorBodySize]
	}
	return nil, &APIError{
		StatusCode: resp.StatusCode,
		Method:     method,
		URL:        path,
		Body:       trimmed,
	}
}

// setPtr writes a string pointer's value into params if non-nil.
func setPtrParam(params url.Values, key string, v *string) {
	if v != nil {
		params.Set(key, *v)
	}
}
