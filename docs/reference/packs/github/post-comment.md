---
title: github.post_comment
description: Post a comment on a GitHub issue or pull request. Returns the new comment id and URL.
keywords: [helmdeck, github, comment, issue, PR, REST, MCP]
---

# `github.post_comment`

The "leave a comment on issue/PR #N" pack. Caller supplies `repo`, the `issue_number` (works on both issues and PRs — GitHub treats them as the same resource for comments), and the `body`. The pack POSTs to `/repos/{repo}/issues/{N}/comments`.

Stateless.

## Inputs

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `repo` | `string` | yes | — | `owner/name`. |
| `issue_number` | `number` | yes | — | Issue or PR number. |
| `body` | `string` | yes | — | Markdown body. |
| `credential` | `string` | no | `github-token` | PAT scoped to write to the target repo. |

## Outputs

| Field | Type | Notes |
|---|---|---|
| `id` | `number` | Comment id. |
| `url` | `string` | API URL. |
| `html_url` | `string` | Web URL — the deep link to the comment anchor. |

## Vault credentials needed

**`github-token`** — required for writes. PAT with `repo` (Classic) or **Pull requests: Write** + **Issues: Write** (fine-grained).

## Use it from your agent (OpenClaw chat-UI worked example)

**Prompt** (sent in OpenClaw chat UI / `openclaw-cli agent`):

> Use helmdeck__github-post_comment against repo "tosin2013/helmdeck-pack-doc-fixtures", issue_number 1, body="Demo comment captured during pack doc work.", credential=github-token. Tell me the comment id and url.

**Tool call** (1 call, no failures):

```json
{
  "name": "helmdeck__github-post_comment",
  "arguments": {
    "repo": "tosin2013/helmdeck-pack-doc-fixtures",
    "issue_number": 1,
    "body": "Demo comment captured during pack doc work.",
    "credential": "github-token"
  }
}
```

**Pack response** (truncated):

```json
{"pack":"github.post_comment","version":"v1","output":{"url":"https://api.github.com/repos/tosin2013/helmdeck-pack-doc-fixtures/issues/comments/4406991866","html_url":"https://github.com/tosin2013/helmdeck-pack-doc-fixtures/issues/1#issuecomment-4406991866","issue_url":"https://api.github.com/repos/tosin2013/helmdeck-pack-doc-fixtures/issues/1","id":4406991866,"node_id":"IC_kwDOSX1U_c8AAAABBq1b-g","user":{"login":"tosin2013","id":1975599,"node_id":"MDQ6VXNlcjE5NzU1OTk=","avatar_url":"https://avatars.githubusercontent.com/u/1975599?u=979f34af430adca7132e3bf2fd67100bae6c8979\u0026v=4","gravatar_id":"","url":"https://api.github.com/users/tosin2013","html_url":"https://github.com/tosin2013","followers_url":"https://api.github.com/users/tosin2013/followers","following_url":"https://api.github.c
  ...[truncated]
```

**Agent's final reply**:

> Comment ID: 4406991866  
> URL: https://github.com/tosin2013/helmdeck-pack-doc-fixtures/issues/1#issuecomment-4406991866

*Verified via OpenClaw 2026.5.6 + helmdeck v0.9.0-dev + `openrouter/openai/gpt-oss-120b` on 2026-05-07 (cost: $0.0013).*
## Developer reference (`curl`)

```bash
curl -fsS -X POST http://localhost:3000/api/v1/packs/github.post_comment \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d '{
    "repo":         "tosin2013/helmdeck-pack-doc-fixtures",
    "issue_number": 1,
    "body":         "Demo comment captured during pack doc work.",
    "credential":   "github-token"
  }'
```

Captured response:

```json
{
  "pack": "github.post_comment",
  "version": "v1",
  "output": {
    "id":       4406638260,
    "url":      "https://api.github.com/repos/tosin2013/helmdeck-pack-doc-fixtures/issues/comments/4406638260",
    "html_url": "https://github.com/tosin2013/helmdeck-pack-doc-fixtures/issues/1#issuecomment-4406638260"
  }
}
```

## Error codes

| Code | Triggers | Captured response |
|---|---|---|
| `handler_failed` | `repo`, `issue_number`, or `body` missing | `{"error":"handler_failed","message":"repo, issue_number, and body are required"}` |
| `handler_failed` | Issue doesn't exist | `github API POST …: 404 Not Found` |
| `handler_failed` | PAT lacks write access | `github API POST …: 403 Resource not accessible by personal access token` |

## Session chaining

**No session.** Stateless. Common upstream: `github.list_issues` (find issue #N) → `github.post_comment` (comment on it). Common upstream: `web.scrape`/`research.deep` (gather context) → `github.post_comment` (post a triage summary).

## Async behavior

Synchronous.

## See also

- Catalog row: [`PACKS.md`](/PACKS) — `github.post_comment`.
- Source: [`internal/packs/builtin/github.go`](https://github.com/tosin2013/helmdeck/blob/main/internal/packs/builtin/github.go).
- ADR 034 — Core GitHub pack set.
- Companion packs: [`github.create_issue`](./create-issue.md), [`github.list_issues`](./list-issues.md), [`github.list_prs`](./list-prs.md).
