package plane

import (
	"bytes"
	"encoding/json"
)

// normalizeCollection turns either a bare JSON array or a dict-shaped
// collection into a dict with a "results" key.
func normalizeCollection(raw []byte) map[string]any {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return map[string]any{"results": []any{}}
	}
	switch trimmed[0] {
	case '[':
		var arr []any
		if err := json.Unmarshal(trimmed, &arr); err != nil {
			return map[string]any{"results": []any{}}
		}
		return map[string]any{"results": arr}
	case '{':
		var obj map[string]any
		if err := json.Unmarshal(trimmed, &obj); err != nil {
			return map[string]any{"results": []any{}}
		}
		if _, ok := obj["results"]; !ok {
			obj["results"] = []any{}
		}
		return obj
	default:
		return map[string]any{"results": []any{}}
	}
}

// extractResults returns the "results" slice from a normalized collection.
func extractResults(raw []byte) []any {
	col := normalizeCollection(raw)
	v, _ := col["results"].([]any)
	if v == nil {
		return []any{}
	}
	return v
}
