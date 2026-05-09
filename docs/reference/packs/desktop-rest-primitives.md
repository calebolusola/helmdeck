---
title: desktop.* REST primitives (16 endpoints)
description: 16 deterministic desktop-action endpoints under /api/v1/desktop/*. Mirror Anthropic computer-use schema and Gemini computer-use conventions. xdotool/scrot under the hood; pixel-coordinate input on a fixed 1920×1080 Xvfb display.
keywords: [helmdeck, desktop, xdotool, scrot, computer use, anthropic, gemini, click, type, screenshot, REST]
---

# `desktop.*` REST primitives (16 endpoints)

The deterministic counterpart to the [`vision.*` packs](./vision/click-anywhere.md). Where vision packs ask an LLM "click the Sign In button" and let the model figure out the coordinates, the `desktop.*` REST endpoints do exactly what you tell them at exactly the pixel coordinates you supply. Use them when the agent already knows where things are (from a prior screenshot, a `vision.extract_visible_text` pass, or a deterministic UI test fixture).

These are **not packs** — they're plain REST endpoints under `/api/v1/desktop/*`, exposed alongside the pack catalog. Most MCP clients won't see them as tools; they're called directly via `http.fetch` (with the helmdeck JWT) when the agent needs deterministic control. Helmdeck's vision packs use them internally — `vision.click_anywhere` is essentially a screenshot loop wrapped around `/api/v1/desktop/click`.

The endpoint vocabulary mirrors **Anthropic's `computer_20251124`** schema and **Gemini's computer-use** conventions, so an agent fluent in either spec can drive helmdeck's desktop with minimal translation. Coordinates are pixel-based on the fixed **1920×1080 Xvfb display** (`DISPLAY=:99`).

## Setup prerequisite

Every endpoint needs a session in desktop mode. Create one via [`desktop.run_app_and_screenshot`](./desktop/run-app-and-screenshot.md) (which sets `HELMDECK_MODE=desktop` automatically), or any `vision.*` pack. Then pass that session's `id` as `session_id` on every desktop endpoint call.

All requests need the helmdeck JWT in `Authorization: Bearer <jwt>`.

## The 16 endpoints

### Capture

#### `POST /api/v1/desktop/screenshot`

Capture the full visible desktop as a PNG.

| Request | Response |
|---|---|
| `{"session_id": "..."}` | `image/png` bytes (capped at 32 MiB) |

Implementation: `scrot -o /tmp/<rand>.png && cat … && rm`.

#### `POST /api/v1/desktop/zoom`

Crop a region of the desktop and upscale it to 1024×1024 for vision-model inspection.

| Request | Response |
|---|---|
| `{"session_id": "...", "x1": 0, "y1": 0, "x2": 800, "y2": 600}` | `image/png` bytes (capped at 32 MiB) |

Implementation: `scrot` + `convert -crop <w>x<h>+<x>+<y> -resize 1024x1024 …`. Useful for zeroing in on a small UI element before deciding what to click.

### Mouse

#### `POST /api/v1/desktop/click`

Click once at the given pixel coordinates.

| Request | Response |
|---|---|
| `{"session_id": "...", "x": 376, "y": 69, "button": "left"}` | `{"ok": true, "x": 376, "y": 69, "button": "1"}` |

`button` is `"left"` (default), `"middle"`, or `"right"`. Implementation: `xdotool mousemove <x> <y> click <btn>`.

#### `POST /api/v1/desktop/double_click`

Two rapid clicks. `button` optional (default `"left"`). Implementation: `xdotool mousemove <x> <y> click --repeat 2 --delay 100 <btn>`.

#### `POST /api/v1/desktop/triple_click`

Three rapid clicks (text-selection idiom). Same shape as `double_click`. Implementation: `--repeat 3`.

#### `POST /api/v1/desktop/modifier_click`

Click while holding modifier keys.

| Request |
|---|
| `{"session_id": "...", "x": 100, "y": 200, "button": "left", "modifiers": ["shift", "ctrl"]}` |

Modifiers are whitelist-validated (`shift`, `ctrl`, `alt`, `super`). Implementation chains `keydown … mousemove … click … keyup …` in one xdotool invocation so the sequence is atomic.

#### `POST /api/v1/desktop/mouse_move`

Move the mouse without clicking.

| Request | Response |
|---|---|
| `{"session_id": "...", "x": 100, "y": 200}` | `{"ok": true, "x": 100, "y": 200}` |

#### `POST /api/v1/desktop/drag`

Press at one point, drag to another, release. Atomic (one xdotool invocation; no Exec lag mid-drag).

| Request |
|---|
| `{"session_id": "...", "start_x": 100, "start_y": 100, "end_x": 400, "end_y": 300, "button": "left"}` |

#### `POST /api/v1/desktop/scroll`

Scroll wheel events at an optional point.

| Request |
|---|
| `{"session_id": "...", "x": 500, "y": 400, "direction": "down", "amount": 5}` |

`direction` is `up`, `down`, `left`, or `right`. `amount` capped at 50 repeats. Implementation: scroll wheel buttons (`4`=up, `5`=down, `6`=left, `7`=right) clicked `amount` times.

### Keyboard

#### `POST /api/v1/desktop/type`

Type literal text — character by character — into whichever window currently has focus.

| Request | Response |
|---|---|
| `{"session_id": "...", "text": "hello world", "delay_ms": 0}` | `{"ok": true, "length": 11}` |

`delay_ms` between keystrokes (default 0 = as fast as possible). Implementation: `xdotool type [--delay <ms>] -- <text>`. The `--` prevents text starting with `-` from being parsed as a flag.

#### `POST /api/v1/desktop/key`

Press a named key or chord. Use xdotool's keysym syntax — **not** literal characters.

| Request | Response |
|---|---|
| `{"session_id": "...", "keys": "Return"}` | `{"ok": true, "keys": "Return"}` |
| `{"session_id": "...", "keys": "ctrl+a"}` | `{"ok": true, "keys": "ctrl+a"}` |

Common keysyms: `Return`, `Tab`, `Escape`, `BackSpace`, `Delete`, `Up`/`Down`/`Left`/`Right`, `Home`, `End`, `Page_Up`, `Page_Down`, `F1`–`F12`, `ctrl+c`, `alt+Tab`, `super+l`.

### Process / window

#### `POST /api/v1/desktop/launch`

Launch an arbitrary command (detached, survives the RPC).

| Request | Response |
|---|---|
| `{"session_id": "...", "command": "xterm", "args": ["-bg", "black"]}` | `{"ok": true, "command": "xterm"}` |

Same shape as the [`desktop.run_app_and_screenshot`](./desktop/run-app-and-screenshot.md) pack but **without** the post-launch screenshot. Use this when you'll capture later via `desktop.screenshot` separately (e.g. you want to launch and then `wait` before screenshotting).

Implementation: `nohup setsid <cmd> <args> >/dev/null 2>&1 &`.

#### `GET /api/v1/desktop/windows`

List visible X11 windows on the session.

| Request (query string) | Response |
|---|---|
| `?session_id=...` | `{"windows": [{"id": "12345", "pid": 678, "name": "xterm"}, …], "count": 3}` |

Capped at 1024 windows. Implementation: `xdotool search ... getwindowname ... getwindowpid ...`.

#### `POST /api/v1/desktop/focus`

Activate (focus) a specific window by its X11 ID.

| Request | Response |
|---|---|
| `{"session_id": "...", "window_id": "12345"}` | `{"ok": true, "window_id": "12345"}` |

`window_id` must be the numeric X11 ID returned by `/windows` — whitelist-validated to numerics only. Implementation: `xdotool windowactivate --sync <id>`.

### Timing

#### `POST /api/v1/desktop/wait`

Sleep inside the session (timing observed from sidecar perspective, not host).

| Request | Response |
|---|---|
| `{"session_id": "...", "seconds": 2.5}` | `{"ok": true, "seconds": 2.5}` |

Capped at 30 seconds. Use sparingly — `xdotool` itself has `--sync` for window activations and `--delay` for typing/clicking, which are usually better than blanket waits.

### Status

#### `POST /api/v1/desktop/agent_status`

Health check / session status. **Implementation pending** — present in SKILLS.md as part of the documented vocabulary; the endpoint may not be wired yet on every helmdeck rev. Treat as best-effort until verified.

## Calling them from an agent

MCP clients don't expose these as tools — but the helmdeck `http.fetch` pack does. Pattern:

```jsonc
// Inside a desktop-mode chained workflow:
{"name": "helmdeck__http-fetch", "arguments": {
  "url": "http://localhost:3000/api/v1/desktop/click",
  "method": "POST",
  "headers": {"Content-Type": "application/json"},
  "body": "{\"session_id\":\"sess-9f\",\"x\":376,\"y\":69,\"button\":\"left\"}"
}}
```

The agent JWT is auto-injected by `http.fetch` for `localhost:3000` calls. For sequences of clicks/types in a known order, this is faster than `vision.click_anywhere` because no per-step LLM/vision round-trip.

## Loop shape

```
desktop.screenshot  → look at the pixels (vision model OR human operator)
                    ↓
                    decide next action from coordinates
                    ↓
desktop.click / type / key / scroll
                    ↓
                    repeat until done
```

For natural-language targeting (`"click the blue Sign In button"`), [`vision.click_anywhere`](./vision/click-anywhere.md) wraps that screenshot-to-coordinates loop for you. For known-good coordinates, drive these endpoints directly.

## See also

- Source: [`internal/api/desktop.go`](https://github.com/tosin2013/helmdeck/blob/main/internal/api/desktop.go).
- ADR 029 — Desktop primitives.
- ADR 035 §2026 — Native computer-use tool routing (T807f).
- Companion: [`desktop.run_app_and_screenshot`](./desktop/run-app-and-screenshot.md), [`vision.click_anywhere`](./vision/click-anywhere.md), [`vision.extract_visible_text`](./vision/extract-visible-text.md), [`vision.fill_form_by_label`](./vision/fill-form-by-label.md).
- Anthropic `computer_20251124` schema — the loop shape this vocabulary mirrors.
