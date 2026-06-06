package mcpserver

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// asTextResult marshals v as compact JSON and wraps it in a single
// text-content tool result.
func asTextResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultErrorf("encode result: %v", err), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

// argStringPtr returns a pointer to the string value of key, or nil if the
// key is absent or holds a non-string.
func argStringPtr(args map[string]any, key string) *string {
	if v, ok := args[key].(string); ok {
		return &v
	}
	return nil
}

// argStringSlicePtr returns a pointer to a []string built from the array
// stored under key. Non-string items are skipped. Returns nil if key is
// missing or not an array.
func argStringSlicePtr(args map[string]any, key string) *[]string {
	raw, ok := args[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return &out
}

// argStringSlice returns []string from args[key], or nil. Useful when the
// caller does not want pointer semantics.
func argStringSlice(args map[string]any, key string) []string {
	if p := argStringSlicePtr(args, key); p != nil {
		return *p
	}
	return nil
}

// argIntPtr coerces args[key] from float64 (JSON's number) into *int.
func argIntPtr(args map[string]any, key string) *int {
	switch v := args[key].(type) {
	case float64:
		n := int(v)
		return &n
	case int:
		return &v
	}
	return nil
}

// argInt64Ptr coerces args[key] into *int64 the same way.
func argInt64Ptr(args map[string]any, key string) *int64 {
	switch v := args[key].(type) {
	case float64:
		n := int64(v)
		return &n
	case int64:
		return &v
	case int:
		n := int64(v)
		return &n
	}
	return nil
}

// argBoolPtr returns a pointer to args[key] as bool, or nil if absent.
func argBoolPtr(args map[string]any, key string) *bool {
	if v, ok := args[key].(bool); ok {
		return &v
	}
	return nil
}
