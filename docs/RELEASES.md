# Helmdeck — Release Plan ("What Ships When")

Forward-looking changelog. Each release maps 1:1 to a phase milestone (`MILESTONES.md`) and has hard exit criteria pulled from `TASKS.md`.

---

## Agent sync checklist — every release

Helmdeck ships its agent instructions as a native **OpenClaw Skill** at `skills/helmdeck/SKILL.md`, stamped with the helmdeck commit hash in its frontmatter (`metadata.openclaw.helmdeckVersion`). The stamp is how operators detect drift between their deployed agent and the latest release.

**Every release — required:**

1. **Update the pack count and decision tables** in `skills/helmdeck/SKILL.md` if this release adds/removes packs, changes an error code, or revises a pattern (e.g. the `repo.fetch` signals table).
2. **Bump the `helmdeckVersion` stamp** — `scripts/configure-openclaw.sh` regenerates this automatically from `git rev-parse --short HEAD` at install time, so you don't edit it by hand. Ensure the release commit lands on `main` before operators run the configure script, otherwise the stamp reflects a stale pointer.
3. **Call out new packs** in the release notes under "Ships" with their full `helmdeck__<name>` MCP prefix, so operators (and agents reading the release notes post-fact) know what's new.
4. **Tell deployed operators to refresh**:
   ```bash
   cd /path/to/helmdeck && git pull
   ./scripts/configure-openclaw.sh            # reinstalls the versioned SKILL.md
   ```
   The script is idempotent; re-running it without other flags will only touch the skill, the JWT (if expiring), and the model pin.
5. **Document upstream regressions** — if OpenClaw itself ships a breaking change between our tested versions and the current one, add a row to the table in `docs/integrations/openclaw-upgrade-runbook.md` pointing at the affected version range and the workaround.
6. **Refresh the README + cost-positioning numbers** — `README.md` opens with time-stamped prose ("Today's helmdeck install ran a full 6-step code-edit loop … for $0.07") and a four-row cost-comparison table. The same numbers live in the long-form `docs/explanation/why-helmdeck.md` and the cost-positioning blog post at `website/blog/2026-05-08-cheap-models-do-frontier-work.md`. On a release that meaningfully changes pack performance, the chat model recommendation, or the OpenRouter pricing landscape:
   - Re-run the 5 reproduction workflows from `docs/explanation/why-helmdeck.md` §"Run the comparison yourself" against the new pack set.
   - Update the comparison table in **all three places** (README, long-form explanation, blog post) so the numbers don't drift between them.
   - Either revise the time-stamped prose at the top of the README to reflect the new release, or — if numbers haven't moved meaningfully — leave it but add a "Last verified: vX.Y on YYYY-MM-DD" footer line so readers know the cited workflow is fresh enough.

   On a release that does NOT change agent-side performance, the cost numbers are stable enough to skip this step; only update if you'd otherwise be overstating the gap.
7. **Operator upgrade procedure** — every release MUST be cleanly upgradable from the prior tag without operator data loss or extended downtime. Verify before tagging:
   - The procedure in [`docs/howto/upgrade-helmdeck.md`](howto/upgrade-helmdeck.md) §"In-place Compose-stack upgrade" runs cleanly against a fresh checkout: `git checkout v<new>; make sidecars; make install` produces a healthy stack
   - `internal/store/migrations/` has any new migrations needed for new tables/columns. Auto-applied via `store.Open` on next startup (no manual `migrate up` required), but the migration file MUST be additive — no `DROP COLUMN` or `ALTER TABLE … RENAME` that would break a v<new-1> binary trying to read the same DB
   - If a release introduces a destructive schema change, flag it under `### Breaking` in `CHANGELOG.md` AND link from the upgrade howto's §7 "Version-specific notes" table
   - Pack-input-schema changes that drop a previously-required field, or change a closed-set value, are also `### Breaking` — agents written against the old schema will error
   - Post-tag, smoke-test the upgrade against a snapshot of v<new-1>'s `helmdeck.db` (a manual cross-version run; the automated CI smoke is tracked at the Phase 7 audit issue list under "upgrade smoke-test in CI")
8. **Re-publish to the official MCP Registry** — automated via [`.github/workflows/mcp-registry.yml`](../.github/workflows/mcp-registry.yml). The workflow fires on every `v*` tag push: it pulls the tag's version into `.mcp/server.json`, schema-validates the document, authenticates to `registry.modelcontextprotocol.io` via GitHub OIDC (no PAT needed — the workflow's `id-token: write` permission is enough), and publishes. Watch the run; the workflow summary prints the live listing URL. Downstream aggregators (mcp.so, Glama, PulseMCP) ingest within 24h.

   If the workflow fails or you need to re-publish without cutting a new tag, two fallback paths:
   - **`workflow_dispatch`** — go to the Actions tab → "Publish to MCP Registry" → "Run workflow" with an optional `version_override` input
   - **Local script** — [`scripts/publish-to-mcp-registry.sh`](../scripts/publish-to-mcp-registry.sh) builds the publisher locally, runs interactive GitHub OAuth, and publishes from a maintainer shell. Useful if the GitHub Actions OIDC path breaks for any reason.

