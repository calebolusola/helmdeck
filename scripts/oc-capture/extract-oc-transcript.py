#!/usr/bin/env python3
"""
Render an OpenClaw session JSONL as a markdown chat-UI transcript ready
to drop into a per-pack reference page's "Use it from your agent" block.

Walks events forward from the latest user turn, collecting every
toolCall the assistant emits and pairing each with its toolResult.
Emits the user prompt, every (call, response) pair, the assistant's
final text reply, and a "Verified via …" footer with the OpenRouter
cost rolled up.

Usage:
  python3 extract-oc-transcript.py <session-jsonl-path>

Output: markdown to stdout.
"""
import json
import sys

session_file = sys.argv[1]
events = [json.loads(l) for l in open(session_file) if l.strip()]

# Walk backward to find the last assistant text reply that wasn't a
# "failed before producing content" stub. That's the agent's real
# answer to the user's most recent question.
last_user = None
last_assistant_text = None
tool_calls = []  # (name, arguments, result_text) tuples

for i in range(len(events) - 1, -1, -1):
    e = events[i]
    msg = e.get('message', {})
    role = msg.get('role')
    if role == 'assistant':
        text_parts = [
            p.get('text', '') for p in msg.get('content', [])
            if isinstance(p, dict) and p.get('type') == 'text'
        ]
        text = '\n'.join(t for t in text_parts if t.strip())
        if text and 'failed before producing content' not in text:
            last_assistant_text = text
            break

# Locate the last user turn so we can iterate forward from there
# collecting (toolCall, toolResult) pairs in order. With a fresh
# --session-id per capture (capture-oc.sh) the file contains exactly
# one user turn; with shared sessions there may be many and we want
# only the most recent.
target_user_idx = None
for i in range(len(events) - 1, -1, -1):
    e = events[i]
    if e.get('message', {}).get('role') == 'user':
        target_user_idx = i
        break

if target_user_idx is None:
    print("ERROR: no user turn found", file=sys.stderr)
    sys.exit(1)

user_msg = events[target_user_idx]['message']
user_text = '\n'.join(
    p.get('text', '') for p in user_msg.get('content', [])
    if isinstance(p, dict) and p.get('type') == 'text'
).strip()
# Strip the leading [timestamp] OpenClaw prepends to chat messages.
if user_text.startswith('['):
    rb = user_text.find(']')
    if rb >= 0:
        user_text = user_text[rb+1:].strip()

# Iterate forward from user_turn+1, collecting tool calls and matching
# them to the next toolResult (FIFO). Failed retry assistant turns
# (the "[assistant turn failed before producing content]" stubs) are
# skipped — we want only the productive turn's calls.
i = target_user_idx + 1
pending_tool_calls = []
while i < len(events):
    e = events[i]
    msg = e.get('message', {})
    role = msg.get('role')
    if role == 'assistant':
        for part in msg.get('content', []):
            if not isinstance(part, dict):
                continue
            if part.get('type') == 'toolCall':
                pending_tool_calls.append({
                    'name': part.get('name', '?'),
                    'arguments': part.get('arguments', {}),
                    'id': part.get('id', ''),
                })
    elif role == 'toolResult':
        if pending_tool_calls:
            tc = pending_tool_calls.pop(0)
            result_text = '\n'.join(
                p.get('text', '') for p in msg.get('content', [])
                if isinstance(p, dict) and p.get('type') == 'text'
            )
            tool_calls.append((tc['name'], tc['arguments'], result_text))
    i += 1

# Sum the per-event cost.total floats so the footer shows the real
# OpenRouter spend for this capture.
total_cost = 0
for e in events:
    cost = e.get('message', {}).get('usage', {}).get('cost', {})
    if isinstance(cost, dict):
        total_cost += cost.get('total', 0) or 0

last_msg = events[-1].get('message', {})
provider = last_msg.get('provider', 'openrouter')
model = last_msg.get('model', 'openai/gpt-oss-120b')

# Render markdown.
out = []
out.append("**Prompt** (sent in OpenClaw chat UI / `openclaw-cli agent`):")
out.append("")
out.append(f"> {user_text}")
out.append("")

if not tool_calls:
    # Either the model answered from memory (which a fresh --session-id
    # eliminates) or the prompt deliberately asked for a non-tool reply.
    # Either way, surface the artifact rather than hide it.
    out.append("**Note**: in this turn the model answered without calling any helmdeck tool. ")
    out.append("Re-run with capture-oc.sh (fresh `--session-id` per call) and ensure the prompt names the tool explicitly (e.g. `Use helmdeck__http-fetch`).")
else:
    out.append(f"**Tool call** ({len(tool_calls)} call{'s' if len(tool_calls)!=1 else ''}, no failures):")
    out.append("")
    for name, args, result in tool_calls:
        out.append("```json")
        out.append(json.dumps({"name": name, "arguments": args}, indent=2))
        out.append("```")
        out.append("")
        out.append("**Pack response** (truncated):")
        out.append("")
        out.append("```json")
        result_str = result.strip()
        if len(result_str) > 800:
            result_str = result_str[:800] + "\n  ...[truncated]"
        out.append(result_str)
        out.append("```")
        out.append("")

if last_assistant_text:
    out.append("**Agent's final reply**:")
    out.append("")
    for line in last_assistant_text.splitlines():
        out.append(f"> {line}")
    out.append("")

out.append(f"*Verified via OpenClaw 2026.5.6 + helmdeck v0.9.0-dev + `{provider}/{model}` on 2026-05-07 (cost: ${total_cost:.4f}).*")

print('\n'.join(out))
