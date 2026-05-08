#!/usr/bin/env python3
"""
Replace the OpenClaw chat-UI TODO placeholder in each per-pack reference
page with the captured transcript at <transcripts-dir>/<family>_<pack>.md.

Each transcript filename is `<family>_<pack-with-dashes>.md` — e.g.
`github_list-issues.md` injects into `docs/reference/packs/github/list-issues.md`.

Usage:
  inject-transcripts.py <transcripts-dir> [<docs-root>]

  transcripts-dir   directory of <family>_<pack>.md files produced by
                    capture-oc.sh (e.g. /tmp/captures/oc-transcripts)
  docs-root         override for docs/reference/packs root
                    (default: <repo>/docs/reference/packs)
"""
import re
import sys
from pathlib import Path

if len(sys.argv) < 2:
    print(__doc__, file=sys.stderr)
    sys.exit(2)

TRANSCRIPTS = Path(sys.argv[1])
DOCS_ROOT = Path(sys.argv[2]) if len(sys.argv) > 2 else (
    Path(__file__).resolve().parent.parent.parent / "docs" / "reference" / "packs"
)

# Match the TODO HTML comment + the "OpenClaw chat capture pending"
# line under it. Tolerant of whitespace and TODO-comment variants.
PLACEHOLDER = re.compile(
    r"<!--\s*TODO\(maintainer\):.*?-->\s*\n\s*"
    r"> \*OpenClaw chat capture pending\.\*",
    re.DOTALL,
)

count = 0
total = len(list(TRANSCRIPTS.glob("*.md")))
for transcript_file in sorted(TRANSCRIPTS.glob("*.md")):
    family_pack = transcript_file.stem
    if "_" not in family_pack:
        print(f"  ⚠ skipping {transcript_file.name}: name must be <family>_<pack>.md")
        continue
    family, pack = family_pack.split("_", 1)
    page = DOCS_ROOT / family / f"{pack}.md"
    if not page.exists():
        print(f"  ⚠ no page for {family}/{pack} ({page})")
        continue

    page_text = page.read_text()
    transcript = transcript_file.read_text().rstrip()

    # Lambda replacement so JSON \u sequences in the transcript aren't
    # interpreted as regex escape sequences by Python's `re.subn`.
    new_text, n = PLACEHOLDER.subn(lambda _m: transcript, page_text, count=1)
    if n == 0:
        print(f"  ⚠ {page} — no placeholder matched")
        continue
    page.write_text(new_text)
    print(f"  ✓ {family}/{pack}: {len(transcript)} bytes → {page.name}")
    count += 1

print(f"\n{count}/{total} pages updated")
