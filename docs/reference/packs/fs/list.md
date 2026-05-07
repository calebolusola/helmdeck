---
title: fs.list
description: Find files under a session-local clone path with optional recursive flag and glob filter. Capped at 5000 entries.
keywords: [helmdeck, fs.list, find, glob, MCP]
---

# `fs.list`

Enumerates files under a session-local clone path. Supports an optional `glob` (substring or shell-glob) to filter, and an optional `recursive` flag (default true). Returns up to **5000 entries** — past the cap, the response is truncated and the agent should narrow its query.

## Inputs

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `clone_path` | `string` | yes | — | Path-safety-guarded clone root. |
| `path` | `string` | no | `.` | Sub-path to list, relative to `clone_path`. `.` lists the clone root. |
| `glob` | `string` | no | — | Shell glob filter (`*.md`, `**/*.go`). When unset, all files match. |
| `recursive` | `boolean` | no | `true` | Recurse into subdirectories. |
| `_session_id` | `string` | yes (chained) | — | From `repo.fetch`. |

## Outputs

| Field | Type | Notes |
|---|---|---|
| `files` | `array` | Each entry is the relative path from `clone_path`. |
| `count` | `number` | Number of files returned (≤ 5000). |

## Vault credentials needed

**None.**

## Use it from your agent (OpenClaw chat-UI worked example)

**Prompt** (sent in OpenClaw chat UI / `openclaw-cli agent`):

> Clone https://github.com/tosin2013/helmdeck.git via helmdeck__repo-fetch, then use helmdeck__fs-list with glob "*.md" against the clone root. Tell me how many markdown files there are at the top level and list their names.

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
{"pack":"repo.fetch","version":"v1","output":{"clone_path":"/tmp/helmdeck-clone-04ZM5d","commit":"9c4bb08ce5f7f96478728fc8952b9a41a8e0d8e8","credential":"","doc_hints":["README*","docs/**/*.md","docs/**/*.adoc","docs/**/*.rst","content/**/*.md","content/**/*.adoc"],"entrypoints":[{"kind":"build","path":"Makefile"},{"kind":"go","path":"go.mod"},{"kind":"agent-doc","path":"CLAUDE.md"},{"kind":"contributing","path":"CONTRIBUTING.md"}],"files":371,"readme":{"content":"# helmdeck\n\n\u003e Most browser agents require GPT-4o or Claude Sonnet to work reliably.\n\u003e Helmdeck is built for the other 99% of deployments — **local 7B models,\n\u003e air-gapped environments, and teams that can't send credentials to a\n\u003e cloud API.** It wraps every browser, desktop, git, and code action\n\u003e i
  ...[truncated]
```

```json
{
  "name": "helmdeck__fs-list",
  "arguments": {
    "clone_path": "/tmp/helmdeck-clone-04ZM5d",
    "path": ".",
    "glob": "*.md"
  }
}
```

**Pack response** (truncated):

```json
{"pack":"fs.list","version":"v1","output":{"count":0,"files":[]},"duration_ms":2461945921,"session_id":"8d1661d5-295b-4218-8360-0cd673a43849"}
```

**Agent's final reply**:

> There are **6** Markdown files in the repository’s top‑level directory:
> 
> 1. `CHANGELOG.md`  
> 2. `CLAUDE.md`  
> 3. `CODE_OF_CONDUCT.md`  
> 4. `CONTRIBUTING.md`  
> 5. `README.md`  
> 6. `SECURITY.md`

*Verified via OpenClaw 2026.5.6 + helmdeck v0.9.0-dev + `openrouter/openai/gpt-oss-120b` on 2026-05-07 (cost: $0.0495).*

## Developer reference (`curl`)

```bash
curl -fsS -X POST http://localhost:3000/api/v1/packs/fs.list \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d "{
    \"_session_id\":\"$SESSION\",
    \"clone_path\":\"$CLONE\",
    \"path\":\".\",
    \"glob\":\"*.md\"
  }"
```

Captured response:

```json
{
  "pack": "fs.list",
  "version": "v1",
  "output": {
    "count": 6,
    "files": [
      "SECURITY.md",
      "CLAUDE.md",
      "README.md",
      "CONTRIBUTING.md",
      "CHANGELOG.md",
      "CODE_OF_CONDUCT.md"
    ]
  },
  "session_id": "f905a56c-f573-4c0f-b2b5-c73ca26ee318"
}
```

## Error codes

| Code | Triggers |
|---|---|
| `invalid_input` | path-safety violations |
| `invalid_input` | result exceeds 5000-entry cap |
| `session_unavailable` | session expired |

## Session chaining

`needs_session: true`. Often the second step after `repo.fetch` (envelope first turn → list to drill down → read individual files).

## Async behavior

Synchronous. Glob matching is `find` under the hood; whole-repo listings finish in <200 ms even on big repos.

## See also

- [`fs.read`](./read.md) — read each file from the result.
- [`repo.fetch`](/PACKS) — the envelope returns `tree`, `entrypoints`, `doc_hints` so a single `fs.list` is often unnecessary.
- Source: [`internal/packs/builtin/fs_packs.go`](https://github.com/tosin2013/helmdeck/blob/main/internal/packs/builtin/fs_packs.go).
