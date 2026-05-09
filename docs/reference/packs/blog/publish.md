---
title: blog.publish
description: Publish a post to a Ghost blog (live Admin API) or write rendered markdown/HTML to the helmdeck artifact store. Two body modes (agent supplies body OR prompt+model the pack expands), two destinations, two formats.
keywords: [helmdeck, blog, publish, ghost, markdown, html, MCP]
---

# `blog.publish`

The "publish a blog post" pack. Two destinations, two body modes, two formats — picked at call time. Closes [#68](https://github.com/tosin2013/helmdeck/issues/68).

| Axis | Options |
|---|---|
| **Destination** | `ghost` (live publish via Ghost Admin API) · `artifact` (render to a helmdeck artifact, no external network) |
| **Body mode** | `body` (the agent already wrote the post) · `prompt + model` (the pack expands the prompt into a body via the gateway LLM) |
| **Format** | `markdown` (rendered to HTML via goldmark when Ghost destination needs it) · `html` (pre-rendered, passes through) |

The two body modes let an agent treat publishing as either a *primitive* (it composed the body upstream and just hands it off) or as a *macro* (it knows what it wants but lets the pack do the writing). The two destinations let the same agent publish a draft to a real Ghost blog OR generate a stand-alone artifact a downstream system can pick up.

## Setup prerequisite

For the `ghost` destination, add the Ghost Admin API key to the *Vault* panel:

| Field | Value |
|---|---|
| **Name** | `ghost-admin-key` (exact string — pack default; override with `credential` input) |
| **Type** | `api_key` |
| **Host pattern** | Your Ghost installation's hostname (e.g. `blog.example.com`) |
| **Value** | The full Admin API key in `<id>:<secret>` form (Ghost ships them this way; secret is hex-encoded) |

Get the key from your Ghost admin: **Settings → Advanced → Integrations → Add custom integration → Admin API Key**. The key looks like `650f...:a1b2c3...` — paste the whole thing, including the colon.

For the `artifact` destination, **no vault credential is needed** — the pack writes locally to the helmdeck artifact store.

## Inputs

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `destination` | `string` | yes | — | `"ghost"` or `"artifact"`. |
| `format` | `string` | yes | — | `"markdown"` or `"html"`. |
| `title` | `string` | yes | — | Post title. Slugified for the artifact filename. |
| `body` | `string` | one-of | — | The post body. **Either** this **or** `prompt`+`model`. |
| `prompt` | `string` | one-of | — | Generation prompt for prompt mode. **Either** this **or** `body`. |
| `model` | `string` | with `prompt` | — | Provider/model for prompt mode (e.g. `openrouter/openai/gpt-4o-mini`). |
| `max_tokens` | `number` | no | `1024` | Cap on the prompt-mode body length. Ignored in body mode. |
| `tags` | `array` | no | `[]` | Tag names. For Ghost, converted to `{name: ...}` objects. |
| `status` | `string` | no | `"draft"` | `"draft"` (default), `"published"`, or `"scheduled"`. |
| `published_at` | `string` | with `status="scheduled"` | — | RFC3339 timestamp in the future. |
| `host` | `string` | with `destination="ghost"` | — | Ghost installation hostname. Accepts `host`, `https://host`, or `http://host:port` (the last for self-hosted Ghost on a non-HTTPS port). |
| `credential` | `string` | no | `"ghost-admin-key"` | Vault credential name. Override only if you store the key under a non-default name. |

**Validation:**
- Exactly one of `body` or (`prompt`+`model`) — providing both or neither errors.
- `status="scheduled"` requires `published_at` in the future.
- `destination="ghost"` requires `host` and a vault credential.

## Outputs

Common fields:

| Field | Type | Notes |
|---|---|---|
| `destination` | `string` | Echo. |
| `format` | `string` | Echo. |
| `body_source` | `string` | `"input"` (body mode) or `"model"` (prompt mode). |
| `model_used` | `string` | Only in prompt mode — the model that generated the body. |

Ghost-specific:

| Field | Type | Notes |
|---|---|---|
| `post_id` | `string` | Ghost post id. |
| `url` | `string` | Public URL. |
| `html_url` | `string` | Same as `url`, for parity with `github.*` packs. |
| `status` | `string` | Ghost-confirmed status. |
| `published_at` | `string` | Ghost-assigned RFC3339. |

Artifact-specific:

| Field | Type | Notes |
|---|---|---|
| `artifact_key` | `string` | `blog.publish/<slug>.{md\|html}`. Resolve via `/api/v1/artifacts/<key>`. |
| `size` | `number` | Bytes. |

## Vault credentials needed

`ghost-admin-key` for ghost destination only. **Optional for artifact destination.**

## Use it from your agent (OpenClaw chat-UI worked example)

<!-- TODO(maintainer): paste an OpenClaw chat-UI transcript here.
     Prompt to use: "Use helmdeck__blog-publish in artifact mode with destination=artifact, format=markdown, title=\"Demo PR-D2 post\", body=\"# Hello\\n\\nThis is a test.\". Tell me the artifact_key and size." -->

> *OpenClaw chat capture pending.*

## Developer reference (`curl`)

### Artifact mode (no Ghost required)

```bash
ADMIN_PW=$(grep HELMDECK_ADMIN_PASSWORD /root/helmdeck/deploy/compose/.env.local | cut -d= -f2)
JWT=$(curl -fsS -X POST http://localhost:3000/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"admin\",\"password\":\"${ADMIN_PW}\"}" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')

curl -fsS -X POST http://localhost:3000/api/v1/packs/blog.publish \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d '{
    "destination": "artifact",
    "format":      "markdown",
    "title":       "Demo PR-D2 post",
    "body":        "# Hello\n\nThis is a test."
  }'
```

Response:

```json
{
  "pack": "blog.publish",
  "version": "v1",
  "output": {
    "destination":  "artifact",
    "format":       "markdown",
    "body_source":  "input",
    "artifact_key": "blog.publish/demo-pr-d2-post.md",
    "size":         101
  }
}
```

### Ghost mode (live API)

```bash
curl -fsS -X POST http://localhost:3000/api/v1/packs/blog.publish \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d '{
    "destination": "ghost",
    "format":      "markdown",
    "title":       "Hello from helmdeck",
    "body":        "# Welcome\n\nThis post was filed via blog.publish.",
    "host":        "blog.example.com",
    "tags":        ["demo","helmdeck"],
    "status":      "draft"
  }'
```

Response:

```json
{
  "pack": "blog.publish",
  "version": "v1",
  "output": {
    "destination":  "ghost",
    "format":       "markdown",
    "body_source":  "input",
    "post_id":      "650f1234567890",
    "url":          "https://blog.example.com/p/hello-from-helmdeck/",
    "html_url":     "https://blog.example.com/p/hello-from-helmdeck/",
    "status":       "draft",
    "published_at": null
  }
}
```

### Prompt mode + Ghost

```bash
curl -fsS -X POST http://localhost:3000/api/v1/packs/blog.publish \
  -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' \
  -d '{
    "destination": "ghost",
    "format":      "markdown",
    "title":       "Why packs beat naive function-calling",
    "prompt":      "Write a 400-word post arguing that typed packs (helmdeck) yield 10x lower per-task LLM cost than naive function-calling on Sonnet. Use a concrete example.",
    "model":       "openrouter/openai/gpt-4o-mini",
    "max_tokens":  600,
    "host":        "blog.example.com",
    "status":      "draft",
    "tags":        ["agent-architecture","cost"]
  }'
```

The pack calls the gateway LLM with a frozen system prompt that instructs it to emit ONLY the post body in the requested format (no preamble, no surrounding code fences, no repeated title).

## Error codes

| Code | Triggers | Captured response |
|---|---|---|
| `invalid_input` | `destination` outside `"ghost"`/`"artifact"` | `destination must be "ghost" or "artifact"` |
| `invalid_input` | `format` outside `"markdown"`/`"html"` | `format must be "markdown" or "html"` |
| `invalid_input` | `title` empty | `title is required` |
| `invalid_input` | Both `body` AND `prompt` supplied | `must provide either body OR prompt+model, not both` |
| `invalid_input` | Neither `body` nor `prompt` supplied | `must provide either body OR prompt+model` |
| `invalid_input` | `prompt` set but `model` missing | `prompt mode requires model (provider/model)` |
| `invalid_input` | `status` outside the closed set | `status must be "draft", "published", or "scheduled"` |
| `invalid_input` | `status="scheduled"` without `published_at` | `published_at (RFC3339) is required when status=scheduled` |
| `invalid_input` | `published_at` not in the future | `published_at must be in the future for status=scheduled` |
| `invalid_input` | `destination="ghost"` without `host` | `host is required when destination=ghost` |
| `invalid_input` | `ghost-admin-key` not in vault | `vault credential "ghost-admin-key" not found …` |
| `invalid_input` | Vault key not in `id:hex_secret` form | `ghost-admin-key vault value must be \`<id>:<secret>\` …` |
| `invalid_input` | Ghost host resolves to a blocked range | `egress denied: …` |
| `internal` | Prompt mode but pack registered without a gateway dispatcher | `blog.publish prompt mode registered without a gateway dispatcher` |
| `handler_failed` | Ghost API non-2xx | `ghost API POST …: 401 Authorization failed` |
| `handler_failed` | Markdown→HTML conversion failed | `markdown→html for Ghost: …` |
| `handler_failed` | Prompt expansion model returned no choices | `blog.publish prompt expansion: model returned no choices` |
| `artifact_failed` | Object store write failed | `artifact upload failed: …` |

## Session chaining

**No session.** Stateless. Composes naturally:

- **`research.deep` → `content.ground` → `blog.publish`** — the canonical "evidence-grounded blog post" chain. Research surfaces sources; content.ground appends citations into a draft body; blog.publish ships it.
- **`web.scrape` → `blog.publish` (artifact mode)** — re-publish a scraped page as a draft artifact for later editing.
- **`repo.fetch` + `fs.read` + `blog.publish` (prompt mode)** — generate a release-notes blog post from a repo's recent changelog without the agent ever materializing the body itself.

## Async behavior

Synchronous. Wall-clock = (prompt-mode LLM call, ~3–10s if used) + (markdown→html via goldmark, ~1ms) + (Ghost API round-trip, ~200–800ms) for ghost mode; or just the goldmark step + artifact upload for artifact mode (~10–50ms).

## See also

- Catalog row: [`PACKS.md`](/PACKS) — `blog.publish`.
- Source: [`internal/packs/builtin/blog_publish.go`](https://github.com/tosin2013/helmdeck/blob/main/internal/packs/builtin/blog_publish.go).
- Issue: [#68](https://github.com/tosin2013/helmdeck/issues/68).
- Companion packs: [`research.deep`](../research/deep.md) (source discovery), [`content.ground`](../content/ground.md) (citation injection), [`http.fetch`](../http/fetch.md) (read-only blog API access if/when needed).
- Vault setup: see "Setup prerequisite" above.
- Ghost Admin API docs: <https://ghost.org/docs/admin-api/>
