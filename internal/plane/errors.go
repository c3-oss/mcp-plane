package plane

import (
	"errors"
	"fmt"
)

// APIError represents a non-2xx response from Plane. The raw body is
// truncated to maxErrorBodySize bytes so callers can include it in MCP
// tool-error results without dumping huge payloads.
type APIError struct {
	StatusCode int
	Method     string
	URL        string
	Body       []byte
}

const maxErrorBodySize = 4 * 1024

func (e *APIError) Error() string {
	if len(e.Body) == 0 {
		return fmt.Sprintf("plane: %s %s: status %d", e.Method, e.URL, e.StatusCode)
	}
	return fmt.Sprintf("plane: %s %s: status %d: %s", e.Method, e.URL, e.StatusCode, string(e.Body))
}

// IsNotFound reports whether err is an *APIError with a 404 status.
func IsNotFound(err error) bool {
	var a *APIError
	if errors.As(err, &a) {
		return a.StatusCode == 404
	}
	return false
}

// IsUnauthorized reports whether err is an *APIError with a 401 or 403 status.
func IsUnauthorized(err error) bool {
	var a *APIError
	if errors.As(err, &a) {
		return a.StatusCode == 401 || a.StatusCode == 403
	}
	return false
}

// IsBadRequest reports whether err is an *APIError with a 400 or 422 status.
func IsBadRequest(err error) bool {
	var a *APIError
	if errors.As(err, &a) {
		return a.StatusCode == 400 || a.StatusCode == 422
	}
	return false
}
