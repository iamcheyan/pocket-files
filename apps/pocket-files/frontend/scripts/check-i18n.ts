// Build-time i18n key-parity assertion.
// Fails the process when any language dictionary drifts from the canonical `en`
// key set. Run via `npm run check:i18n` (and it also throws at module load in
// the browser, so a drift is caught before the app can render).

import { dictionaries, en, LANGS } from '../src/i18n';

const base = Object.keys(en);
let failed = false;

for (const code of LANGS) {
  const keys = Object.keys(dictionaries[code]);
  const missing = base.filter((key) => !keys.includes(key));
  const extra = keys.filter((key) => !base.includes(key));
  if (missing.length > 0 || extra.length > 0) {
    failed = true;
    console.error(`i18n key parity failure for "${code}"`);
    if (missing.length > 0) console.error(`  missing: ${missing.join(', ')}`);
    if (extra.length > 0) console.error(`  extra:   ${extra.join(', ')}`);
  }
}

if (!failed) {
  console.log(`i18n key parity OK: ${LANGS.length} languages x ${base.length} keys`);
}

process.exit(failed ? 1 : 0);
