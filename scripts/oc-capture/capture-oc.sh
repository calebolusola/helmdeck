#!/usr/bin/env bash
# capture-oc.sh — drive an OpenClaw agent turn against a fresh session,
# fetch the resulting session JSONL from the OpenClaw container, render
# a chat-UI-style transcript, and emit the markdown to stdout.
#
# Used to populate the "Use it from your agent (OpenClaw chat-UI worked
# example)" block on each per-pack reference page.
#
# Usage: capture-oc.sh "<prompt>"
#
# Environment:
#   OPENCLAW_COMPOSE   path to the OpenClaw docker-compose.yml
#                      (default: /root/openclaw/docker-compose.yml)
#   OPENCLAW_GATEWAY   gateway container name
#                      (default: openclaw-openclaw-gateway-1)
set -euo pipefail
PROMPT="$1"
OPENCLAW_COMPOSE="${OPENCLAW_COMPOSE:-/root/openclaw/docker-compose.yml}"
OPENCLAW_GATEWAY="${OPENCLAW_GATEWAY:-openclaw-openclaw-gateway-1}"

TMPJSON=$(mktemp)
trap 'rm -f "$TMPJSON" "$TMPJSON.session"' EXIT

# Use a FRESH session id per invocation. Without this, each capture
# inherits the prior turn's context and the model recalls answers from
# memory instead of calling the tool — yielding "agent answered without
# calling any helmdeck tool" parser artifacts even on perfectly-good
# prompts. Discovered while diagnosing two PR-B captures (#95 follow-up).
SID="capture-$(date +%s%N)-$$"

docker compose -f "$OPENCLAW_COMPOSE" run --rm -T openclaw-cli agent \
  --agent main --session-id "$SID" --json --message "$PROMPT" > "$TMPJSON" 2>&1

# Strip any non-JSON preamble (compose may write progress lines like
# "Container … Running" before the agent's JSON response).
SESSION_FILE=$(sed -n '/^{/,$p' "$TMPJSON" | python3 -c "
import json, sys
d = json.load(sys.stdin)
print(d.get('result', {}).get('meta', {}).get('agentMeta', {}).get('sessionFile', ''))
")
if [[ -z "$SESSION_FILE" ]]; then
  echo "ERROR: no sessionFile in response (session_id=$SID)" >&2
  head -20 "$TMPJSON" >&2
  exit 1
fi

docker exec "$OPENCLAW_GATEWAY" cat "$SESSION_FILE" > "$TMPJSON.session"
python3 "$(dirname "$0")/extract-oc-transcript.py" "$TMPJSON.session"
