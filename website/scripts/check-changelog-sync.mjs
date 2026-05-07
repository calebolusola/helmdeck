// Verifies that website/src/pages/changelog.md (the docs-site page) and the
// canonical /CHANGELOG.md at repo root have identical content under the
// frontmatter. Run by docs-build CI on every PR; fails fast when contributors
// update one but forget the other.

import {readFile} from 'node:fs/promises';
import {existsSync} from 'node:fs';
import {dirname, resolve} from 'node:path';
import {fileURLToPath} from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, '..', '..', 'CHANGELOG.md');
const page = resolve(here, '..', 'src', 'pages', 'changelog.md');

if (!existsSync(root)) {
  console.error(`[check-changelog-sync] missing canonical file: ${root}`);
  process.exit(1);
}
if (!existsSync(page)) {
  console.error(`[check-changelog-sync] missing site page: ${page}`);
  process.exit(1);
}

const rootBody = (await readFile(root, 'utf8')).trimEnd();
const pageRaw = await readFile(page, 'utf8');

// Strip the leading frontmatter block from the page (---\n...\n---\n\n).
const fm = pageRaw.match(/^---\n[\s\S]*?\n---\n+/);
const pageBody = (fm ? pageRaw.slice(fm[0].length) : pageRaw).trimEnd();

if (rootBody !== pageBody) {
  console.error(
    `[check-changelog-sync] DRIFT detected.\n  canonical: ${root}\n  site page: ${page}\n  Run:  cp ${root} ${page} && (re-add frontmatter)\n`,
  );
  process.exit(1);
}
console.log('[check-changelog-sync] in sync ✓');