**Related:**
- [OpenClaw upgrade runbook](integrations/openclaw-upgrade-runbook.md) — the operator-facing sync procedure
- [ADR 025 — MCP client integrations](adrs/025-mcp-client-integrations.md) — architecture decision record; the §2026-04-18 revision covers CLI vs chat-UI regression policy
- `skills/helmdeck/SKILL.md` — the canonical agent skill file (source of truth)

---

## v0.1.0 — Core Infrastructure (Week 4)
**Theme:** "A browser session is one REST call away."

> **Milestone:** [v0.1 — Core Infrastructure (Phase 1)](MILESTONES.md#milestone-v01--core-infrastructure-phase-1) · **Tasks:** [Phase 1](TASKS.md#phase-1--core-infrastructure-weeks-14)

### Ships
- Go control plane binary (Gin + chromedp + Docker SDK)
- Browser sidecar image with Chromium, Marp, Tesseract, ffmpeg, xdotool, Xvfb, XFCE4, noVNC
- Ephemeral session lifecycle: `POST /api/v1/sessions` … `DELETE /api/v1/sessions/{id}`
- CDP REST endpoints: navigate, extract, screenshot, execute, interact
- JWT bearer auth on every endpoint
- Audit log (write-only at this stage)
- Single-node Compose deployment (`deploy/compose/compose.yaml`)
- `make smoke` end-to-end harness in CI

### Does NOT ship
- AI gateway, packs, MCP, vault, UI, Kubernetes — all later

### Audience
Internal only. Tag a pre-release on GitHub.

---

## v0.2.0 — AI Gateway & First Packs (Week 8)
**Theme:** "Capability Packs are real, and weak models can drive them."

> **Milestone:** [v0.2 — AI Gateway & Pack Substrate (Phase 2)](MILESTONES.md#milestone-v02--ai-gateway--pack-substrate-phase-2) · **Tasks:** [Phase 2](TASKS.md#phase-2--ai-gateway--capability-pack-substrate-weeks-58)

### Ships
- OpenAI-compatible `/v1/chat/completions` and `/v1/models`
- Provider adapters: Anthropic, Gemini, OpenAI, Ollama, Deepseek
- Encrypted key store with rotation API
- Fallback routing rules (rate-limit / error / timeout triggers)
- **Pack Execution Engine** with input/output schema validation
- **Typed error code enforcement** (closed set per pack)
- Pack registry with versioned dispatch
- **Three reference packs:** `browser.screenshot_url`, `web.scrape_spa`, `slides.render`
- Object store integration + signed-URL artifacts
- A2A Agent Card at `/.well-known/agent.json`

### Hard exit gate
**≥90% success rate on `browser.screenshot_url` and `web.scrape_spa` against MiniMax-M2.7 and Llama 3.2 7B.** This is the *defining* metric of the platform — without it nothing else matters.

### Audience
Design partners. Public alpha tag.

---

## v0.3.0 — Bridge & Client Integrations (Week 10)
**Theme:** "Register one MCP server, get every helmdeck pack."

> **Milestone:** [v0.3 — MCP Bridge & Client Integrations (Phase 3)](MILESTONES.md#milestone-v03--mcp-bridge--client-integrations-phase-3) · **Tasks:** [Phase 3](TASKS.md#phase-3--mcp-registry--bridge--client-integrations-weeks-910)

### Ships
- MCP registry with stdio/SSE/WebSocket transports
- Built-in MCP server auto-derived from the pack catalog
- **`helmdeck-mcp` bridge binary** distributed via:
  - Homebrew tap `tosin2013/helmdeck`
  - Scoop bucket `tosin2013/helmdeck`
  - npm `@helmdeck/mcp-bridge` (with `npx` postinstall)
  - OCI image `ghcr.io/tosin2013/helmdeck-mcp`
  - GitHub Releases (cosigned)
- CI smoke matrix verifying `browser.screenshot_url` from **Claude Code, Claude Desktop, OpenClaw, Gemini CLI**

### Audience
Public beta. First "helmdeck works with my agent" demo video.

---

## v0.4.0 — Desktop & Vision (Week 13)
**Theme:** "Beyond the DOM."

> **Milestone:** [v0.4 — Desktop & Vision (Phase 4)](MILESTONES.md#milestone-v04--desktop--vision-phase-4) · **Tasks:** [Phase 4](TASKS.md#phase-4--desktop-actions--vision-mode-weeks-1113)

### Ships
- Desktop Actions REST API (xdotool/scrot)
- `desktop.run_app_and_screenshot`, `doc.ocr`
- Vision-mode endpoint `POST /api/v1/sessions/{id}/vision/act`
- Reference vision packs: `vision.click_anywhere`, `vision.extract_visible_text`, `vision.fill_form_by_label`
- noVNC live viewer endpoint

### Audience
Public beta continues.

---

## v0.5.0 — Vault & Repo Packs (Week 16)
**Theme:** "Agents stop holding secrets."

> **Milestone:** [v0.5 — Vault, Repo Packs & Hardening (Phase 5)](MILESTONES.md#milestone-v05--vault-repo-packs--hardening-phase-5) · also covers [v0.5.5 — Code Edit Loop](MILESTONES.md#milestone-v055--code-edit-loop-phase-55) · **Tasks:** [Phase 5](TASKS.md#phase-5--credential-vault--repo-packs--hardening-weeks-1416), [Phase 5.5](TASKS.md#phase-55--code-edit-loop-interleaved-with-phase-5)

### Ships
- AES-256-GCM Credential Vault with placeholder-token injection
- Vault types: login, session cookies, API keys, OAuth (with refresh), SSH/git
- CDP cookie injection at session start
- HTTP gateway intercept-and-substitute for outbound agent traffic
- **`repo.fetch` and `repo.push`** (closes the canonical 2026-04-06 git-SSH failure)
- **`web.login_and_fetch`, `web.fill_form`, `slides.video`** (vault-dependent packs)
- NetworkPolicy egress allowlist + metadata IP / RFC 1918 block
- Sandbox baseline: non-root, drop-all-caps, seccomp
- OpenTelemetry GenAI semantic conventions on every span
- Trivy CRITICAL gate in CI

### Audience
Production design partners. Hardening RC.

---

## v0.6.0 — Management UI (Week 20)
**Theme:** "Operators close the weak-model gap themselves."

> **Milestone:** [v0.6 — Management UI (Phase 6)](MILESTONES.md#milestone-v06--management-ui-phase-6) · **Tasks:** [Phase 6](TASKS.md#phase-6--management-ui-weeks-1720)

### Ships
- React/Tailwind/shadcn UI embedded in Go binary
- All read-only panels: Dashboard, Sessions, AI Providers, MCP Registry, **Capability Packs**, Security Policies, Credential Vault, Audit Logs, Connect Clients
- **Model Success Rates** section on the AI Providers panel (per-(provider, model) rollup over a configurable window, backed by the new `provider_calls` aggregation table written by every gateway dispatch)
- "Connect" panel emitting per-client MCP config snippets for Claude Code, Claude Desktop, OpenClaw, Gemini CLI, and Hermes Agent

### Deferred from v0.6.0
- **Pack Authoring** (T608) — moved to v1.x (Phase 8) and clustered with T801 (WASM Executor). The pack registry is in-process today and has no publish surface; building one requires either landing a sandboxed code runtime first (WASM) or a composite-pack JSON runtime. Neither is on the v0.6.0 critical path. Operators *observe and dispatch* packs in v0.6.0; they author them in v1.x.

### Audience
Public beta — full self-service for everything except authoring custom packs.

---

## v0.8.0 — MCP Server Hosting & Pack Evolution (Phase 6.5) — ✅ Shipped 2026-04-12 {#v080}

**Theme:** "Host third-party agent infrastructure instead of rebuilding it."

> **Milestone:** [v0.8 — MCP Server Hosting & Pack Evolution (Phase 6.5)](MILESTONES.md#milestone-v08) ✅
> **Tasks:** Phase 6.5 — see [`docs/TASKS.md`](TASKS.md#phase-65--mcp-server-hosting--pack-evolution)

### Ships (36 packs total at the v0.8.0 cutover)

- **Playwright MCP bundled in the browser sidecar** ([T807a](TASKS.md#phase-65--mcp-server-hosting--pack-evolution)) — auto-attached to the running Chromium via CDP; one browser, one cookie jar, shared state with chromedp packs.
- **Firecrawl as an optional compose overlay** ([T807b](TASKS.md#phase-65--mcp-server-hosting--pack-evolution)) — `compose.firecrawl.yml`; new `web.scrape` pack returns clean markdown.
- **Docling as an optional compose overlay** ([T807c](TASKS.md#phase-65--mcp-server-hosting--pack-evolution)) — `compose.docling.yml`; new `doc.parse` pack supersedes `doc.ocr` for layout/tables.
- **Native computer-use tool routing** ([T807f](TASKS.md#phase-65--mcp-server-hosting--pack-evolution), supersedes T807d) — Anthropic / OpenAI / Gemini schemas wired through the gateway; eight new desktop REST primitives; `vision.StepNative` cross-provider executor; `EventComputerUse` audit + replay.
- **`web.test`** ([T807e](TASKS.md#phase-65--mcp-server-hosting--pack-evolution)) — natural-language browser testing via Playwright MCP accessibility tree; egress-guarded mid-test navigations.
- **`research.deep`** ([T622](TASKS.md#phase-65--mcp-server-hosting--pack-evolution)) — Firecrawl-backed research composite (search + per-source scrape + LLM synthesis with inline citations).
- **`repo.fetch` context envelope + `repo.map`** ([T622a](TASKS.md#phase-65--mcp-server-hosting--pack-evolution)) — agents orient on the first turn without chaining `fs.list`/`fs.read`; ctags-derived structural symbol map under a token budget.
- **`content.ground`** ([T623](TASKS.md#phase-65--mcp-server-hosting--pack-evolution)) — link grounding for blog posts; verbatim-substring patching skips hallucinated claims.
- **`slides.narrate`** ([T406](TASKS.md#phase-65--mcp-server-hosting--pack-evolution), moved from Phase 4) — narrated MP4 from Marp decks via ElevenLabs TTS + ffmpeg + LLM-generated YouTube metadata.
- **Provider-adapter community contributions** — Groq (PR #45) and Mistral (PR #47) adapters land alongside ([T202a](TASKS.md#phase-6--management-ui-weeks-1720)).

### Hard exit gate (met)

`scripts/validate-phase-6-5.sh` passes against a fresh stack including the Firecrawl + Docling overlays; native computer-use round-trip works against at least one frontier provider; 36 packs total.

### Audience

Public beta continues. Tag `v0.8.0` (shipped 2026-04-12). Sets up Phase 7 (Kubernetes & GA) as the next gate.

---

## v0.9.0 — Polish + plumbing (Phase 6.5+) — ✅ Shipped 2026-05-07 {#v090}

**Theme:** "Tighten what shipped before adding more."

> **Milestone:** Continuation of v0.8 / Phase 6.5 — no new milestone created. Aggregates 70 commits of post-v0.8.0 hardening.

### Ships

No new packs. No API changes. The 36-pack catalog from v0.8.0 stays the surface area. Operationally: a real install fix, public docs site at [helmdeck.dev](https://helmdeck.dev/), two community-contributed AI provider adapters (Groq, Mistral), gitleaks secret scanning, the planning-doc cross-references that were documented-but-not-implemented at v0.8.0, and the priority-label taxonomy (`priority/P0..P3`) on every issue.

See the full per-section breakdown in [`CHANGELOG.md` v0.9.0](https://github.com/tosin2013/helmdeck/blob/main/CHANGELOG.md#090---2026-05-07).

### Audience

Existing v0.8.0 operators. A direct upgrade — `git pull && make install` picks up everything (the install fix is the highest-value change for fresh deploys; existing deploys can ignore it).

---

## v0.10.0 — Content packs (Phase 6.5+) — ✅ Shipped 2026-05-09 {#v0100}

**Theme:** "Two new packs (blog + podcast), the cost story, and an upgrade procedure."

> **Repurposed slot.** The originally-planned v0.10.0 (Pack Authoring + Test Runner) didn't ship this cycle — `blog.publish` and `podcast.generate` were ready, plus the v0.9.0 → v0.10.0 doc work earned the version bump on its own. The Pack-Authoring + Test-Runner plan moves to v0.11.0 below.

### Ships

- **`blog.publish`** (#68) — Ghost Admin API + artifact-store destinations × body/prompt modes × markdown/html formats. Vault credential `ghost-admin-key`. Closes the personal-content marketplace seed.
- **`podcast.generate`** — multi-speaker (1..N) MP3 from script / prompt+model / source_url-or-source_text. Five themed system prompts (interview, debate, news-roundup, deep-dive, solo-essay) bake in podcast best practices. Day 1 ships **ElevenLabs** behind a `podcast.Engine` interface in `internal/podcast/` so future PRs add PlayHT / Hume.ai / Resemble.ai by adding one file. Vault credential `elevenlabs-key` (same as `slides.narrate`); silent-fallback when missing.
- **38 per-pack reference pages** at helmdeck.dev/reference/packs — every shipped pack on the agent-first / developer-second template, with live OpenClaw chat-UI transcripts.
- **OpenClaw transcript capture pipeline** at `scripts/oc-capture/` — `capture-oc.sh`, `capture-batch.sh`, `extract-oc-transcript.py`, `inject-transcripts.py`, plus prompt files for the three pack-doc clusters.
- **Cost-positioning blog** (`/blog/cheap-models-do-frontier-work`) + **long-form why-helmdeck reference** (`/explanation/why-helmdeck`) with five comparison tables and a reproduction recipe.
- **Operator upgrade documentation** at `/howto/upgrade-helmdeck` — pre-flight checklist, in-place Compose path, schema-migration handling, post-upgrade validation, rollback, Helm-path preview. **Closes the upgrade-docs gap** that was the maintainer's blocker for v1.0 prep.
- **SKILLS.md "Freshness contract"** + per-client "Load the agent skills" subsections for every integration doc.
- **Per-release-checklist additions** — step 6 (refresh README + cost numbers), step 7 (operator upgrade procedure smoke).

### Fixed (highlights — full list in `CHANGELOG.md`)

- `vision.click_anywhere` mechanical loop bug (#102) — per-step screenshots now reflect post-action state. **Caveat**: model-side completion-detection limitation remains; tracked at #112. Treat both vision packs as **experimental for production**.
- `repo.fetch` empty-remote infinite hang (#94)
- `fs.patch` Anthropic-edit-shape rejection (#90)
- `doc.parse` `formats: "markdown"` rejection (#91)
- OpenClaw capture pipeline cross-prompt context bleed — fresh `--session-id` per call ([#97](https://github.com/tosin2013/helmdeck/pull/97))

### Pre-Kubernetes audit issues filed (no v0.10.0 blockers)

- #108 — schema-migration cross-version test (P1, Phase 7)
- #109 — sidecar version pinning (P2, Phase 7)
- #110 — vault master-key rotation (P2, Phase 7)
- #111 — cross-version upgrade smoke in CI (P2, Phase 7)
- #112 — `vision.click_anywhere` model-side convergence research (P2)

### Audience

Production design partners + community.

---

## v0.10.2 — MCP Resources + registry description refinement — ✅ Shipped 2026-05-09 {#v0102}

**Theme:** "Browse helmdeck state as MCP resources, not just tools."

> **Closes [#44](https://github.com/tosin2013/helmdeck/issues/44).** Adds `resources/list` + `resources/read` so MCP clients can browse `helmdeck://packs` and `helmdeck://sessions` as read-only resources alongside the existing `tools/*` surface. Strictly additive — no breaking changes.

### Ships

- **MCP Resources spec implementation** — `resources/list` returns `helmdeck://packs` (always) and `helmdeck://sessions` (when a session runtime is wired); `resources/read` serves both as JSON. The `initialize` capabilities advert now includes `resources: {}`.
- **Refined registry description** — "Self-hosted MCP server: sandboxed browser, desktop, vision, code-edit packs for any agent." (was a 38-pack feature list). Leads with the value prop + self-hosted differentiator.
- **Registry submission script + workflow doc fixes** — point at the search API URL instead of the broken `/servers/<name>` web URL (registry is API-only in preview).

### Audience

Same as v0.10.1 — production design partners + community. MCP-client builders who want a browsable resource surface; everyone else can skip.

### Out of scope (deferred follow-ups)

- JWT scope filtering on resources (full #44 acceptance criteria item)
- Per-MCP-client integration tests for resource discovery

---

## v0.10.1 — MCP Registry namespace verification — ✅ Shipped 2026-05-09 {#v0101}

**Theme:** "Make the published artifacts pass the MCP Registry's namespace-verification checks."

> **Functionally identical to v0.10.0.** No pack/API/binary behavior changes. This release exists solely to add two pieces of metadata the official MCP Registry's validators need to confirm we own the `io.github.tosin2013/helmdeck` namespace. Existing v0.10.0 installs do not need to upgrade unless they specifically want the registry-listed install path.

### Ships

- **`mcpName` field on the npm package** — `@helmdeck/mcp-bridge@0.10.1`'s `package.json` now declares `"mcpName": "io.github.tosin2013/helmdeck"`. The npm validator reads this to confirm the package belongs to the registered namespace.
- **`io.modelcontextprotocol.server.name` label on the OCI image** — `ghcr.io/tosin2013/helmdeck-mcp:0.10.1` now carries the label. The OCI validator reads this to confirm namespace ownership.
- **`.github/workflows/mcp-registry.yml`** auto-publishes `.mcp/server.json` to `registry.modelcontextprotocol.io` on every `v*` tag push (also supports `workflow_dispatch` for ad-hoc runs). Authenticates via GitHub OIDC — no PAT required.

### Live registry entry

`io.github.tosin2013/helmdeck` published to the [official MCP Registry](https://registry.modelcontextprotocol.io/) as of `2026-05-09T17:13Z`, status `active`, both packages (npm + OCI) registered. Verify via the search API:

```
https://registry.modelcontextprotocol.io/v0/servers?search=io.github.tosin2013%2Fhelmdeck
```

Downstream aggregators (mcp.so, Glama, PulseMCP) ingest from the official registry on a 1–24h schedule and will appear automatically.

### Audience

Same as v0.10.0 — production design partners + community. Skip this release unless you need the registry-listed install path.

---

## v0.11.0 — podcast/slides UX hardening + image generation — ✅ Shipped 2026-05-10 {#v0110}

**Theme:** "The new content packs work — now their first-run UX matches."

> **Closes #136, #137, #138, #140, #141, #142, #143, #145, and ships the new `image.generate` pack (#71).** Adds the `helmdeck://voices` MCP resource. Closes #139 + #144 as duplicates. Defers #146 (chained image-gen integrations) to a follow-up release.

A coherent feature release driven by 9 issues filed during a v0.10.2 OpenClaw integration: silent MP3s when the credential name was wrong, hardcoded `/root/openclaw` paths, blocking Go preflight on the docker-only path, no voice discovery, no cost preview. The vault env-hydrate fix (#142) is the load-bearing piece — it root-causes the silent-fallback class of bug, not just the ElevenLabs instance.

### Ships

- **`image.generate` pack (#71)** — text → image via fal.ai's synchronous `fal.run` endpoint. Default model `fal-ai/flux/schnell` (~$0.003/image, 1-3s). 1-4 images per call. The `engine` input field is reserved for a follow-up community PR to add Replicate. Vault credential `fal-key` (with `HELMDECK_FAL_KEY` env-var fallback, auto-hydrated via #142).
- **Vault env-hydrate (#142)** — `WellKnownEnvCredentials` registry auto-imports `HELMDECK_*_API_KEY` env vars into the vault under their canonical names at startup. New `vault.Store.UpsertByName`. Wildcard ACL granted on first create; user-managed entries never clobbered. One INFO log per hydration (`vault env hydrate ok name=elevenlabs-key`).
- **`podcast.generate` + `slides.narrate` require narration by default (#138)** — pre-this-change, missing the ElevenLabs credential silently produced a silence-padded artifact. Now both packs hard-fail with `missing_credential` + an actionable message. Pass `allow_silent_output: true` to opt back into the silent path. Shared 4-step credential resolver: explicit input → vault `elevenlabs-key` → vault `elevenlabs-api-key` (back-compat alias) → `os.Getenv("HELMDECK_ELEVENLABS_API_KEY")`.
- **`helmdeck://voices` MCP resource (#143)** — exposes the operator's ElevenLabs voice catalog with 1h cache keyed on credential fingerprint. New `internal/voices/` package with `ListVoices(ctx, apiKey) → []Voice`.
- **`min_turn_duration_s` per-turn floor (#141)** — both packs gain the input (default `5s`); short TTS turns get padded with `anullsrc` so output respects the floor. `0` opts out.
- **`dry_run` + cost preview (#145)** — both packs gain `dry_run:bool`; short-circuits before TTS, returns `tts_chars` + `estimated_cost_usd`. Cost block also included in regular responses. Plan rate table covers Free/Starter/Creator/Pro/Scale; override via `HELMDECK_ELEVENLABS_RATE_PER_CHAR_USD`.
- **`slides.narrate` ffmpeg failure surfaces full stderr (#140)** — inline cap raised 512 → 4096 bytes; full stderr persisted to artifact store as `ffmpeg-stderr-segment-NNN.txt`.
- **`scripts/install.sh` `--no-build` fix (#136)** — Go preflight skipped when `--no-build` is set; unblocks the docker-only path on hosts with apt-default Go 1.22.
- **`scripts/configure-openclaw.sh` paths + auth (#137)** — new `OPENCLAW_COMPOSE_FILE` env override; `OPENCLAW_LOAD_SHELL_ENV=true` recognized so the auth-list probe doesn't false-positive.

### Audience

Operators integrating helmdeck with OpenClaw or running the content packs (`podcast.generate`, `slides.narrate`); anyone wanting `image.generate` for podcast covers / blog hero images. The credential fail-loud change (#138) is a behavior break — silent-fallback callers must add `allow_silent_output: true` to keep working. Strictly additive otherwise.

### Out of scope (deferred follow-ups)

- **#146** — chain `image.generate` into `podcast.generate.cover_image` / `slides.narrate.shield_image` + `slide_images` / `blog.publish.hero_image`. The pack lands in this release; the integration layer on top of it lands later.
- Voice-id pre-validation in `podcast.generate` / `slides.narrate` — currently agents discover voices via `helmdeck://voices` and pass the IDs verbatim; future work could pre-validate at handler entry and return `invalid_voice` synchronously.
- `speakers: {"alice":"auto"}` auto-pick mode for `podcast.generate` — pick distinct voices automatically with seed for reproducibility.
- Replicate engine for `image.generate` — flagged as a community-friendly follow-up; the `engine` input field is in the schema from day 1 so adding it is a new switch arm rather than a schema break.

### MCP Registry

The auto-publish workflow (`.github/workflows/mcp-registry.yml`) republishes the listing on `v*` tag push. After tagging, verify at `https://registry.modelcontextprotocol.io/v0/servers?search=io.github.tosin2013%2Fhelmdeck` (expect `version: 0.11.0`, `isLatest: true`).

---

## v0.12.0 — Content-pack image chaining + v1.0 install-path unblocker + pack-authoring MVP — ✅ Shipped 2026-05-12 {#v0120}

**Theme:** "Covers come for free, the install path becomes Kubernetes-ready, and pack-authoring grows up."

Bundled release across four threads that lined up after v0.11.0. Originally framed as Pack Authoring + Test Runner alone; re-scoped during the v0.11.0 retrospective to absorb #146 (unblocked by v0.11.0's #71), #158 (sibling), and #134 step 1 (v1.0 prerequisite). Planning artefact: `/root/.claude/plans/i-would-like-to-elegant-kahan.md`.

### Shipped

- **#146 — chain `image.generate` into the three content packs.** `podcast.generate` gains `cover_image: bool` → emits `cover_image_artifact_key`. `slides.render` and `slides.narrate` gain `hero_image_prompt: string` → injects inline base64 PNG (before slide 1 for render; INTO slide 1 for narrate to preserve the per-slide TTS pipeline). `blog.publish` gains `feature_image_artifact_key` + `hero_image: bool` — for Ghost, uploads via `/ghost/api/admin/images/upload/` first then stamps the URL into `feature_image`; for artifact-mode, writes a sidecar `<slug>-cover.png`. All four packs share one `RunImageGen` entrypoint (extracted from `internal/packs/builtin/image_generate.go` in PR #165's first commit) so chains don't pay for a registry round-trip.
- **#158 — `helmdeck://image-models` MCP resource.** Mirrors `helmdeck://voices` (v0.11.0). 7-model curated catalog: flux/schnell, flux/dev, flux-pro/v1.1, fast-sdxl, flux-realism, recraft-v3, ideogram/v2. Each entry has cost, p50 latency, seed/image-size support, max resolution, capability tags. New `internal/imagemodels` package. Also lands the long-overdue `fal-key` entry in `WellKnownEnvCredentials` — closes the consistency gap `image_generate.go:74` advertised since v0.11.0.
- **#134 step 1 — unified install paths (P1 v1.0-rc1 unblocker).** `deploy/compose/compose.yaml` strips `build:` blocks, pins versioned tags. New `deploy/compose/compose.build.yaml` overlay re-adds them for source-build. `scripts/install.sh --image-mode` flag pulls pre-built images, skips Go/Node/make preflight. Hosts with only Docker + `openssl` + `curl` can install the full stack. The Helm chart (v1.0-rc1) will reuse the same versioned-tag convention.
- **T606a MVP — Pack Test Runner UI.** Click a pack row in `/packs` → modal with JSON textarea + Submit. POSTs to `/api/v1/packs/{name}`, renders response (duration, cost hint, full JSON). Closes the "no UI today" gap. Schema-derived form ships v0.13.0.
- **T811 MVP — subprocess pack type.** `packs.NewCommandPack(...)` constructor + `LoadCommandPacks` dir-scanner + `HELMDECK_COMMAND_PACKS_DIR` wire-up. Pack authors can ship in any language (Python, Node, Bash, Rust) without a Go toolchain. Protocol: stdin = JSON input; stdout = JSON output; exit ≠0 → `handler_failed` with truncated stderr.

### Hard exit gates (all met)

1. **Image-mode install works on a clean VM with no Go toolchain.** Verified locally; CI smoke leg (`compose-lint` job) validates both compose layouts on every PR.
2. **All four content-pack chains produce valid output end-to-end.** ~20 new unit tests cover each chain with stubbed fal.ai/ElevenLabs/Ghost.
3. **`helmdeck://image-models` lists 7 models.** Verified in `internal/mcp/resources_test.go`.
4. **T606a UI can run `image.generate` end-to-end.** Manual click-through plus full TypeScript strict-mode build green.
5. **T811 example pack round-trips through subprocess with audit-log parity to a Go pack.** 17 new tests via the self-exec pattern.

### Slipped to v0.13.0

- **T606a schema-derived form** — JSON Schema → React form (replaces the v0.12.0 MVP textarea)
- **T811 manifest format** — typed schemas via YAML sidecar (`#173`)
- **T811 egress sandbox** — confine subprocess pack network access (`#174`)
- Marketplace UI / install CLI — bundled with v0.13.0's T810

### Slipped to v1.0-rc1

- **#134 step 2** — the Helm chart itself
- arm64 sidecar image (still blocked on Marp upstream)

### Audience

Operators wanting Kubernetes prep; community contributors who want to write packs without Go; existing users who want covers/heroes for free.

### MCP Registry

The auto-publish workflow republishes the listing on `v*` tag push. Watch for the npm-publish race condition documented in `release.yml:118-157` — workflow_dispatch the `mcp-registry.yml` after npm publish completes if the first run fails with "package not found."

---

## v0.13.0 — Marketplace beta (planned, was v0.12.0) {#v0130}

**Theme:** "Discover and install community packs."

### Ships (planned)

- **T810** — Pack marketplace registry model. `index.yaml` catalog schema, `helmdeck-pack.yaml` manifest, cosign trust verification, `HELMDECK_MARKETPLACE_URL` env var, catalog refresh endpoint.
- **T811-followup — subprocess pack manifest format ([#173](https://github.com/tosin2013/helmdeck/issues/173)).** YAML/JSON sidecar manifest declares typed `input_schema`/`output_schema`, version, author, env, timeout. The v0.12.0 MVP shipped passthrough schemas only; this completes the authoring surface.
- **T811-followup — subprocess egress sandbox ([#174](https://github.com/tosin2013/helmdeck/issues/174)).** Today subprocess egress is the host environment's responsibility (helmdeck's `EgressGuard` only intercepts in-process HTTP). Proxy-mode confinement closes the gap for the 90% case.
- **T606a-followup — schema-derived test-runner form.** Replaces the v0.12.0 MVP textarea with a real React form rendered from the pack's `BasicSchema`. Dropdowns for closed-set fields, client-side required-fields validation, typed inputs.
- **T812** — `helmdeck pack install/uninstall` CLI commands + `POST /api/v1/marketplace/install` REST endpoint. Hot-load (no restart).
- **T813** — Marketplace UI panel at `/marketplace`. Browse-by-category, search, pack detail, install/uninstall, trust badges (Core / Signed / Unsigned).
- **T814** — Community marketplace repo (`tosin2013/helmdeck-marketplace`) seeded with the worked-example pack from v0.10 + initial catalog from accepted `pack-candidate` issues.

### Audience

Operators looking for "an existing pack for X" before writing one. Designed to land before K8s so community surface area precedes enterprise surface area.

---

## v1.0.0-rc1 — Kubernetes preview (planned) {#v100rc1}

**Theme:** "Helm install works; production hardening pending."

### Hard prerequisite (must land before any rc1 work)

- **[#134](https://github.com/tosin2013/helmdeck/issues/134)** — unified install paths so `compose.yaml` and the Helm chart reference the same versioned GHCR tags (`ghcr.io/tosin2013/helmdeck:0.X.Y`) instead of the build-time-only `:dev` tag. The Helm chart cannot ship referencing `:dev` (operators have no source tree), so this gates rc1.

### Ships (planned)

- **T701** `client-go` `SessionRuntime` backend.
- **T702** Helm chart `charts/baas-platform/`.
- **T703** PostgreSQL StatefulSet sub-chart (Bitnami) with `database.external.enabled` toggle.
- **T704** Session pod template (seccomp, restartPolicy: Never, memory-backed `/dev/shm`).

Operators can install on GKE/EKS but production-hardening items (NetworkPolicy, isolation tiers, TLS, audit) are not gates.

---

## v1.0.0 — Kubernetes & GA (Week 22)
**Theme:** "Production."

> **Milestone:** [v1.0 — Kubernetes & GA (Phase 7)](MILESTONES.md#milestone-v10--kubernetes--ga-phase-7) · **Tasks:** [Phase 7](TASKS.md#phase-7--kubernetes--helm--production-hardening-weeks-2122)

### Ships
- `client-go` `SessionRuntime` backend
- **Helm chart** `charts/baas-platform/` with all toggles
- Two-namespace layout (`baas-system` / `baas-sessions`) + scoped RBAC
- Session pod template (seccomp, restartPolicy: Never, memory-backed `/dev/shm`)
- NetworkPolicies (ingress + egress)
- KEDA ScaledObject on `baas_queued_session_requests` + utilization
- `browser-pool-warmup` Deployment for cold-start elimination
- **`isolation.level`**: standard (Docker) / enhanced (gVisor) / maximum (Firecracker via RuntimeClass)
- cert-manager + Ingress-NGINX TLS termination
- OTel Collector DaemonSet
- External Secrets Operator integration
- Argo CD reference manifest in `deploy/gitops/`

### Hard exit gates
- Helm install on a fresh GKE or EKS cluster passes the same smoke matrix as Compose
- Load test: 100 concurrent sessions, 24h soak, ≤150 MB control plane footprint, ≤5 s recovery
- gVisor tier passes the smoke matrix
- External security audit clean

### Audience
**General availability.** Tag `v1.0.0`. Announce.

---

## v1.x — Post-GA Innovation Tracks

Released as feature-gated minors as they stabilize. No hard sequence.

| Version | Headline feature | ADR |
| :--- | :--- | :--- |
| v1.1 | WASM Executor for sandboxed third-party packs | 012, 024 |
| v1.2 | Four-tier Memory API (Working/Episodic/Semantic/Procedural) | 029 |
| v1.3 | Procedural→Pack promotion UI | 024, 029 |
| v1.4 | WebRTC live session streaming | 028 |
| v1.5 | WebMCP detection and preferential routing | 027 |
| v1.6 | Pre-packaged Chrome DevTools MCP / Playwright MCP entries | 006 |
| v1.7 | Firecracker production hardening (bare-metal node guidance) | 011 |
| v1.x | Lightpanda alternate browser engine | 001 |

---

## Versioning policy

- **Pre-1.0:** every minor may break compatibility; document in release notes.
- **1.0 onward:** SemVer. Breaking pack-schema changes require a new pack version under `/api/v1/packs/{name}/v{n}` (ADR 024); the previous version stays callable for at least one full minor cycle.
- **Bridge ↔ control plane:** version-pinned. The bridge logs a deprecation warning when older than the platform's minimum recommended (ADR 030).

## Distribution channels at GA

| Artifact | Channel |
| :--- | :--- |
| Control plane image | `ghcr.io/tosin2013/helmdeck:vX.Y.Z` |
| Browser sidecar image | `ghcr.io/tosin2013/helmdeck-sidecar:vX.Y.Z` |
| Helm chart | `oci://ghcr.io/tosin2013/charts/baas-platform` |
| `helmdeck-mcp` bridge | Homebrew, Scoop, npm, OCI, GH Releases |
| Compose stack | `deploy/compose/compose.yaml` in repo |
