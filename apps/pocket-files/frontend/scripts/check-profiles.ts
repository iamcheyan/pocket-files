// Verifies the viewport → device profile mapping against the reference devices
// so a threshold change cannot silently regress the handheld/reader layouts.
import { detectProfile } from '../src/hooks/usePreferences';

const cases: Array<[number, number, string]> = [
  [320, 240, 'handheld'],
  [480, 320, 'handheld'],
  [640, 480, 'reader'],
  [720, 720, 'reader'],
  [640, 340, 'dap'],
  [1280, 720, 'desktop'],
  [1920, 1080, 'desktop'],
];

let failed = false;
for (const [width, height, expected] of cases) {
  const actual = detectProfile(width, height);
  if (actual !== expected) {
    failed = true;
    console.error(`${width}x${height}: expected ${expected}, got ${actual}`);
  }
}
console.log(failed ? 'profile mapping FAILED' : `profile mapping OK (${cases.length} viewports)`);
process.exit(failed ? 1 : 0);
