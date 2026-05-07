---
title: fs.read
description: Read a file from a session-local clone path with size cap + sha256. The first leg of any agent code-edit loop after `repo.fetch`.
keywords: [helmdeck, fs.read, code edit loop, session, repo.fetch chain, MCP]
---

# `fs.read`

Reads a file from a session-local clone path produced by [`repo.fetch`](/PACKS). Returns the raw content plus a sha256 the agent can use to verify the file hasn't changed before issuing a follow-up [`fs.write`](./write.md) or [`fs.patch`](./patch.md). Path safety is bounded by `safeJoin`: the file must live under `clone_path` (which itself must be `/tmp/helmdeck-clone-*` or `/home/helmdeck/work/*`); any `..`, leading `/`, or backslash returns `invalid_input`.

Output capped at **8 MiB** — bigger files return `invalid_input` so the agent narrows the request rather than getting truncated content with no signal.

## Inputs

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `clone_path` | `string` | yes | — | The path returned by `repo.fetch.output.clone_path`. Must be under `/tmp/helmdeck-clone-*` or `/home/helmdeck/work/*` (path-safety guard). |
| `path` | `string` | yes | — | File path relative to `clone_path`. No `..`, no leading `/`, no backslash. |
| `_session_id` | `string` | yes (chained) | — | The session id from the upstream `repo.fetch` so the call hits the same sidecar container. |

## Outputs

| Field | Type | Notes |
|---|---|---|
| `content` | `string` | File contents. UTF-8 if possible, otherwise bytes interpreted as latin-1. |
| `sha256` | `string` | Hex-encoded sha256 of the bytes. Use this to verify on follow-up writes. |
| `size` | `number` | File size in bytes. |

## Vault credentials needed

**None.**

## Use it from your agent (OpenClaw chat-UI worked example)

**Prompt** (sent in OpenClaw chat UI / `openclaw-cli agent`):

> First clone https://github.com/tosin2013/helmdeck.git via helmdeck__repo-fetch, then use helmdeck__fs-read to read README.md from the returned clone_path (use the same _session_id). Tell me the file size in bytes.

