---
title: git.diff
description: Show the diff of changes in a session-local clone. Empty when working tree is clean. Untracked files don't appear — use `cmd.run git status` if you need them.
keywords: [helmdeck, git.diff, code edit loop, MCP]
---

# `git.diff`

Returns `git diff` output for the working tree of a session-local clone. The diff covers **modified tracked files** by default — untracked files don't appear (a quirk of `git diff`'s default behavior, not helmdeck's; use `cmd.run` with `git status` if the agent needs to see untracked files too).

Useful as the verify-before-commit step in the code-edit loop: `fs.read` → `fs.patch` → `git.diff` → if reasonable, `git.commit`.

## Inputs

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `clone_path` | `string` | yes | — | Path-safety-guarded session clone. |
| `path` | `string` | no | — | Limit the diff to a single file or directory. Relative to `clone_path`. |
| `staged` | `boolean` | no | `false` | When true, runs `git diff --cached` (shows what's staged for the next commit instead of the working-tree changes). |
| `_session_id` | `string` | yes (chained) | — | From `repo.fetch`. |

## Outputs

| Field | Type | Notes |
|---|---|---|
| `diff` | `string` | The unified-diff output. Empty when no changes. |
| `files_changed` | `number` | Count of files with changes in the diff. |

## Vault credentials needed

**None.**

## Use it from your agent (OpenClaw chat-UI worked example)

**Prompt** (sent in OpenClaw chat UI / `openclaw-cli agent`):

> Clone https://github.com/tosin2013/helmdeck.git via helmdeck__repo-fetch, then use helmdeck__git-diff against the clean working tree. Report files_changed (should be 0).

**Tool call** (2 calls, no failures):

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
{"pack":"repo.fetch","version":"v1","output":{"clone_path":"/tmp/helmdeck-clone-9mPeEZ","commit":"9c4bb08ce5f7f96478728fc8952b9a41a8e0d8e8","credential":"","doc_hints":["README*","docs/**/*.md","docs/**/*.adoc","docs/**/*.rst","content/**/*.md","content/**/*.adoc"],"entrypoints":[{"kind":"build","path":"Makefile"},{"kind":"go","path":"go.mod"},{"kind":"agent-doc","path":"CLAUDE.md"},{"kind":"contributing","path":"CONTRIBUTING.md"}],"files":371,"readme":{"content":"# helmdeck\n\n\u003e Most browser agents require GPT-4o or Claude Sonnet to work reliably.\n\u003e Helmdeck is built for the other 99% of deployments — **local 7B models,\n\u003e air-gapped environments, and teams that can't send credentials to a\n\u003e cloud API.** It wraps every browser, desktop, git, and code action\n\u003e i
  ...[truncated]
```

```json
{
  "name": "helmdeck__git-diff",
  "arguments": {
    "clone_path": "/tmp/helmdeck-clone-9mPeEZ",
    "staged": false
  }
}
```

**Pack response** (truncated):

```json
{"pack":"git.diff","version":"v1","output":{"diff":"","files_changed":0},"duration_ms":2514513803,"session_id":"4657bd1f-4a31-45ef-8bb8-4536379e8500"}
```

**Agent's final reply**:

> **git‑diff result**
> 
> - **files_changed:** 0  
> - **diff:** *(empty)*  

*Verified via OpenClaw 2026.5.6 + helmdeck v0.9.0-dev + `openrouter/openai/gpt-oss-120b` on 2026-05-07 (cost: $0.1086).*

## Developer reference (`curl`)

```bash
curl -fsS -X POST http://localhost:3000/api/v1/packs/git.diff \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d "{\"_session_id\":\"$SESSION\",\"clone_path\":\"$CLONE\"}"
```

Captured response on a clean working tree:

```json
{
  "pack": "git.diff",
  "version": "v1",
  "output": {
    "diff": "",
    "files_changed": 0
  },
  "session_id": "022b902e-fcf4-4853-b65e-97cf9896cc81"
}
```

After modifying a tracked file, the response would include the unified-diff content under `diff`.

## Error codes

| Code | Triggers |
|---|---|
| `invalid_input` | path-safety violations |
| `session_unavailable` | session expired |
| `handler_failed` | `git diff` itself errors (e.g. clone is corrupt) |

## Session chaining

`needs_session: true`. Always between an fs change and `git.commit`.

## Async behavior

Synchronous. Sub-200 ms.

## See also

- [`git.commit`](./commit.md), [`git.log`](./log.md).
- [`cmd.run`](../cmd/run.md) — when you need `git status` (which DOES show untracked files) instead of `git diff`.
- Source: [`internal/packs/builtin/fs_packs.go`](https://github.com/tosin2013/helmdeck/blob/main/internal/packs/builtin/fs_packs.go).
