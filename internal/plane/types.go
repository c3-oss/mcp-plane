package plane

// We deliberately model API entities as map[string]any. Plane evolves its
// schema frequently, and the MCP server's contract is "pass through whatever
// the API returned". A strongly-typed model would force a breaking change
// every time the upstream adds a field.

// Issue represents a single Plane issue.
type Issue = map[string]any

// State represents a workflow state.
type State = map[string]any

// Label represents a project label.
type Label = map[string]any

// Comment represents an issue comment.
type Comment = map[string]any

// Activity represents an issue activity entry.
type Activity = map[string]any

// Attachment represents an issue attachment metadata record.
type Attachment = map[string]any

// IssueOptions are optional fields accepted by CreateIssue / UpdateIssue.
// Pointer fields preserve the distinction between "absent" and "empty":
// a nil pointer means "do not send"; a non-nil pointer means "send this value
// even if empty".
type IssueOptions struct {
	DescriptionHTML *string
	Assignees       *[]string
	Priority        *string
	State           *string
	Labels          *[]string
	Parent          *string
}

// IssueList is the dict-shaped result of ListIssues. Unknown fields land in
// Extra so callers and clients see whatever pagination metadata the upstream
// chose to return.
type IssueList = map[string]any

// ListIssuesOptions holds optional query parameters for ListIssues. All
// fields are optional; nil means "do not include this query parameter".
type ListIssuesOptions struct {
	PerPage     *int
	Cursor      *string
	Archived    *bool
	StateGroup  *string
	Assignees   *string
	Labels      *string
	Priority    *string
	CreatedAt   *string
	TargetDate  *string
	CompletedAt *string
	Name        *string
	OrderBy     *string
	Expand      *string
	Parent      *string
}
