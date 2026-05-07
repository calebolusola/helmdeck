---
title: node.run
description: Run JavaScript code or commands inside a Node-equipped sidecar (per-pack image override). Node 20 LTS + npm + pnpm + yarn + tsc preinstalled.
keywords: [helmdeck, node.run, sidecar, code execution, MCP]
---

# `node.run`

The JavaScript counterpart to [`python.run`](./python-run.md). Runs JS code (`node -e`) or an argv-shape command inside a Node-equipped sidecar. **Node 20 LTS** + `npm` + `pnpm` + `yarn` + `tsc` (TypeScript compiler) preinstalled.

> ⚙️ **Setup note**: same as `python.run` — needs `make sidecars` to build the image, plus `HELMDECK_SIDECAR_NODE=helmdeck-sidecar-node:dev` in `.env.local`, plus a control-plane recreate.

## Inputs

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `code` | `string` | one of | — | JS code to run via `node -e`. |
| `command` | `array` | one of | — | argv-style command (e.g. `["npx", "tsc", "--noEmit"]`). |
| `cwd` | `string` | no | — | Working directory. |
| `stdin` | `string` | no | — | Bytes piped to stdin. |

## Outputs

| Field | Type | Notes |
|---|---|---|
| `stdout` | `string` | |
| `stderr` | `string` | |
| `exit_code` | `number` | |
| `runtime` | `string` | Always `node`. |

## Vault credentials needed

**None directly.** Same env-var-based vault pattern available as `python.run` if needed.

## Use it from your agent (OpenClaw chat-UI worked example)

**Prompt** (sent in OpenClaw chat UI / `openclaw-cli agent`):

> Use the helmdeck__node-run tool to execute this JavaScript: const v = process.version; const r = require("crypto").createHash("sha256").update("helmdeck").digest("hex"); console.log(v, r). Report the Node version and the sha256 hex digest.

**Tool call** (1 call, no failures):

```json
{
  "name": "helmdeck__node-run",
  "arguments": {
    "code": "const v = process.version; const r = require(\"crypto\").createHash(\"sha256\").update(\"helmdeck\").digest(\"hex\"); console.log(v, r);"
  }
}
```

**Pack response** (truncated):

```json
{"pack":"node.run","version":"v1","output":{"exit_code":0,"runtime":"node","stderr":"","stdout":"v20.20.2 ab8f104b28f3f2ca9f9f41099400b31114ed7d7599487666f1e6e998616fb8b4\n"},"duration_ms":2284290299,"session_id":"4ab3278a-2601-4897-a475-8fce0886d3f2"}
```

**Agent's final reply**:

> Node version: **v20.20.2**  
> SHA‑256 hex digest of “helmdeck”: **ab8f104b28f3f2ca9f9f41099400b31114ed7d7599487666f1e6e998616fb8b4**

*Verified via OpenClaw 2026.5.6 + helmdeck v0.9.0-dev + `openrouter/openai/gpt-oss-120b` on 2026-05-07 (cost: $0.1351).*

## Developer reference (`curl`)

```bash
curl -fsS -X POST http://localhost:3000/api/v1/packs/node.run \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d '{"code":"console.log(\"Node\", process.version); console.log(2+2);"}'
```

Captured response:

```json
{
  "pack": "node.run",
  "version": "v1",
  "output": {
    "exit_code": 0,
    "runtime": "node",
    "stderr": "",
    "stdout": "Node v20.20.2\n4\n"
  },
  "session_id": "3a78ae97-9cb7-4648-8393-de5aeb683508"
}
```

## Error codes

Same closed set as `python.run`:

| Code | Triggers |
|---|---|
| `invalid_input` | both / neither of `code`/`command` set |
| `session_unavailable` | Node sidecar image missing |
| `handler_failed` | container exec fails |

Non-zero exits are *not* errors — they're surfaced as `exit_code` with the corresponding stderr.

## Session chaining

Same pattern as `python.run` — own sidecar, can be chained off a `repo.fetch` by passing the `clone_path` as `cwd`.

## Async behavior

Synchronous. ~1–2 second cold start, fast on warm sessions.

## See also

- [`python.run`](./python-run.md) — Python sibling.
- [`cmd.run`](../cmd/run.md) — for ad-hoc shell work in the browser sidecar.
- Source: [`internal/packs/builtin/node_run.go`](https://github.com/tosin2013/helmdeck/blob/main/internal/packs/builtin/node_run.go).
- ADR 001 — sidecar pattern.
- [`SIDECAR-LANGUAGES.md`](/SIDECAR-LANGUAGES) — adding new language sidecars.
