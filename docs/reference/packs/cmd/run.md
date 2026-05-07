---
title: cmd.run
description: Run an arbitrary shell command inside a session-local clone path. Non-zero exits are normal pack outcomes, not pack errors. Output capped at 8 MiB combined stdout+stderr.
keywords: [helmdeck, cmd.run, shell, exec, code edit loop, MCP]
---

# `cmd.run`

Runs an arbitrary command inside the sidecar session that holds a clone path. The command is given as an `["argv", "form"]` array — **not a string** — so there's no shell-quoting ambiguity. Non-zero exit codes are returned as data (`exit_code`), not as pack errors. The combined stdout+stderr is capped at **8 MiB** to keep agent context windows bounded.

This is the workhorse pack for the Phase 5.5 code-edit loop: build, test, lint, run a script, anything the LLM wants to verify before `git.commit`.

## Inputs

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `clone_path` | `string` | yes | — | Path-safety-guarded session-local clone. |
| `command` | `array` | yes | — | argv-style: `["go", "test", "./..."]`. Pass through `sh -c` explicitly if you need shell features: `["sh","-c","echo $PATH | grep go"]`. |
| `stdin` | `string` | no | — | Stdin bytes if the command reads from it. |
| `_session_id` | `string` | yes (chained) | — | From `repo.fetch`. |

## Outputs

| Field | Type | Notes |
|---|---|---|
| `stdout` | `string` | Captured stdout, UTF-8. Truncated at 8 MiB combined. |
| `stderr` | `string` | Captured stderr. |
| `exit_code` | `number` | The command's exit code. **Not** an error — agents inspect this and decide what to do. |

## Vault credentials needed

**None directly.** If the command needs an auth token (e.g. `gh api ...`), set up a vault credential and reference it via the `${vault:NAME}` placeholder pattern in the command (the same resolver `http.fetch` uses). For most agent code-edit work, no credential is needed.

## Use it from your agent (OpenClaw chat-UI worked example)

**Prompt** (sent in OpenClaw chat UI / `openclaw-cli agent`):

> Clone https://github.com/tosin2013/helmdeck.git via helmdeck__repo-fetch, then use helmdeck__cmd-run with command ["ls","docs/reference/packs"] to list the per-pack docs. Report the entries from stdout.

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
{"pack":"repo.fetch","version":"v1","output":{"clone_path":"/tmp/helmdeck-clone-qpNMUG","commit":"9c4bb08ce5f7f96478728fc8952b9a41a8e0d8e8","credential":"","doc_hints":["README*","docs/**/*.md","docs/**/*.adoc","docs/**/*.rst","content/**/*.md","content/**/*.adoc"],"entrypoints":[{"kind":"build","path":"Makefile"},{"kind":"go","path":"go.mod"},{"kind":"agent-doc","path":"CLAUDE.md"},{"kind":"contributing","path":"CONTRIBUTING.md"}],"files":371,"readme":{"content":"# helmdeck\n\n\u003e Most browser agents require GPT-4o or Claude Sonnet to work reliably.\n\u003e Helmdeck is built for the other 99% of deployments — **local 7B models,\n\u003e air-gapped environments, and teams that can't send credentials to a\n\u003e cloud API.** It wraps every browser, desktop, git, and code action\n\u003e i
  ...[truncated]
```

```json
{
  "name": "helmdeck__cmd-run",
  "arguments": {
    "clone_path": "/tmp/helmdeck-clone-qpNMUG",
    "command": [
      "ls",
      "docs/reference/packs"
    ]
  }
}
```

**Pack response** (truncated):

```json
{"pack":"cmd.run","version":"v1","output":{"exit_code":2,"stderr":"sh: 1: cd: can't cd to /tmp/helmdeck-clone-qpNMUG\n","stdout":""},"duration_ms":2058628272,"session_id":"63105ea4-8eb7-41ae-870d-0505338f810e"}
```

**Agent's final reply**:

> The command failed because the working directory wasn’t changed to the cloned repository. The error output was:
> 
> ```
> sh: 1: cd: can't cd to /tmp/helmdeck-clone-qpNMUG
> ```

*Verified via OpenClaw 2026.5.6 + helmdeck v0.9.0-dev + `openrouter/openai/gpt-oss-120b` on 2026-05-07 (cost: $0.0883).*

## Developer reference (`curl`)

Happy path — list a directory:

```bash
curl -fsS -X POST http://localhost:3000/api/v1/packs/cmd.run \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d "{
    \"_session_id\":\"$SESSION\",
    \"clone_path\":\"$CLONE\",
    \"command\":[\"ls\",\"docs/reference/packs\"]
  }"
```

Captured response:

```json
{
  "pack": "cmd.run",
  "version": "v1",
  "output": {
    "exit_code": 0,
    "stderr": "",
    "stdout": "browser\nindex.md\n_template.md\n"
  },
  "session_id": "022b902e-fcf4-4853-b65e-97cf9896cc81"
}
```

Non-zero exit (still a successful pack call — exit_code is data):

```bash
curl -fsS -X POST http://localhost:3000/api/v1/packs/cmd.run \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d "{
    \"_session_id\":\"$SESSION\",
    \"clone_path\":\"$CLONE\",
    \"command\":[\"sh\",\"-c\",\"echo to stderr 1>&2; exit 7\"]
  }"
```

```json
{
  "pack": "cmd.run",
  "version": "v1",
  "output": {
    "exit_code": 7,
    "stderr": "to stderr\n",
    "stdout": ""
  }
}
```

## Error codes

| Code | Triggers | Captured response |
|---|---|---|
| `invalid_input` | `command` is a string, not an array | `{"error":"invalid_input","message":"field \"command\": expected array, got string"}` |
| `invalid_input` | path-safety violations on `clone_path` | per `safeJoin` |
| `session_unavailable` | session expired | session not found |
| `handler_failed` | container exec itself fails (sidecar dead) | exec error |

A non-zero exit is *not* an error — it's a normal outcome with `exit_code` set. Output truncation past 8 MiB *also* isn't an error; the trailing bytes are silently dropped (consider piping to a file via `>` if you need full output for huge runs).

## Session chaining

`needs_session: true`. The full Phase 5.5 loop: `repo.fetch` → `fs.read` → `fs.patch` → **`cmd.run`** (build/test) → `git.commit` → `repo.push`.

## Async behavior

Synchronous. Bounded by the engine's per-pack deadline (default 60s, override via session `timeout`).

## See also

- [`fs.write`](../fs/write.md), [`fs.patch`](../fs/patch.md) — produce the changes `cmd.run` then verifies.
- [`git.commit`](../git/commit.md) — capture the verified state.
- [`http.fetch`](../http/fetch.md) — for vault placeholder substitution patterns; the same resolver underlies this pack's command env.
- Source: [`internal/packs/builtin/fs_packs.go`](https://github.com/tosin2013/helmdeck/blob/main/internal/packs/builtin/fs_packs.go).