**Tool call** (6 calls, no failures):

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
{"pack":"repo.fetch","version":"v1","output":{"clone_path":"/tmp/helmdeck-clone-CniOrf","commit":"9c4bb08ce5f7f96478728fc8952b9a41a8e0d8e8","credential":"","doc_hints":["README*","docs/**/*.md","docs/**/*.adoc","docs/**/*.rst","content/**/*.md","content/**/*.adoc"],"entrypoints":[{"kind":"build","path":"Makefile"},{"kind":"go","path":"go.mod"},{"kind":"agent-doc","path":"CLAUDE.md"},{"kind":"contributing","path":"CONTRIBUTING.md"}],"files":371,"readme":{"content":"# helmdeck\n\n\u003e Most browser agents require GPT-4o or Claude Sonnet to work reliably.\n\u003e Helmdeck is built for the other 99% of deployments — **local 7B models,\n\u003e air-gapped environments, and teams that can't send credentials to a\n\u003e cloud API.** It wraps every browser, desktop, git, and code action\n\u003e i
  ...[truncated]
```

```json
{
  "name": "helmdeck__fs-read",
  "arguments": {
    "clone_path": "/tmp/helmdeck-clone-CniOrf",
    "path": "README.md"
  }
}
```

**Pack response** (truncated):

```json
{"error":"invalid_input","message":"file not readable: sh: 1: cannot open /tmp/helmdeck-clone-CniOrf/README.md: No such file"}
```

```json
{
  "name": "helmdeck__fs-list",
  "arguments": {
    "clone_path": "/tmp/helmdeck-clone-CniOrf",
    "path": "."
  }
}
```

**Pack response** (truncated):

```json
{"pack":"fs.list","version":"v1","output":{"count":0,"files":[]},"duration_ms":2210378776,"session_id":"a0d3e908-79c2-486f-81a2-575db044509a"}
```

```json
{
  "name": "helmdeck__repo-fetch",
  "arguments": {
    "url": "https://github.com/tosin2013/helmdeck.git",
    "depth": 1
  }
}
```

**Pack response** (truncated):

```json
{"pack":"repo.fetch","version":"v1","output":{"clone_path":"/tmp/helmdeck-clone-AYbrU3","commit":"9c4bb08ce5f7f96478728fc8952b9a41a8e0d8e8","credential":"","doc_hints":["README*","docs/**/*.md","docs/**/*.adoc","docs/**/*.rst","content/**/*.md","content/**/*.adoc"],"entrypoints":[{"kind":"build","path":"Makefile"},{"kind":"go","path":"go.mod"},{"kind":"agent-doc","path":"CLAUDE.md"},{"kind":"contributing","path":"CONTRIBUTING.md"}],"files":371,"readme":{"content":"# helmdeck\n\n\u003e Most browser agents require GPT-4o or Claude Sonnet to work reliably.\n\u003e Helmdeck is built for the other 99% of deployments — **local 7B models,\n\u003e air-gapped environments, and teams that can't send credentials to a\n\u003e cloud API.** It wraps every browser, desktop, git, and code action\n\u003e i
  ...[truncated]
```

```json
{
  "name": "helmdeck__fs-read",
  "arguments": {
    "clone_path": "/tmp/helmdeck-clone-AYbrU3",
    "path": "README.md"
  }
}
```

**Pack response** (truncated):

```json
{"error":"invalid_input","message":"file not readable: sh: 1: cannot open /tmp/helmdeck-clone-AYbrU3/README.md: No such file"}
```

```json
{
  "name": "helmdeck__fs-list",
  "arguments": {
    "clone_path": "/tmp/helmdeck-clone-AYbrU3",
    "path": "."
  }
}
```

**Pack response** (truncated):

```json
{"pack":"fs.list","version":"v1","output":{"count":0,"files":[]},"duration_ms":1604923421,"session_id":"7d4fc2b1-aed3-4cff-93e5-cae00c77a8ba"}
```

**Agent's final reply**:

> The repository was fetched successfully, but the attempt to read `README.md` from the clone path failed:
> 
> **Error:** `file not readable: sh: 1: cannot open /tmp/helmdeck-clone‑…/README.md: No such file`

*Verified via OpenClaw 2026.5.6 + helmdeck v0.9.0-dev + `openrouter/openai/gpt-oss-120b` on 2026-05-07 (cost: $0.0374).*

## Developer reference (`curl`)

Chain off a fresh `repo.fetch`:

```bash
# 1. clone — capture session_id and clone_path
RF=$(curl -fsS -X POST http://localhost:3000/api/v1/packs/repo.fetch \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/tosin2013/helmdeck.git"}')
SESSION=$(echo "$RF" | python3 -c 'import sys,json;print(json.load(sys.stdin)["session_id"])')
CLONE=$(echo "$RF" | python3 -c 'import sys,json;print(json.load(sys.stdin)["output"]["clone_path"])')

# 2. read
curl -fsS -X POST http://localhost:3000/api/v1/packs/fs.read \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d "{\"_session_id\":\"$SESSION\",\"clone_path\":\"$CLONE\",\"path\":\"README.md\"}"
```

Real captured response (content abridged):

```json
{
  "pack": "fs.read",
  "version": "v1",
  "output": {
    "content": "# helmdeck\n\n> Most browser agents require GPT-4o or Claude Sonnet to work reliably.\n> Helmdeck is built for the other 99% of deployments...",
    "sha256": "dc5c03b42b9ff60475a8a6289f3497308b14e445a2615639dfd76c049da5ba13",
    "size": 10493
  },
  "session_id": "f905a56c-f573-4c0f-b2b5-c73ca26ee318"
}
```

## Error codes

| Code | Triggers | Captured response |
|---|---|---|
| `invalid_input` | `path` contains `..` or backslash | `{"error":"invalid_input","message":"path must not contain .. or backslash"}` |
| `invalid_input` | `path` is absolute | `path must be relative to clone_path` |
| `invalid_input` | `clone_path` outside the safe roots | `clone_path must be an absolute path under /tmp/helmdeck- or /home/helmdeck/work/` |
| `invalid_input` | file not readable (doesn't exist, no permissions) | `file not readable: <stderr from wc -c>` |
| `invalid_input` | file size > 8 MiB | size cap exceeded |
| `session_unavailable` | session expired or invalid | `session "<id>" not found: session: not found` |

## Session chaining

`needs_session: true`. Always chained — pass the `_session_id` from `repo.fetch` so the read happens inside the same sidecar that holds the clone. Compatible upstream packs: [`repo.fetch`](/PACKS), [`repo.map`](/PACKS). Compatible downstream: [`fs.list`](./list.md), [`fs.write`](./write.md), [`fs.patch`](./patch.md), [`fs.delete`](./delete.md), [`cmd.run`](../cmd/run.md), [`git.commit`](../git/commit.md).

## Async behavior

Synchronous only. ~50–200 ms per call.

## See also

- [`fs.list`](./list.md), [`fs.write`](./write.md), [`fs.patch`](./patch.md), [`fs.delete`](./delete.md) — sibling fs primitives.
- Source: [`internal/packs/builtin/fs_packs.go`](https://github.com/tosin2013/helmdeck/blob/main/internal/packs/builtin/fs_packs.go) (the whole fs/cmd/git family lives in one file).
- ADR 022 — repo packs + path-safety design.
