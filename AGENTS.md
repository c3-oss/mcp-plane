# AGENTS

Canonical guide for humans and AI coding agents working in this repository.
Read this end-to-end before proposing substantial changes.

## What this is

`mcp-plane` is an MCP server (stdio) that exposes the Plane REST API as
tools. See [`docs/architecture.md`](docs/architecture.md) for the package
layering.

## Project shape

- **Module**: `github.com/c3-oss/mcp-plane` (declared in `go.mod`).
- **Layout**:
  - `cmd/mcp-plane/` — the binary's `main` package; sets up signal handling
    and delegates to `internal/cli`.
  - `internal/`:
    - `buildinfo` — version metadata stamped at link time.
    - `cli` — cobra root command; default action runs the MCP server, the
      only subcommand is `version`.
    - `logging` — slog handler → stderr (stdout is the JSON-RPC transport).
    - `config` — env-var loader (`PLANE_BASE_URL`, `PLANE_API_TOKEN`, `PLANE_WORKSPACE`).
    - `plane` — pure HTTP client over the Plane REST API. No MCP types.
    - `workpad` — render & upsert the workpad comment.
    - `transfer` — cross-project issue transfer flow.
    - `mcpserver` — wires every operation as an MCP tool and serves it
      over stdio via `mark3labs/mcp-go`.
  - `pkg/` — exportable packages; empty by default.
  - `scripts/` — small bash utilities (rename, coverage report).
  - `docs/` — `tools.md`, `architecture.md`, `verifying.md`.
- **Generated outputs** (gitignored): `bin/`, `dist/`, `coverage.*`.

## Configuration contract

Three env vars, all read at startup by `internal/config`:

| Variable | Required | Default | Notes |
|---|---|---|---|
| `PLANE_API_TOKEN` | yes | — | placeholders (`changeme`, `tbd`, …) are rejected. |
| `PLANE_WORKSPACE` | yes | — | workspace slug. |
| `PLANE_BASE_URL` | yes | — | trimmed, no trailing slash. |

The token is never logged and is redacted by `config.Settings.String()`.

## Build, test, develop

Toolchain is pinned in `devbox.json`. Enter the shell with `devbox shell` and
all subsequent commands resolve through the pinned versions.

Common tasks (run via `just <target>`):

| Target | Purpose |
|---|---|
| `just build` | compile every `cmd/*` into `bin/` |
| `just test` | `go test ./...` |
| `just test-race` | full race detector + `-count=1` |
| `just cover` | coverage profile and per-function totals |
| `just lint` | `golangci-lint run ./...` |
| `just lint-sec` | `gosec` static security analysis |
| `just lint-vuln` | `govulncheck` against `./...` |
| `just quality` | Markdown lint, link check, secret scan |
| `just ci` | local mirror of the PR pipeline |
| `just snapshot` | GoReleaser dry-run with SBOMs |
| `just docker-build` | local Docker image |
| `just clean` | drop build outputs |

`just tools` installs Go-based binaries (`govulncheck`, `gosec`) into
`./bin`. Devbox does this automatically on `devbox shell` entry.

## Coding style

- `gofumpt` formats Go (stricter than `gofmt`). Run via golangci-lint or
  your editor.
- `goimports` orders imports.
- Tests live alongside source: `*_test.go` in the same package.
- Test assertions use `github.com/stretchr/testify/require` for failures
  that should abort the test and `assert` for soft checks.
- Logging goes to stderr (via `internal/logging`); stdout is reserved for
  command output that callers may pipe.
- Comments explain *why*, not *what*. Identifiers carry the *what*.

## Commits and PRs

Conventional Commits with **mandatory scope** are enforced by commitlint
and validated by CI.

- Format: `<type>(<scope>): <subject>` — e.g. `feat(cli): add status command`.
- Allowed types (any `@commitlint/config-conventional` type): `feat`, `fix`,
  `chore`, `docs`, `test`, `build`, `ci`, `refactor`, `perf`, `style`, `revert`.
- The CI changelog (via GoReleaser) groups `feat` and `fix` and drops
  `docs`, `test`, `chore`, `ci`, and merge commits.

PRs target `master`. CI runs five jobs that must all pass: `quality`,
`test`, `lint`, `security`, `build` (Ubuntu + macOS matrix).

## Hooks

`./.husky/` is wired automatically by `pnpm install` (which runs as part
of `devbox shell`).

- `pre-commit` → `lint-staged` (Markdown) + `gitleaks protect --staged`.
- `commit-msg` → `commitlint` (mandatory scope).
- `pre-push` → `just hooks-pre-push` (== `just quality`).

## Releases

Push a tag `v<semver>` to `master` and `.github/workflows/release.yml`
takes it from there:

1. GoReleaser builds binaries for linux/darwin × amd64/arm64, publishes
   archives + SHA-256 checksums + per-archive SPDX SBOMs (via Syft) to
   GitHub Releases.
2. `scripts/publish-npm.sh` stamps the tag version into the five
   `npm/*/package.json` files, copies the freshly-built binaries into
   the four platform sub-packages, and publishes them all to npm —
   `@c3-oss/mcp-plane` (main shim) and `@c3-oss/mcp-plane-{darwin,linux}-{amd64,arm64}`.
   This is what makes `npx -y @c3-oss/mcp-plane` work.
3. Docker pushes a multi-arch image to GHCR.

`just snapshot` is the local equivalent for steps 1+3 and writes
everything to `dist/`. The npm step requires `NODE_AUTH_TOKEN` and is
CI-only.

The npm publish needs a granular `NPM_TOKEN` secret in the GitHub repo,
scoped to the `@c3-oss` org with publish permission.

## What is intentionally *not* here

- No `Makefile` — `just` only.
- No `pkg/` content out of the box — add carefully, exports become contracts.
- No CGO. Switch `CGO_ENABLED=1` in `Dockerfile` and `.goreleaser.yaml` if you need it.
- No web framework / DB driver / message broker — add what the project needs.
- No `.claude/agents` or `.codex/skills` content — the structure is ready; populate per project.
