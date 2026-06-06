package plane

import (
	"path/filepath"
	"strconv"
	"strings"
)

// BuildIssueWebURL returns the browser URL for a Plane issue, or an empty
// string when any input is missing.
func BuildIssueWebURL(baseURL, workspace, identifier string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	workspace = strings.TrimSpace(workspace)
	identifier = strings.TrimSpace(identifier)
	if baseURL == "" || workspace == "" || identifier == "" {
		return ""
	}
	return baseURL + "/" + workspace + "/browse/" + identifier
}

// ExtractIssueIdentifier returns the display identifier (e.g. "PROJ-123").
// If the issue payload omits "identifier", a "<project_code>-<sequence_id>"
// form is synthesised when the sequence_id is available.
func ExtractIssueIdentifier(issue Issue, projectCode string) string {
	if issue == nil {
		return ""
	}
	if id, ok := issue["identifier"].(string); ok && id != "" {
		return id
	}
	seqStr := stringifySequenceID(issue["sequence_id"])
	if seqStr == "" {
		return ""
	}
	code := strings.TrimSpace(projectCode)
	if code == "" {
		if proj, ok := issue["project"].(map[string]any); ok {
			if c, ok := proj["identifier"].(string); ok && c != "" {
				code = c
			} else if c, ok := proj["project_code"].(string); ok && c != "" {
				code = c
			}
		}
	}
	if code == "" {
		return seqStr
	}
	return code + "-" + seqStr
}

// ExtractIssueUUID returns the issue's UUID, looking first at "id" and
// falling back to "uuid".
func ExtractIssueUUID(issue Issue) string {
	if issue == nil {
		return ""
	}
	if v, ok := issue["id"].(string); ok && v != "" {
		return v
	}
	if v, ok := issue["uuid"].(string); ok && v != "" {
		return v
	}
	return ""
}

// ExtractIssueName returns the issue's title ("name" field).
func ExtractIssueName(issue Issue) string {
	if issue == nil {
		return ""
	}
	if v, ok := issue["name"].(string); ok {
		return v
	}
	return ""
}

// ExtractAssigneeIDs flattens any combination of strings and dicts into a
// list of assignee UUIDs.
func ExtractAssigneeIDs(raw any) []string {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	var ids []string
	for _, item := range list {
		switch x := item.(type) {
		case string:
			if x != "" {
				ids = append(ids, x)
			}
		case map[string]any:
			if id, ok := x["id"].(string); ok && id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// AttachmentName extracts the display name from an attachment dict.
func AttachmentName(att Attachment) string {
	if att == nil {
		return ""
	}
	if v, ok := att["name"].(string); ok && v != "" {
		return v
	}
	if attrs, ok := att["attributes"].(map[string]any); ok {
		if v, ok := attrs["name"].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// AttachmentSize extracts the size in bytes from an attachment dict.
func AttachmentSize(att Attachment) int64 {
	if att == nil {
		return 0
	}
	if v, ok := numericInt64(att["size"]); ok {
		return v
	}
	if attrs, ok := att["attributes"].(map[string]any); ok {
		if v, ok := numericInt64(attrs["size"]); ok {
			return v
		}
	}
	return 0
}

// AttachmentKind classifies an attachment by file extension.
func AttachmentKind(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return "image"
	case ".txt", ".md":
		return "text"
	default:
		return "other"
	}
}

func stringifySequenceID(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(x)
	case float64:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case int:
		return strconv.Itoa(x)
	}
	return ""
}

func numericInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case int64:
		return x, true
	case int:
		return int64(x), true
	case string:
		if n, err := strconv.ParseInt(x, 10, 64); err == nil {
			return n, true
		}
	}
	return 0, false
}
