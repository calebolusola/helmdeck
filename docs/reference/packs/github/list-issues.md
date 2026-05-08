---
title: github.list_issues
description: List issues on a GitHub repository. Filter by state, labels, and assignee. Returns the GitHub issue array verbatim plus a count.
keywords: [helmdeck, github, issues, REST, MCP]
---

# `github.list_issues`

The "show me what's open on this repo" pack. Caller supplies `repo` and optional filters (`state`, `labels`, `assignee`); the pack GETs `/repos/{repo}/issues` with the filters and returns the GitHub issue array verbatim wrapped in `{issues: [...], count: N}`.

Stateless — no session needed. Public-repo reads work without a credential (60 req/hr unauthenticated rate limit); authenticated calls get the standard 5000 req/hr.

## Inputs

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `repo` | `string` | yes | — | `owner/name` slug. |
| `state` | `string` | no | `"open"` | One of `open`, `closed`, `all`. |
| `labels` | `string` | no | `""` | Comma-separated label names. AND'd — issues must have all listed labels. |
| `assignee` | `string` | no | `""` | Login of an assignee. |
| `credential` | `string` | no | `github-token` | Optional vault credential. Public reads work without it. |

## Outputs

| Field | Type | Notes |
|---|---|---|
| `issues` | `array` | The GitHub issue objects verbatim. Each carries `number`, `title`, `state`, `body`, `labels[].name`, `user.login`, `assignees[]`, `created_at`, `updated_at`, `html_url`, etc. |
| `count` | `number` | `len(issues)`. Note: paginated upstream — for a repo with &gt;30 open issues, this is page 1's count, not the total. Filter or paginate via additional calls. |

> ⚠️ **Indexing latency observed at capture**: immediately after `github.create_issue` succeeds, this pack may return `count: 0` for a few minutes — GitHub's list endpoint is eventually-consistent with respect to issue creation. The created issue is fetchable directly via `gh api repos/.../issues/N` without delay.

## Vault credentials needed

**Optional.** `github-token` (PAT) raises the rate limit to 5000 req/hr and is required for private repos. Public-repo reads work without it.

## Use it from your agent (OpenClaw chat-UI worked example)

**Prompt** (sent in OpenClaw chat UI / `openclaw-cli agent`):

> Use the helmdeck__github-list_issues tool against repo "tosin2013/helmdeck-pack-doc-fixtures" with state=open and credential=github-token. Tell me the count and the titles.

**Tool call** (1 call, no failures):

```json
{
  "name": "helmdeck__github-list_issues",
  "arguments": {
    "repo": "tosin2013/helmdeck-pack-doc-fixtures",
    "state": "open",
    "credential": "github-token"
  }
}
```

**Pack response** (truncated):

```json
{"pack":"github.list_issues","version":"v1","output":{"count":2,"issues":[{"url":"https://api.github.com/repos/tosin2013/helmdeck-pack-doc-fixtures/issues/2","repository_url":"https://api.github.com/repos/tosin2013/helmdeck-pack-doc-fixtures","labels_url":"https://api.github.com/repos/tosin2013/helmdeck-pack-doc-fixtures/issues/2/labels{/name}","comments_url":"https://api.github.com/repos/tosin2013/helmdeck-pack-doc-fixtures/issues/2/comments","events_url":"https://api.github.com/repos/tosin2013/helmdeck-pack-doc-fixtures/issues/2/events","html_url":"https://github.com/tosin2013/helmdeck-pack-doc-fixtures/issues/2","id":4406682427,"node_id":"I_kwDOSX1U_c8AAAABBqijOw","number":2,"title":"Demo issue from helmdeck pack capture","user":{"login":"tosin2013","id":1975599,"node_id":"MDQ6VXNlcjE5N
  ...[truncated]
```

**Agent's final reply**:

> The repository has **2 open issues**:
> 
> 1. **Demo issue from helmdeck pack capture** (issue #2)  
> 2. **Demo issue from helmdeck pack capture** (issue #1)

*Verified via OpenClaw 2026.5.6 + helmdeck v0.9.0-dev + `openrouter/openai/gpt-oss-120b` on 2026-05-07 (cost: $0.0013).*
## Developer reference (`curl`)

```bash
curl -fsS -X POST http://localhost:3000/api/v1/packs/github.list_issues \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d '{
    "repo":       "tosin2013/helmdeck-pack-doc-fixtures",
    "state":      "open",
    "credential": "github-token"
  }'
```

Captured response (empty repo):

```json
{
  "pack": "github.list_issues",
  "version": "v1",
  "output": {
    "count":  0,
    "issues": []
  },
  "duration_ms": 372152259
}
```

## Error codes

| Code | Triggers | Captured response |
|---|---|---|
| `handler_failed` | `repo` missing | `{"error":"handler_failed","message":"repo is required"}` |
| `handler_failed` | Repo doesn't exist or PAT lacks access | `{"error":"handler_failed","message":"github API GET /repos/…/issues: 404 Not Found"}` |
| `handler_failed` | Rate-limited (60 req/hr unauthenticated) | `{"error":"handler_failed","message":"github API GET …: 403 API rate limit exceeded …"}` |

## Session chaining

**No session.** Stateless. Common chain — list, then iterate via `github.post_comment` against each one ("triage every open issue with comment X").

## Async behavior

Synchronous.

## See also

- Catalog row: [`PACKS.md`](/PACKS) — `github.list_issues`.
- Source: [`internal/packs/builtin/github.go`](https://github.com/tosin2013/helmdeck/blob/main/internal/packs/builtin/github.go).
- ADR 034 — Core GitHub pack set.
- Companion packs: [`github.list_prs`](./list-prs.md), [`github.search`](./search.md), [`github.create_issue`](./create-issue.md).
