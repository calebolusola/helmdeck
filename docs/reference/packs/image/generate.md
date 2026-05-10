---
title: image.generate
description: Generate an image from a text prompt via fal.ai. Day 1 ships fal.ai (FLUX schnell/dev/pro, SDXL); Replicate is reserved for a follow-up community PR.
keywords: [helmdeck, image, generate, fal.ai, flux, sdxl, MCP]
---

# `image.generate`

Text → image via [fal.ai](https://fal.ai)'s synchronous `fal.run` endpoint. Caller supplies a prompt; the pack POSTs to fal.run, downloads the resulting PNG/JPEG bytes, and stores them in helmdeck's artifact store. Returns an `image_artifact_key` the agent (or downstream packs) can resolve.

**Day 1** ships **fal.ai** because its sync API returns the generated image URL in the same POST response — no polling loop, fits within the MCP 60s JSON-RPC timeout, ~150 lines of pack code. The `engine` input field is reserved so a follow-up PR can add Replicate (queue+poll) without changing the schema.

Default model is `fal-ai/flux/schnell` — fast (~1-3s), cheap (~$0.003/image), photorealistic enough for podcast covers, blog hero images, and slide shields. Operators pass their own `model` for FLUX dev/pro/SDXL/etc.

## Setup prerequisite

Add the fal.ai API key to the *Vault* panel:

| Field | Value |
|---|---|
| **Name** | `fal-key` (exact string — pack default; override with `credential` input) |
| **Type** | `api_key` |
| **Host pattern** | `fal.run` |
| **Value** | Your fal.ai API key (`f1_…`) |

Or set `HELMDECK_FAL_KEY=...` in `deploy/compose/.env.local` — once #142 (vault env-hydrate) lands, this auto-imports as `fal-key` on startup. Until then, the env var works as a last-resort fallback after the vault lookup.

**Required.** Without a key the pack returns `invalid_input` with `"fal.ai key not found. Set HELMDECK_FAL_KEY in deploy/compose/.env.local..."`. (Same fail-loud shape as `podcast.generate` post-#138.)

## Inputs

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `prompt` | `string` | yes | — | Plain-English description of the image. |
| `engine` | `string` | no | `"fal"` | Closed set; day 1 only `"fal"`. |
| `model` | `string` | no | `"fal-ai/flux/schnell"` | Any fal.ai model id. Common: `fal-ai/flux/dev`, `fal-ai/flux-pro`, `fal-ai/fast-sdxl`. |
| `image_size` | `string` | no | model default | fal.ai-specific: `square_hd`, `portrait_4_3`, `landscape_16_9`, etc. See the model's fal.ai page. |
| `num_images` | `number` | no | `1` | 1-4. Each image is a separate `image_artifact_key`. |
| `seed` | `number` | no | random | For reproducibility. fal.ai echoes the seed it used in `seed_used`. |
| `credential` | `string` | no | `"fal-key"` | Vault credential name override. |

## Outputs

| Field | Type | Notes |
|---|---|---|
| `image_artifact_key` | `string` | First (or only) generated image. `image.generate/<rand>.png`. |
| `image_size` | `number` | Bytes of the first image. |
| `engine` | `string` | Echo. Always `"fal"` day 1. |
| `model_used` | `string` | Echo of resolved model id. |
| `prompt_used` | `string` | Echo. |
| `seed_used` | `number` | Whichever seed fal.ai actually used (echoed from the response). |
| `image_artifact_keys` | `array` | Present when `num_images > 1`. Same order as fal.ai's `images[]`. |

## Vault credentials needed

`fal-key` (canonical name). Falls back to `HELMDECK_FAL_KEY` env var if vault lookup misses. Hard-fails with `missing_credential`-style message when neither is set.

## Use it from your agent (OpenClaw chat-UI worked example)

<!-- TODO(maintainer): paste an OpenClaw chat-UI transcript here.
     Prompt to use: "Use helmdeck__image-generate to make a podcast cover for an episode about WebAssembly performance. Use FLUX schnell. Tell me the image_artifact_key." -->

> *OpenClaw chat capture pending.*

## Developer reference (`curl`)

```bash
TOKEN=$(./bin/control-plane -mint-token dev -mint-token-scopes admin)
curl -fsS -X POST http://localhost:3000/api/v1/packs/image.generate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "a cat sitting on a podcast microphone, photorealistic",
    "model": "fal-ai/flux/schnell",
    "image_size": "square_hd"
  }'
```

Response:

```json
{
  "image_artifact_key": "image.generate/c34d.../image-000.png",
  "image_size": 287342,
  "engine": "fal",
  "model_used": "fal-ai/flux/schnell",
  "prompt_used": "a cat sitting on a podcast microphone, photorealistic",
  "seed_used": 42
}
```

Resolve the artifact:

```bash
curl -fsS -H "Authorization: Bearer $TOKEN" \
  http://localhost:3000/api/v1/artifacts/image.generate/c34d.../image-000.png \
  -o cover.png
```

## Cost

| Model | Approx. cost / image | Approx. wall time |
|---|---|---|
| `fal-ai/flux/schnell` | $0.003 | 1-3s |
| `fal-ai/flux/dev` | $0.025 | 3-8s |
| `fal-ai/flux-pro` | $0.05 | 5-15s |
| `fal-ai/fast-sdxl` | $0.005 | 1-2s |

Cost preview / `dry_run` semantics like `podcast.generate` (#145) aren't yet wired into `image.generate` — file an issue if you need them.

## Errors

| Code | When |
|---|---|
| `invalid_input` | Empty `prompt`; unknown `engine`; `num_images` out of [1, 4]; missing `fal-key`. |
| `handler_failed` | fal.ai returns 4xx/5xx; image download fails; response shape unexpected. |
| `artifact_failed` | Artifact store rejects the upload (disk full, S3 5xx, etc.). |
| `internal` | No artifact store wired (control-plane misconfiguration). |

## Future engines

The `engine` field is closed-set (`"fal"` only) day 1. Adding **Replicate** is a community-friendly follow-up: switch on `engine`, factor out the fal.run logic into an internal `falEngine`, add a `replicateEngine` peer that handles the queue+poll loop. The credential lookup ladder + artifact upload paths are engine-agnostic — only the request/response shape changes.

See [#71](https://github.com/tosin2013/helmdeck/issues/71) for the original spec; [#146](https://github.com/tosin2013/helmdeck/issues/146) tracks the chained-into-podcast/slides/blog work that builds on this pack.
