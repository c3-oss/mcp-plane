# mcp-plane architecture

A one-page tour of how a tool call flows through the server.

```
mcp client (stdio)
   │  jsonrpc 2.0
   ▼
cmd/mcp-plane/main.go              signal.NotifyContext + cli.ExecuteContext
   │
   ▼
internal/cli                       cobra root command; default action = serve
   │
   ▼
internal/config                    PLANE_* env loading with placeholder rejection
   │
   ▼
internal/plane (.Client)           net/http client; auth header; 30 s timeout
   │                               ──► download client (60 s) for attachments
   ▼
internal/mcpserver                 22 tools registered; arg-coercion helpers
   │                               and error mapping to mcp.CallToolResult
   ├── tools_issues.go             6 tools
   ├── tools_states.go             1 tool
   ├── tools_labels.go             1 tool
   ├── tools_comments.go           4 tools
   ├── tools_activities.go         1 tool
   ├── tools_attachments.go        5 tools (incl. high-level upload/read)
   ├── tools_transfer.go           1 tool → internal/transfer
   ├── tools_workpad.go            1 tool → internal/workpad
   └── tools_workspace.go          2 sanity tools
```

## Package responsibilities

| Package | Talks to | Knows nothing about |
|---|---|---|
| `internal/config` | env vars | http, mcp |
| `internal/plane` | plane http api | mcp, cli |
| `internal/workpad` | a `CommentClient` interface (subset of plane.Client) | mcp, cli |
| `internal/transfer` | an `API` interface (subset of plane.Client) | mcp, cli |
| `internal/mcpserver` | mark3labs/mcp-go + plane + workpad + transfer | environment, cli |
| `internal/cli` | config + plane + mcpserver | the mcp protocol details |
| `internal/logging` | slog → stderr | everything else |

Two principles enforce this layering:

1. **Stdout is the protocol**. Anything that wants to talk to a human writes
   to stderr via `internal/logging`. The plane client logs request method,
   path, status, duration — never the body, never the token.
2. **Pointer-field optionality**. `plane.IssueOptions` and
   `plane.ListIssuesOptions` use `*string` / `*[]string` / `*int` etc., so
   "absent" (nil) and "set to empty" (non-nil pointer to "") survive the
   round-trip from json arguments through to the request body.

## Out of scope

| Feature | Why we skipped it |
|---|---|
| User-directory mapping (chat-handle ↔ Plane user) | depends on out-of-band config; mcp clients pass uuids directly. |
| Per-user preferences | local cli concern, not server state. |
| Attachment disk-caching on read | mcp returns bytes (text inline / images base64); no need to spill to disk. |
