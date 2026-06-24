# mcp-plane

[![CI](https://github.com/c3-oss/mcp-plane/actions/workflows/ci.yml/badge.svg)](https://github.com/c3-oss/mcp-plane/actions/workflows/ci.yml)
[![Release](https://github.com/c3-oss/mcp-plane/actions/workflows/release.yml/badge.svg)](https://github.com/c3-oss/mcp-plane/actions/workflows/release.yml)
[![License: CC0 1.0](https://img.shields.io/badge/license-CC0%201.0-lightgrey.svg)](LICENSE)

mcp server (stdio) exposing the plane rest api as tools so an mcp client can
drive plane directly.

## Configuration

Set three environment variables:

| Variable | Required | Default | Notes |
|---|---|---|---|
| `PLANE_API_TOKEN` | yes | — | personal access token from plane → settings → api tokens. placeholders like `changeme` are rejected. |
| `PLANE_WORKSPACE` | yes | — | workspace slug (the value in plane urls). |
| `PLANE_BASE_URL` | yes | — | self-hosted plane base url, no trailing slash. |

## Running

The published npm package is the recommended path — no build step, no
binary to manage manually:

```bash
PLANE_BASE_URL=... PLANE_API_TOKEN=... PLANE_WORKSPACE=... \
  npx -y @c3-oss/mcp-plane
```

`@c3-oss/mcp-plane` is a thin JS shim that delegates to the prebuilt Go
binary matching your platform (darwin/linux × amd64/arm64), distributed
via npm `optionalDependencies`. No `postinstall` download.

Or build locally:

```bash
just build
PLANE_BASE_URL=... PLANE_API_TOKEN=... PLANE_WORKSPACE=... ./bin/mcp-plane
```

Either way, the binary speaks JSON-RPC 2.0 over stdio per the
[MCP specification](https://modelcontextprotocol.io). Logs go to stderr;
stdout is reserved for the protocol.

### Registering with an MCP client

Claude Desktop / Claude Code (`~/.claude.json`), via npx:

```json
{
  "mcpServers": {
    "plane": {
      "command": "npx",
      "args": ["-y", "@c3-oss/mcp-plane"],
      "env": {
        "PLANE_BASE_URL": "https://plane.example.com",
        "PLANE_API_TOKEN": "...",
        "PLANE_WORKSPACE": "..."
      }
    }
  }
}
```

Or pointing at a locally-built binary:

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

## Tools

23 tools registered. See [`docs/tools.md`](docs/tools.md) for the full table
with input schemas; summary below.

| Group | Tools |
|---|---|
| Projects | `plane_project_list` |
| Issues | `plane_issue_create`, `plane_issue_list`, `plane_issue_get`, `plane_issue_get_by_identifier`, `plane_issue_update`, `plane_issue_delete` |
| States / Labels | `plane_state_list`, `plane_label_list` |
| Comments | `plane_comment_list`, `plane_comment_add`, `plane_comment_update`, `plane_comment_delete` |
| Activities | `plane_activity_list` |
| Attachments | `plane_attachment_list`, `plane_attachment_init_upload`, `plane_attachment_complete_upload`, `plane_attachment_upload`, `plane_attachment_read` |
| Higher-level | `plane_issue_transfer`, `plane_issue_workpad_upsert` |
| Sanity | `plane_workspace_info`, `plane_health` |

Inputs accept plane-native ids (uuid for project/issue/state/label, or the
workspace identifier like `PROJ-7` for the by-identifier lookups).
`plane_project_list` returns project ids and identifiers for project-scoped
tools.

## Verifying

See [`docs/verifying.md`](docs/verifying.md) for an end-to-end stdin/stdout
probe and a manual smoke test.

## Development

Toolchain is pinned via [devbox](https://www.jetify.com/devbox); enter with
`devbox shell`. then:

```bash
just build         # compile bin/mcp-plane
just test-race     # full race detector
just lint          # golangci-lint v2
just ci            # local mirror of the PR pipeline
just snapshot      # goreleaser --snapshot (writes dist/ with SBOMs)
just docker-build  # build the local docker image
```

See [`AGENTS.md`](AGENTS.md) for the canonical project guide.

## License

[CC0 1.0 Universal](LICENSE) — public domain dedication.
