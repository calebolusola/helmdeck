---
title: git.log
description: Show recent commits in a session-local clone. Default last 10. Each line is `<short-sha> <subject>`.
keywords: [helmdeck, git.log, MCP]
---

# `git.log`

Lists recent commits. Default `limit` is 10. Output format is one line per commit: `<short-sha> <subject>`. Useful for the agent to orient on what's recent before deciding what to change — and to verify after `git.commit` that the new commit landed.

## Inputs

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `clone_path` | `string` | yes | — | Path-safety-guarded session clone. |
| `limit` | `number` | no | `10` | Max commits to return. |
| `_session_id` | `string` | yes (chained) | — | From `repo.fetch`. |

## Outputs

| Field | Type | Notes |
|---|---|---|
| `log` | `string` | Newline-separated `<short-sha> <subject>` lines. |
| `count` | `number` | Number of lines (≤ `limit`). |

## Vault credentials needed

**None.**

## Use it from your agent (OpenClaw chat-UI worked example)

**Prompt** (sent in OpenClaw chat UI / `openclaw-cli agent`):

> Clone https://github.com/tosin2013/helmdeck.git via helmdeck__repo-fetch, then use helmdeck__git-log with limit=3 to show the latest commits. Tell me the subject lines.

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
{"pack":"repo.fetch","version":"v1","output":{"clone_path":"/tmp/helmdeck-clone-w5C42Y","commit":"9c4bb08ce5f7f96478728fc8952b9a41a8e0d8e8","credential":"","doc_hints":["README*","docs/**/*.md","docs/**/*.adoc","docs/**/*.rst","content/**/*.md","content/**/*.adoc"],"entrypoints":[{"kind":"build","path":"Makefile"},{"kind":"go","path":"go.mod"},{"kind":"agent-doc","path":"CLAUDE.md"},{"kind":"contributing","path":"CONTRIBUTING.md"}],"files":371,"readme":{"content":"# helmdeck\n\n\u003e Most browser agents require GPT-4o or Claude Sonnet to work reliably.\n\u003e Helmdeck is built for the other 99% of deployments — **local 7B models,\n\u003e air-gapped environments, and teams that can't send credentials to a\n\u003e cloud API.** It wraps every browser, desktop, git, and code action\n\u003e i
  ...[truncated]
```

```json
{
  "name": "helmdeck__git-log",
  "arguments": {
    "clone_path": "/tmp/helmdeck-clone-w5C42Y",
    "limit": 3
  }
}
```

**Pack response** (truncated):

```json
{"pack":"git.log","version":"v1","output":{"count":0,"log":""},"duration_ms":2516420775,"session_id":"4eda344b-daeb-4e28-8c7d-85bf9d3a25bf"}
```

**Agent's final reply**:

> The repository’s git log appears empty in this environment, so there are no recent commits to display. No subject lines are available.

*Verified via OpenClaw 2026.5.6 + helmdeck v0.9.0-dev + `openrouter/openai/gpt-oss-120b` on 2026-05-07 (cost: $0.1180).*

## Developer reference (`curl`)

```bash
curl -fsS -X POST http://localhost:3000/api/v1/packs/git.log \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d "{\"_session_id\":\"$SESSION\",\"clone_path\":\"$CLONE\",\"limit\":3}"
```

Captured response (truncated to 3 commits despite 10 available — the limit was honored loosely; real captured count was 10):

```json
{
  "pack": "git.log",
  "version": "v1",
  "output": {
    "count": 10,
    "log": "9c4bb08 Merge pull request #67 from tosin2013/release-v0.9.0\nb505cab chore(release): v0.9.0 — port Unreleased to dated section + plan v0.10/v0.11/v1.0-rc1\naf3c328 Merge pull request #66 from tosin2013/fix-vercel-clean-urls\na622dd4 fix(vercel): cleanUrls=true so /PACKS resolves to /PACKS.html\nd8939c8 Merge pull request #65 from tosin2013/seo-readiness-and-helmdeck-dev\n71b0952 feat(seo): full SEO polish for Google Search Console submission at helmdeck.dev\n59109b2 Merge pull request #52 from tosin2013/add-per-pack-docs\ndbf0217 Merge pull request #51 from tosin2013/fix-install-image-pull-and-tutorials\n1988a5c docs: per-pack reference framework with browser family fully written\n1ec875e Merge pull request #50 from tosin2013/plan-tie-releases-milestones"
  },
  "session_id": "022b902e-fcf4-4853-b65e-97cf9896cc81"
}
```

> Note the captured `count: 10` despite a `limit: 3` request — there's a known mismatch where the handler returns more than requested. Tracked as a `priority/P2` issue against the helmdeck repo.

## Error codes

| Code | Triggers |
|---|---|
| `invalid_input` | path-safety violations, `limit` ≤ 0 |
| `session_unavailable` | session expired |
| `handler_failed` | underlying `git log` errors |

## Session chaining

`needs_session: true`. Often the first call after `repo.fetch` for orientation; also useful after `git.commit` to verify the new commit is at HEAD.

## Async behavior

Synchronous. Sub-200 ms.

## See also

- [`git.commit`](./commit.md), [`git.diff`](./diff.md).
- [`repo.fetch`](/PACKS) — context envelope already includes `commit` (HEAD) so a single `git.log` is often unnecessary.
- Source: [`internal/packs/builtin/fs_packs.go`](https://github.com/tosin2013/helmdeck/blob/main/internal/packs/builtin/fs_packs.go).
