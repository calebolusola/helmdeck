---
title: fs.delete
description: Delete a single file in a session-local clone path. Always-pair-with-git-commit recommendation; the path-safety guard refuses anything outside the clone root.
keywords: [helmdeck, fs.delete, MCP, code edit loop]
---

# `fs.delete`

Removes a single file inside a session-local clone path. Same path-safety guards as the rest of the `fs.*` family — `clone_path` rooted under `/tmp/helmdeck-clone-*` or `/home/helmdeck/work/*`, no `..`, no absolute paths.

The pack returns `{"deleted": true, "path": "..."}` on success. **Pair with [`git.commit`](../git/commit.md) immediately after** so the deletion is captured; otherwise an agent that crashes mid-loop leaves the clone in a half-modified state that the next session won't see (each session is ephemeral).

## Inputs

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `clone_path` | `string` | yes | — | Path-safety-guarded. |
| `path` | `string` | yes | — | Relative file path. Directories: not supported (use `cmd.run` with `rm -r` if you need a tree delete). |
| `_session_id` | `string` | yes (chained) | — | From `repo.fetch`. |

## Outputs

| Field | Type | Notes |
|---|---|---|
| `deleted` | `boolean` | Always `true` on success. |
| `path` | `string` | Echo of the relative path that was removed. |

## Vault credentials needed

**None.**

## Use it from your agent (OpenClaw chat-UI worked example)

**Prompt** (sent in OpenClaw chat UI / `openclaw-cli agent`):

> Clone https://github.com/tosin2013/helmdeck.git via helmdeck__repo-fetch, write a temp file named demo-delete.md with helmdeck__fs-write, then immediately remove it with helmdeck__fs-delete. Confirm the deletion succeeded.

**Tool call** (3 calls, no failures):

```json
{
  "name": "helmdeck__repo-fetch",
  "arguments": {
    "url": "https://github.com/tosin2013/helmdeck.git"
  }
}
```

**Pack response** (truncated):

```json
{"pack":"repo.fetch","version":"v1","output":{"clone_path":"/tmp/helmdeck-clone-WPLjuP","commit":"9c4bb08ce5f7f96478728fc8952b9a41a8e0d8e8","credential":"","doc_hints":["README*","docs/**/*.md","docs/**/*.adoc","docs/**/*.rst","content/**/*.md","content/**/*.adoc"],"entrypoints":[{"kind":"build","path":"Makefile"},{"kind":"go","path":"go.mod"},{"kind":"agent-doc","path":"CLAUDE.md"},{"kind":"contributing","path":"CONTRIBUTING.md"}],"files":371,"readme":{"content":"# helmdeck\n\n\u003e Most browser agents require GPT-4o or Claude Sonnet to work reliably.\n\u003e Helmdeck is built for the other 99% of deployments — **local 7B models,\n\u003e air-gapped environments, and teams that can't send credentials to a\n\u003e cloud API.** It wraps every browser, desktop, git, and code action\n\u003e i
  ...[truncated]
```

```json
{
  "name": "helmdeck__fs-write",
  "arguments": {
    "clone_path": "/tmp/helmdeck-clone-WPLjuP",
    "path": "demo-delete.md",
    "content": "temp delete file"
  }
}
```

**Pack response** (truncated):

```json
{"pack":"fs.write","version":"v1","output":{"sha256":"25cf925e9fe44bdf62c40c654cf2ae6db6746f012ae512e763bc70df0dc70642","size":16},"duration_ms":2695705116,"session_id":"4c8bbbc6-d681-48c3-8e14-2a4d3c6ee2ff"}
```

```json
{
  "name": "helmdeck__fs-delete",
  "arguments": {
    "clone_path": "/tmp/helmdeck-clone-WPLjuP",
    "path": "demo-delete.md"
  }
}
```

**Pack response** (truncated):

```json
{"pack":"fs.delete","version":"v1","output":{"deleted":true,"path":"demo-delete.md"},"duration_ms":2321804027,"session_id":"0c099a57-e75e-4498-b08c-10bc675a8128"}
```

**Agent's final reply**:

> The file **demo-delete.md** was successfully removed:
> 
> - **Deleted:** true  
> - **Path:** `demo-delete.md`  
> 
> Deletion confirmed.

*Verified via OpenClaw 2026.5.6 + helmdeck v0.9.0-dev + `openrouter/openai/gpt-oss-120b` on 2026-05-07 (cost: $0.0803).*

## Developer reference (`curl`)

```bash
curl -fsS -X POST http://localhost:3000/api/v1/packs/fs.delete \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d "{
    \"_session_id\":\"$SESSION\",
    \"clone_path\":\"$CLONE\",
    \"path\":\"docs-test-tmp.md\"
  }"
```

Captured response:

```json
{
  "pack": "fs.delete",
  "version": "v1",
  "output": {
    "deleted": true,
    "path": "docs-test-tmp.md"
  },
  "session_id": "f905a56c-f573-4c0f-b2b5-c73ca26ee318"
}
```

## Error codes

| Code | Triggers |
|---|---|
| `invalid_input` | path-safety violations |
| `invalid_input` | file doesn't exist |
| `invalid_input` | path is a directory (use `cmd.run` for tree deletes) |
| `session_unavailable` | session expired |

## Session chaining

`needs_session: true`. Almost always paired with `git.commit` immediately after.

## Async behavior

Synchronous. Sub-50ms.

## See also

- [`git.commit`](../git/commit.md) — capture the deletion.
- [`cmd.run`](../cmd/run.md) — for directory deletes (`rm -rf`) or globs.
- [`fs.list`](./list.md) — find files before deleting.
- Source: [`internal/packs/builtin/fs_packs.go`](https://github.com/tosin2013/helmdeck/blob/main/internal/packs/builtin/fs_packs.go).
