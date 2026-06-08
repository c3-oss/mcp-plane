# @c3-oss/mcp-plane

MCP server (stdio) exposing the [Plane](https://plane.so) REST API as
tools so an MCP client can drive Plane directly.

```sh
npx -y @c3-oss/mcp-plane
```

Or install globally:

```sh
npm install -g @c3-oss/mcp-plane
mcp-plane
```

`mcp-plane` is a small JavaScript shim that delegates to the prebuilt
Go binary matching your platform, distributed through npm
`optionalDependencies`. There is no `postinstall` download — npm
filters by `os` and `cpu` and only installs the sub-package your
machine needs.

Supported platforms:

- macOS arm64 / amd64
- Linux amd64 / arm64

Configure with three environment variables (`PLANE_BASE_URL`,
`PLANE_API_TOKEN`, `PLANE_WORKSPACE`) and register the command in your
MCP client. See the
[main README](https://github.com/c3-oss/mcp-plane#readme) for client
config examples and the
[full tools list](https://github.com/c3-oss/mcp-plane/blob/master/docs/tools.md).

Source code, documentation, and issue tracker:
<https://github.com/c3-oss/mcp-plane>
