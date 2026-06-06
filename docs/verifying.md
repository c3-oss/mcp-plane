# verifying mcp-plane

## Static gates

Mirror of CI:

```bash
just ci   # tidy-check vet lint lint-sec lint-vuln test-race build
```

Coverage spot-check:

```bash
just cover
```

## End-to-end probe (stdin/stdout JSON-RPC)

`mcp-plane` speaks JSON-RPC 2.0 over stdio. The two-line `printf` below is
the minimum handshake an MCP client performs: `initialize`, then a real
request.

### List the registered tools

```bash
export PLANE_BASE_URL=https://plane.example.com
export PLANE_API_TOKEN=<token>
export PLANE_WORKSPACE=<slug>
just build

printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"probe","version":"0"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | bin/mcp-plane --log-level=debug \
  | jq .
```

Expect 22 tools in the second response's `result.tools`.

### Call a tool

```bash
PROJECT_ID=<project-uuid>
printf '%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"probe","version":"0"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"plane_state_list","arguments":{"project_id":"'"$PROJECT_ID"'"}}}' \
  | bin/mcp-plane \
  | jq .
```

Expect `result.content[0].text` to be a JSON string containing `"results"`
with the project's workflow states.

## Manual smoke test

1. Create a throwaway issue:

   ```json
   { "name": "plane_issue_create",
     "arguments": { "project_id": "<P>", "name": "mcp-plane smoke" } }
   ```

2. Run `plane_issue_workpad_upsert` against it (with `summary: "first"`).
   Run it a second time with `summary: "second"`. The two calls must return
   the **same** `comment_id` — `MCP Workpad` is a single comment that
   is updated in place, not appended.

3. Run `plane_issue_transfer` to a second test project with
   `close_source: false`. Verify:
   - the new target issue exists,
   - the response's `skipped_labels` lists every source label that has no
     same-name match in the target project,
   - both issues have a fresh comment containing `Transferred …`.

If those three flows succeed, the server is wired end-to-end.

## Registering with an MCP client

Claude Desktop / Claude Code (`~/.claude.json`):

```json
{
  "mcpServers": {
    "plane": {
      "command": "/abs/path/to/bin/mcp-plane",
      "env": {
        "PLANE_BASE_URL": "https://plane.example.com",
        "PLANE_API_TOKEN": "...",
        "PLANE_WORKSPACE": "..."
      }
    }
  }
}
```

Restart the client. The tools appear under the `plane` server and can be
invoked from any conversation.
