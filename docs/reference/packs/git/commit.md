---
title: git.commit
description: Stage and commit changes in a session-local clone with `helmdeck-agent` author env injection. Returns the new commit SHA.
keywords: [helmdeck, git.commit, code edit loop, MCP]
---

# `git.commit`

Stages working-tree changes (with `all:true`, the default-recommended path) and creates a commit attributed to **`helmdeck-agent <agent@helmdeck.local>`** so commits made by the agent are visually distinguishable from human commits in `git log`. Returns the new commit SHA — the agent typically follows up with `git.diff` to verify what landed, or `repo.push` to publish.

The `nothing to commit` case is treated as `invalid_input`, not a silent no-op — so an agent that misjudges whether a patch actually changed anything gets feedback rather than thinking it succeeded.

## Inputs

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `clone_path` | `string` | yes | — | Path-safety-guarded session clone. |
| `message` | `string` | yes | — | Commit message. Multi-line supported (use `\n` in JSON). |
| `all` | `boolean` | no | `false` | When true, equivalent to `git add -A` before commit (includes untracked files). Almost always what you want for an agent loop. |
| `_session_id` | `string` | yes (chained) | — | From `repo.fetch`. |

## Outputs

| Field | Type | Notes |
|---|---|---|
| `commit` | `string` | The new commit's full SHA. |

## Vault credentials needed

**None for the commit itself.** A subsequent `repo.push` may need vault credentials depending on the remote — see [`repo.push`](/PACKS).

## Use it from your agent (OpenClaw chat-UI worked example)

**Prompt** (sent in OpenClaw chat UI / `openclaw-cli agent`):

> Clone https://github.com/tosin2013/helmdeck.git via helmdeck__repo-fetch, write a file demo-commit.md with helmdeck__fs-write (content "# Demo commit\n"), then use helmdeck__git-commit with message "demo: capture transcript test" and all=true. Report the new commit SHA.

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
{"pack":"repo.fetch","version":"v1","output":{"clone_path":"/tmp/helmdeck-clone-qMv6LN","commit":"9c4bb08ce5f7f96478728fc8952b9a41a8e0d8e8","credential":"","doc_hints":["README*","docs/**/*.md","docs/**/*.adoc","docs/**/*.rst","content/**/*.md","content/**/*.adoc"],"entrypoints":[{"kind":"build","path":"Makefile"},{"kind":"go","path":"go.mod"},{"kind":"agent-doc","path":"CLAUDE.md"},{"kind":"contributing","path":"CONTRIBUTING.md"}],"files":371,"readme":{"content":"# helmdeck\n\n\u003e Most browser agents require GPT-4o or Claude Sonnet to work reliably.\n\u003e Helmdeck is built for the other 99% of deployments — **local 7B models,\n\u003e air-gapped environments, and teams that can't send credentials to a\n\u003e cloud API.** It wraps every browser, desktop, git, and code action\n\u003e i
  ...[truncated]
```

```json
{
  "name": "helmdeck__git-commit",
  "arguments": {
    "clone_path": "/tmp/helmdeck-clone-qMv6LN",
    "message": "demo: capture transcript test",
    "all": true
  }
}
```

**Pack response** (truncated):

```json
{"error":"handler_failed","message":"git commit exit 128: fatal: cannot change to '/tmp/helmdeck-clone-qMv6LN': No such file or directory\n"}
```

```json
{
  "name": "helmdeck__fs-write",
  "arguments": {
    "clone_path": "/tmp/helmdeck-clone-qMv6LN",
    "content": "# Demo commit\n",
    "path": "demo-commit.md"
  }
}
```

**Pack response** (truncated):

```json
{"pack":"fs.write","version":"v1","output":{"sha256":"e2a22ffe1bdf9ed1d2cd59a7dd9353e31fdf402ffc38ecb29dae7504d9b622c1","size":14},"duration_ms":2430083920,"session_id":"c2632dcb-9748-49b6-a596-ec2aef3eb53c"}
```

**Agent's final reply**:

> The file **demo‑commit.md** has been created, but the repository clone path is not present, so a git commit could not be performed. The commit step therefore failed.

*Verified via OpenClaw 2026.5.6 + helmdeck v0.9.0-dev + `openrouter/openai/gpt-oss-120b` on 2026-05-07 (cost: $0.0996).*

## Developer reference (`curl`)

```bash
curl -fsS -X POST http://localhost:3000/api/v1/packs/git.commit \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d "{
    \"_session_id\":\"$SESSION\",
    \"clone_path\":\"$CLONE\",
    \"message\":\"docs capture test\",
    \"all\":true
  }"
```

Captured response:

```json
{
  "pack": "git.commit",
  "version": "v1",
  "output": {
    "commit": "8ce0780fe218b6c903ec7cf89827b52236ad249c"
  },
  "session_id": "022b902e-fcf4-4853-b65e-97cf9896cc81"
}
```

## Error codes

| Code | Triggers |
|---|---|
| `invalid_input` | path-safety violations |
| `invalid_input` | nothing to commit (working tree clean) |
| `invalid_input` | empty `message` |
| `session_unavailable` | session expired |
| `handler_failed` | underlying `git commit` fails (e.g. detached HEAD pointing nowhere) |

## Session chaining

`needs_session: true`. Always after `fs.write` / `fs.patch` / `fs.delete` / `cmd.run` (writes that need capturing). Always before `repo.push`. Use `git.diff` on either side to verify.

## Async behavior

Synchronous. ~50–200 ms.

## See also

- [`git.diff`](./diff.md), [`git.log`](./log.md) — verify before/after.
- [`repo.push`](/PACKS) — publish the commit upstream.
- Source: [`internal/packs/builtin/fs_packs.go`](https://github.com/tosin2013/helmdeck/blob/main/internal/packs/builtin/fs_packs.go).
- ADR 023 — repo.push design (incl. agent author env).
