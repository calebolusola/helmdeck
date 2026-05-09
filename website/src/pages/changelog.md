---
title: Changelog
description: Release history for helmdeck.
---

# Changelog

All notable changes to helmdeck are documented here. The format follows
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) starting
at v1.0.0; pre-1.0 minor versions may break compatibility (documented per release).

For the forward-looking *release plan* — what is targeted for upcoming versions
and the hard exit gates for each — see
[`docs/RELEASES.md`](https://github.com/tosin2013/helmdeck/blob/main/docs/RELEASES.md).

## [Unreleased]

## [0.10.2] - 2026-05-09

A small patch release that ships the **MCP Resources** surface (closes [#44](https://github.com/tosin2013/helmdeck/issues/44)) plus a refined registry-listing description. Functionally additive only; no breaking changes.

### Added

- **MCP Resources** (`#44`) — the MCP server now serves `resources/list` and `resources/read` per the 2024-11-05 spec, alongside the existing `tools/list` / `tools/call`. Two read-only resources surface today:
  - `helmdeck://packs` — the live pack catalog (every registered pack with its input schema). Equivalent to `tools/list` as a browsable resource.
  - `helmdeck://sessions` — live session list (id, status, image, created_at). Wired only when the control plane has an active session runtime; safely omitted otherwise.
  - The `initialize` response now declares the `resources` capability so MCP clients discover the new surface automatically.
  - 7 unit tests cover both happy paths, the missing-runtime fallback, the unknown-URI error, lister error propagation, and the capability declaration.

### Changed

- **Registry description** now reads *"Self-hosted MCP server: sandboxed browser, desktop, vision, code-edit packs for any agent."* (was "38 capability packs (browser, desktop, vision, repo, fs, slides, podcast) for MCP agents."). Leads with the value proposition + self-hosted differentiator instead of the feature list.
- **Registry submission script + workflow** corrected to point at the search API URL — the registry has no human-facing web UI today, only the metadata API. Was a pre-1.0 documentation bug from the v0.10.1 cycle.

### Operator notes

- **No action required for existing v0.10.1 installs** — MCP Resources is purely additive (new methods don't break existing tools/* clients). Upgrade if you want to expose `helmdeck://sessions` and `helmdeck://packs` to your agent for browsing.
- **Out of scope for #44** (deferred): JWT scope filtering on resources, per-MCP-client integration tests. Tracked as follow-ups; the spec implementation is complete and the 7 unit tests cover the surface.

---

## [0.10.1] - 2026-05-09

A patch release that completes helmdeck's listing on the [official MCP Registry](https://registry.modelcontextprotocol.io/). The v0.10.0 attempt failed namespace verification because two pieces of metadata weren't yet declared on the published artifacts — this release adds them. Functionally identical to v0.10.0; no pack/API/binary behavior changes.

### Fixed

- **`@helmdeck/mcp-bridge` npm package** now declares `mcpName: "io.github.tosin2013/helmdeck"` in its `package.json`. The MCP Registry's npm validator reads this field to confirm the package belongs to the registered namespace; without it, registry submission failed with `NPM package '@helmdeck/mcp-bridge' is missing required 'mcpName' field`.
- **`ghcr.io/tosin2013/helmdeck-mcp` OCI image** now carries the `io.modelcontextprotocol.server.name="io.github.tosin2013/helmdeck"` label. The OCI validator reads this label to confirm namespace ownership; the v0.10.0 image lacked it.

### Operator notes

- **No action required for existing v0.10.0 installs.** The bridge binary, control plane, and all 38 packs are unchanged. Skip this release unless you specifically need the registry-listed install path.
- **Registry entry goes live on tag push.** [`.github/workflows/mcp-registry.yml`](https://github.com/tosin2013/helmdeck/blob/main/.github/workflows/mcp-registry.yml) auto-fires; verify via the search API at `https://registry.modelcontextprotocol.io/v0/servers?search=io.github.tosin2013%2Fhelmdeck` (the registry is API-only in preview — there is no human-facing web UI; browse downstream aggregators like mcp.so, Glama, and PulseMCP instead).

---

## [0.10.0] - 2026-05-09

A "content packs" release. Two new packs land — **`blog.publish`** for posting to Ghost or stuffing markdown/HTML into the artifact store, and **`podcast.generate`** for multi-speaker podcast MP3s via a pluggable TTS engine. The capture pipeline ships in-repo, the upgrade procedure is documented for the first time, and the README now opens with the quantified cost-positioning argument the platform earned by shipping the per-pack reference work. Pack count: **36 → 38**.

The originally-planned v0.10.0 theme (Pack Authoring + Test Runner) slips to v0.11.0 — the work didn't happen this cycle, the slot got repurposed because the new packs were ready.

### Added

- **`blog.publish` pack** (#68 via [#103](https://github.com/tosin2013/helmdeck/pull/103)) — publish to a Ghost installation (live Admin API) OR render markdown/HTML to the helmdeck artifact store. Two body modes (agent-supplied OR prompt+model the pack expands). Goldmark added to `go.mod` for the markdown→HTML shim. Ghost JWT minted inline via `golang-jwt/jwt/v5` (5-min HS256, audience `/admin/`).
- **`podcast.generate` pack** ([#106](https://github.com/tosin2013/helmdeck/pull/106)) — produce a 1..N speaker podcast MP3 from a script, a prompt, or long-form content (URL/text → LLM converts). Three input modes (script / prompt+model / source_*+model). Five themed system prompts: `interview`, `debate`, `news-roundup`, `deep-dive`, `solo-essay`. Day 1: **ElevenLabs** behind a `podcast.Engine` interface so future PRs (PlayHT, Hume.ai, Resemble.ai) slot in by adding a new file under `internal/podcast/`. Vault credential `elevenlabs-key` (same as `slides.narrate`); silent-fallback when missing. Optional `cover_image_prompt` output for downstream image-gen packs.
- **38 per-pack reference pages** at [helmdeck.dev/reference/packs](https://helmdeck.dev/reference/packs) — every shipped pack on the agent-first / developer-second template, with live OpenClaw chat-UI transcripts embedded alongside `curl` developer references. (PR-A [#83](https://github.com/tosin2013/helmdeck/pull/83) + PR-B [#95](https://github.com/tosin2013/helmdeck/pull/95) + PR-C [#101](https://github.com/tosin2013/helmdeck/pull/101).) Closes #51, #53, #54, #55, #56, #58, #59, #60, #61, #62, #63, #64.
- **OpenClaw transcript capture pipeline** at `scripts/oc-capture/` ([#97](https://github.com/tosin2013/helmdeck/pull/97) + [#104](https://github.com/tosin2013/helmdeck/pull/104)) — three scripts (`capture-oc.sh`, `extract-oc-transcript.py`, `inject-transcripts.py`), a generic `capture-batch.sh` driver, and prompt files for the three pack-doc clusters.
- **Cost-positioning blog + long-form reference** ([#99](https://github.com/tosin2013/helmdeck/pull/99)) — `website/blog/2026-05-08-cheap-models-do-frontier-work.md` + `docs/explanation/why-helmdeck.md` with five per-task comparison tables vs. Anthropic Computer Use, OpenAI Operator, Browser-use, Cursor, Aider, Unstructured.io, LlamaParse, Pictory. Includes a "Run the comparison yourself" reproduction recipe + community-contribution invitation.
- **Operator upgrade documentation** at [`docs/howto/upgrade-helmdeck.md`](https://helmdeck.dev/howto/upgrade-helmdeck) ([#107](https://github.com/tosin2013/helmdeck/pull/107)) — pre-flight checklist, in-place Compose-stack upgrade, schema-migration handling, post-upgrade validation, rollback, Kubernetes/Helm path preview.
- **SKILLS.md gains a "Freshness contract" section** ([#98](https://github.com/tosin2013/helmdeck/pull/98)) — teaches agents to re-call stateful packs when state may have changed since the last call. Plus per-client "Load the agent skills" subsections for every integration doc (Claude Code via CLAUDE.md, Claude Desktop via Projects, Gemini CLI via GEMINI.md, Hermes via system_prompt_file).
- **Per-release-checklist additions** in `docs/RELEASES.md`: step 6 (refresh README + cost numbers per release, [#100](https://github.com/tosin2013/helmdeck/pull/100)), step 7 (operator upgrade procedure smoke, [#107](https://github.com/tosin2013/helmdeck/pull/107)).

### Fixed

- **`vision.click_anywhere` mechanical loop bug** (#102 via [#105](https://github.com/tosin2013/helmdeck/pull/105)) — per-step screenshots now genuinely reflect post-action desktop state. Two changes: `Step` and `StepNative` thread prior-turn actions into the next user message as textual history, and a 250 ms post-dispatch wait gives Xvfb time to repaint. Same fix applies to `vision.fill_form_by_label`. Verified live: per-step PNG artifacts now have **distinct file sizes** between iterations (vs. PR-B baseline where every step's bytes were identical because Xvfb hadn't repainted before scrot fired). **However**, the model-side completion-detection limitation remains — the model still rarely emits `done` on real tasks even when the click visibly landed. **Tracked separately at [#112](https://github.com/tosin2013/helmdeck/issues/112)** for follow-up research (try gpt-4o vs. haiku-4.5, native computer-use schema, two-shot verification). Treat `vision.click_anywhere` as **experimental for production workflows** until #112 lands an answer.
- **`repo.fetch` empty-remote infinite hang** (#94 via [#96](https://github.com/tosin2013/helmdeck/pull/96)) — `git ls-remote --heads` runs first; pack errors fast with `invalid_input: remote has no branches; push at least one commit before cloning`.
- **`fs.patch` Anthropic-edit-shape rejection** (#90 via [#93](https://github.com/tosin2013/helmdeck/pull/93)) — both `{search, replace}` and `{edits: [{oldText, newText}]}` shapes accepted.
- **`doc.parse` `formats: "markdown"` rejection** (#91 via [#93](https://github.com/tosin2013/helmdeck/pull/93)) — `markdown` aliases `md`; both work.
- **OpenClaw capture pipeline cross-prompt context bleed** ([#97](https://github.com/tosin2013/helmdeck/pull/97)) — every `capture-oc.sh` invocation now mints a fresh `--session-id`. Side-effect: per-call cost dropped ~140× (no 280-event session bloat shipped on every turn).
- **Vision pack loops now check `ctx.Err()`** (in [#105](https://github.com/tosin2013/helmdeck/pull/105)) — cancelled callers exit cleanly instead of spinning to `max_steps`.
- **`vision.fill_form_by_label` parity fix** ([#105](https://github.com/tosin2013/helmdeck/pull/105)) — now records per-step PNG artifacts (parity with `click_anywhere`).

### Changed

- **Pack count: 36 → 38** (`blog.publish` + `podcast.generate`)
- **`README.md`** opens with the quantified cost-positioning argument ($0.07 Phase 5.5 loop on `gpt-oss-120b` vs $0.30+ on Sonnet via Cursor) plus a 4-row comparison table; "other 99%" framing kept as the follow-on paragraph
- **Homepage tagline** rewritten from "Self-hosted AI agent platform for small open-weight models" to lead with the cost angle
- **`docs/integrations/SKILLS.md`** picks up the Freshness contract, expanded "How to load" subsection with per-client instructions, "Blog" and "Podcast" catalog entries, and the pack count bump

### Operator notes

- **Upgrade procedure**: `git fetch && git checkout v0.10.0 && make sidecars && make install`. See [`/howto/upgrade-helmdeck`](https://helmdeck.dev/howto/upgrade-helmdeck) for the full pre-/post-upgrade checklist.
- **Schema migrations**: auto-applied on `store.Open`. Cross-version smoke is tracked in [#108](https://github.com/tosin2013/helmdeck/issues/108) (P1).
- **OpenClaw skill refresh**: re-run `./scripts/configure-openclaw.sh` after pulling so the new SKILL.md (with podcast/blog entries + Freshness contract) lands in the OpenClaw container.
- **No breaking changes** to existing pack input/output schemas. All `### Added` work is additive; all `### Fixed` items improve observable behavior in agents' favor.
- **Pre-Kubernetes audit issues filed**: [#108](https://github.com/tosin2013/helmdeck/issues/108) (schema-migration cross-version test, P1), [#109](https://github.com/tosin2013/helmdeck/issues/109) (sidecar version pinning, P2), [#110](https://github.com/tosin2013/helmdeck/issues/110) (vault master-key rotation, P2), [#111](https://github.com/tosin2013/helmdeck/issues/111) (cross-version upgrade smoke in CI, P2). All tagged Phase 7; none block v0.10.0.
- **Known limitation**: `vision.click_anywhere` and `vision.fill_form_by_label` are **experimental** — the underlying loop fix in #105 works mechanically (screenshots progress per turn) but the vision model rarely emits `done` on real tasks. See [#112](https://github.com/tosin2013/helmdeck/issues/112) for the research track. Use at your own risk in production workflows; prefer `web.test` (Playwright MCP, deterministic) for browser-automation goals where possible.

## [0.9.0] - 2026-05-07

A "polish + plumbing" release. No new packs and no API changes — the 36
packs from v0.8.0 stay the surface area. What landed: a real install
fix that was breaking first-session sessions, a public docs site at
helmdeck.dev, two community-contributed AI provider adapters, secret
scanning in CI, and the planning-doc cross-references that were
documented-but-not-implemented at v0.8.0.

### Added
- **Documentation site** at [helmdeck.dev](https://helmdeck.dev/) —
  Docusaurus 3, Diataxis-organized (Tutorials / How-to / Reference /
  Explanation), deployed to Vercel with auto-preview on PRs. Search via
  `@easyops-cn/docusaurus-search-local`. SEO-tuned for Google Search
  Console submission: explicit titles, OG social card, robots.txt,
  sitemap with per-route priority bumps, schema.org/WebSite +
  FAQPage JSON-LD.
- **Install tutorials** — `docs/tutorials/install-cli.md` (10-minute
  walkthrough from `git clone` to running stack) and
  `docs/tutorials/install-ui-walkthrough.md` (panel-by-panel UI tour).
- **Troubleshooting how-to** — `docs/howto/troubleshoot-install.md`
  with FAQPage schema covering 10 known sharp edges (502 on first
  session, GHCR pull failures, lost admin password, etc.).
- **Per-pack documentation framework** — `docs/reference/packs/` with
  template + fully-written browser family (`browser.screenshot_url`,
  `browser.interact`). 12 family-tracking issues opened for community
  to pick up the remaining 34 packs.
- **OSS hygiene files** at repo root — `CHANGELOG.md`, `SECURITY.md`
  (90-day disclosure window), `CODE_OF_CONDUCT.md` (Contributor
  Covenant 2.1).
- **GitHub priority taxonomy** — `priority/P0..P3` labels applied to
  all 39 open issues. P1 cohort (14 items) is the next-release
  shortlist.
- **`docs/sitemap.xml`** — documcp-generated source-side sitemap for
  link audits and search-engine submission tracking, separate from
  Docusaurus's runtime sitemap.
- **Custom logo** — helm-wheel + H letterform mark, light/dark
  variants, SVG favicon. Replaced the scaffolded Docusaurus brand
  assets.
- **Provider adapters via community PRs** — Groq (PR #45 by @Dev-31)
  and Mistral (PR #47, resolved from @vijit-vishnoi's PR #46) both
  ride the `HELMDECK_{PROVIDER}_API_KEY[_FILE]` / `_BASE_URL` /
  `_MODELS` env-var contract introduced for OpenRouter in v0.8.0.

### Changed
- **Planning docs** (`RELEASES.md`, `MILESTONES.md`, `TASKS.md`) are
  now cross-linked. Every release has a Milestone + Tasks pointer;
  every milestone has a Ships-in pointer; the v0.8.0 RELEASES section
  was added (was missing). 19 task IDs that lived in MILESTONES
  without rows in TASKS got promoted into proper rows.
- README's install section links to the new tutorial pages.
- Trivy CI scan scope narrowed to `scanners: vuln,misconfig`. Action
  pin bumped 0.28.0 → 0.35.0.

### Fixed
- **Install bug** — `docker compose up -d --build` only builds
  services with a `build:` clause, so published images (Garage, the
  GHCR-published sidecar tag) weren't pulled before stack-up. Result:
  first session calls hung on a 30-second timeout. Fix: new
  `compose_pull` step in `scripts/install.sh` runs `docker compose
  pull --ignore-buildable` between sidecar build and `compose up`,
  fast-failing on network/proxy issues with an actionable error. The
  `sidecar-warm` service no longer swallows pull failures with
  `|| true`.
- **CI race** — `TestBridgeRoundTrip`'s shared `bytes.Buffer` between
  the test goroutine and the bridge writer. Wrapped in a
  `sync.Mutex`-guarded `safeBuffer`. Production code unchanged.
- **`vercel.json`** — `cleanUrls: true` added so `/PACKS` resolves to
  `/PACKS.html` (matched to Docusaurus's `trailingSlash: false`).

### Security
- **Gitleaks** secret-scanning CI workflow on every push + PR. Runs
  via `gitleaks/gitleaks-action@v2` with `fetch-depth: 0` so the
  scanner walks full history. Allowlist covers stable dev credentials
  in `deploy/compose/garage.toml` (file header already documents
  these as override-in-production).
- **`serialize-javascript`** bumped 6.0.2 → 7.0.5 via npm `overrides`
  to address GHSA-5c6j-r48x-rmvq (HIGH) and CVE-2026-34043 (MEDIUM).
  Both shipped as transitive deps in @docusaurus/bundler.

### Developer experience
- **`make check`** target wraps `vet + race test + build` — exactly
  what CI's `vet + test + build` job runs. Plus `make install-hooks`
  to wire an opt-in `pre-push` hook.

## [0.8.0] - 2026-04-12

### Added
- 36 capability packs total (browser, web, research, slides, GitHub, repo,
  filesystem, shell, HTTP, document, desktop, vision, language families).
- Phase 6.5 validation script (`scripts/validate-phase-6-5.sh`).
- Multi-provider AI gateway adapters: Groq, Mistral.
- gitleaks secret-scanning CI workflow with allowlist.

### Changed
- README leads with the weak-model success story; v0.8.0 + 36-pack catalog
  refresh.
- Trivy CI scan scope narrowed to vuln+misconfig (secrets owned by gitleaks).

## [0.5.1] - 2026-04-08

### Fixed
- npm trusted publishing: bump npm + add `--provenance` so
  `@helmdeck/mcp-bridge` releases include attestations.

## [0.5.0] - 2026-04-08

### Added
- AES-256-GCM Credential Vault with placeholder-token injection
  (login, session cookies, API keys, OAuth-with-refresh, SSH/git).
- CDP cookie injection at session start.
- HTTP gateway intercept-and-substitute for outbound agent traffic.
- `repo.fetch`, `repo.push`, `web.login_and_fetch`, `web.fill_form`,
  `slides.video` packs (vault-dependent).
- NetworkPolicy egress allowlist + metadata IP / RFC 1918 block.
- Sandbox baseline: non-root, drop-all-caps, seccomp.
- OpenTelemetry GenAI semantic conventions on every span.
- Trivy CRITICAL gate in CI.

## [0.3.0] - 2026-04-08

### Added
- MCP registry with stdio/SSE/WebSocket transports.
- Built-in MCP server auto-derived from the pack catalog.
- `helmdeck-mcp` bridge binary distributed via Homebrew, Scoop, npm
  (`@helmdeck/mcp-bridge`), GHCR OCI image, and signed GitHub Releases.
- CI smoke matrix verifying `browser.screenshot_url` from Claude Code,
  Claude Desktop, OpenClaw, and Gemini CLI.

### Fixed
- `release.yml`: gate binary jobs to push events only.

## [0.2.0] - 2026-04-08

### Added
- OpenAI-compatible `/v1/chat/completions` and `/v1/models`.
- Provider adapters: Anthropic, Gemini, OpenAI, Ollama, Deepseek.
- Encrypted key store with rotation API.
- Fallback routing rules (rate-limit / error / timeout triggers).
- Pack Execution Engine with input/output schema validation.
- Typed error code enforcement (closed set per pack).
- Pack registry with versioned dispatch.
- Three reference packs: `browser.screenshot_url`, `web.scrape_spa`,
  `slides.render`.
- Object store integration with signed-URL artifacts.
- A2A Agent Card at `/.well-known/agent.json`.

### Hardware exit gate met
- ≥90% success rate on `browser.screenshot_url` and `web.scrape_spa`
  against MiniMax-M2.7 and Llama 3.2 7B.

## [0.1.1] - 2026-04-07

### Fixed
- `sidecar.yml`: publish amd64 only until Marp ships an arm64 tarball.

## [0.1.0] - 2026-04-07

### Added
- Go control plane binary (Gin + chromedp + Docker SDK).
- Browser sidecar image with Chromium, Marp, Tesseract, ffmpeg, xdotool,
  Xvfb, XFCE4, noVNC.
- Ephemeral session lifecycle (`POST /api/v1/sessions` …
  `DELETE /api/v1/sessions/{id}`).
- CDP REST endpoints: navigate, extract, screenshot, execute, interact.
- JWT bearer auth on every endpoint.
- Audit log (write-only).
- Single-node Compose deployment (`deploy/compose/compose.yaml`).
- `make smoke` end-to-end harness in CI.

[Unreleased]: https://github.com/tosin2013/helmdeck/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/tosin2013/helmdeck/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/tosin2013/helmdeck/compare/v0.5.1...v0.8.0
[0.5.1]: https://github.com/tosin2013/helmdeck/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/tosin2013/helmdeck/compare/v0.3.0...v0.5.0
[0.3.0]: https://github.com/tosin2013/helmdeck/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/tosin2013/helmdeck/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/tosin2013/helmdeck/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/tosin2013/helmdeck/releases/tag/v0.1.0
