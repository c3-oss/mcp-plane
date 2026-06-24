# mcp-plane tool reference

Every tool returns a single text content block whose body is a compact JSON
document. Errors map to MCP tool-error results (`isError: true`) — the
calling agent decides how to recover.

All endpoints are relative to `https://<PLANE_BASE_URL>/api/v1/workspaces/<PLANE_WORKSPACE>/`.

## Projects

### `plane_project_list`

GET `/projects/`.

Returns compact project records:

```json
{ "results": [{ "id": "...", "identifier": "TOOLS", "name": "Squad Web/Tools" }] }
```

Use `id` as `project_id` in project-scoped tools. Use `identifier` as the
project code in cross-project transfer arguments and issue identifiers.

## Issues

### `plane_issue_create`

POST `/projects/{project_id}/issues/`.

Required: `project_id`, `name`.
Optional: `description_html` (string), `assignees` (array of plane user ids),
`priority` (`urgent` | `high` | `medium` | `low` | `none`), `state`
(state uuid), `labels` (array of label ids), `parent` (issue uuid).

Returns the created issue object.

### `plane_issue_list`

GET `/projects/{project_id}/issues/`.

Required: `project_id`.
Optional: `per_page` (number), `cursor` (string), `archived` (boolean),
`state_group` (one of `backlog`, `unstarted`, `started`, `completed`,
`cancelled`), `assignees` (array of plane user ids), `labels` (array of label
ids), `priority`, `created_at`, `target_date`, `completed_at`, `name`,
`order_by`, `expand`, `parent` (strings).

Date filters use plane's `<value>;<op>` syntax — e.g.
`"2026-04-01;after,2026-04-30;before"`.
Comma-separated strings are also accepted for `assignees` and `labels`.

Returns `{ "results": [...], "count": n, "next_cursor": "..." }`.

### `plane_issue_get`

GET `/projects/{project_id}/issues/{issue_id}/`.

Required: `project_id`, `issue_id`.

### `plane_issue_get_by_identifier`

GET `/issues/{identifier}/`.

Required: `identifier` (e.g. `PROJ-7`).

### `plane_issue_update`

PATCH `/projects/{project_id}/issues/{issue_id}/`.

Required: `project_id`, `issue_id`. All other fields optional; only those
provided are sent (the client uses pointer fields to preserve the
absent-vs-empty distinction).

### `plane_issue_delete`

DELETE `/projects/{project_id}/issues/{issue_id}/`.

Required: `project_id`, `issue_id`. Returns `{ "deleted": true, "issue_id": "..." }`.

## States

### `plane_state_list`

GET `/projects/{project_id}/states/`.

Required: `project_id`. Returns `{ "results": [{id, name, group, color}, ...] }`.

## Labels

### `plane_label_list`

GET `/projects/{project_id}/labels/`.

Required: `project_id`. Returns `{ "results": [{id, name, color}, ...] }`.

## Comments

### `plane_comment_list`

GET `/projects/{project_id}/issues/{issue_id}/comments/`.

Required: `project_id`, `issue_id`.

### `plane_comment_add`

POST `/projects/{project_id}/issues/{issue_id}/comments/`.

Required: `project_id`, `issue_id`, `comment_html`.
Optional: `access` (`INTERNAL` or `EXTERNAL`, default `INTERNAL`).

### `plane_comment_update`

PATCH `/projects/{project_id}/issues/{issue_id}/comments/{comment_id}/`.

Required: `project_id`, `issue_id`, `comment_id`, `comment_html`.
Optional: `access` (default `INTERNAL`).

### `plane_comment_delete`

DELETE `/projects/{project_id}/issues/{issue_id}/comments/{comment_id}/`.

Required: `project_id`, `issue_id`, `comment_id`.

## Activities

### `plane_activity_list`

GET `/projects/{project_id}/issues/{issue_id}/activities/`.

Required: `project_id`, `issue_id`. Read-only audit log.

## Attachments

### `plane_attachment_list`

GET `/projects/{project_id}/issues/{issue_id}/issue-attachments/`.

Required: `project_id`, `issue_id`.

### `plane_attachment_init_upload`

POST `/projects/{project_id}/issues/{issue_id}/issue-attachments/`.

Low-level: returns `{ id, upload_data: { url, fields: {...} } }`. Use
`plane_attachment_upload` for the full init → s3 → complete flow.

Required: `project_id`, `issue_id`, `name`.
Optional: `file_type` (mime type), `size` (number, bytes).

### `plane_attachment_complete_upload`

PATCH `/projects/{project_id}/issues/{issue_id}/issue-attachments/{asset_id}/`.

Required: `project_id`, `issue_id`, `asset_id`.

### `plane_attachment_upload`

High-level: takes an absolute `file_path` on the host running the mcp
server, guesses the mime type, requests s3 credentials, posts the multipart
form, and finalises.

Required: `project_id`, `issue_id`, `file_path`.
Optional: `display_name` (default = basename of `file_path`).

Returns `{ asset_id, name, size, result }`.

### `plane_attachment_read`

Lists attachments to find one by id, then downloads it. Behaviour depends on
file extension:

- text (`.txt`, `.md`) → `{ id, name, kind: "text", size, mime_type, content: "..." }`.
- image (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.bmp`) →
  `{ id, name, kind: "image", size, mime_type, data_base64: "..." }`.
- anything else → metadata only.

Required: `project_id`, `issue_id`, `attachment_id`.

## Higher-level

### `plane_issue_transfer`

Cross-project copy. Loads the source issue, translates labels and state by
name (case-insensitive), creates a new issue in the target project, then
adds cross-link comments on both sides.

Required: `source_project_id`, `source_issue_id`, `source_project_code`,
`target_project_id`, `target_project_code`.
Optional: `close_source` (boolean; defaults to false) — moves the source
issue to a state in plane's `completed` group when true.

Returns:

```json
{
  "source_issue": {...},
  "target_issue": {...},
  "target_project_code": "TGT",
  "source_closed": false,
  "comment_failures": [],
  "skipped_labels": ["label-not-in-target"],
  "state_fallback_used": false
}
```

Project codes are required so cross-link comments use readable identifiers.
Use `plane_project_list` to discover project ids and identifiers.

### `plane_issue_workpad_upsert`

Renders the workpad as HTML and either adds or updates a single "workpad"
comment on the issue. Detection is done by searching for the marker text
`"MCP Workpad"` in `comment_html` / `comment_text` / `comment_stripped` /
`comment`.

Required: `project_id`, `issue_id`, `summary`.
Optional: `delegation_status`, `repo_key`, `repo_path`, `publish_remote`,
`branch`, `pr_url`, `validation` (array of strings), `blockers` (array of
strings).

Returns `{ status: "workpad_upserted", comment_id, issue }`.

## Sanity

### `plane_workspace_info`

Returns `{ base_url, workspace }`. Never echoes the api token.

### `plane_health`

Required: `project_id`. Probes the states and issues endpoints for that
project and returns `{ ok, checks, warnings }` diagnostics. Empty states or
endpoint failures produce `ok: false`.
