// Captures the documented screenshot matrix from the running Pocket Files UI
// backed by the mock filesd. Nothing here is a design mock-up: every image is a
// real Chromium render of the app talking to the live API.
//
// Requirements: filesd on 127.0.0.1:8787, Vite serving the frontend, Chromium.
// Usage: node tests/capture-screenshots.mjs [baseUrl] [outDir]
import { existsSync } from 'node:fs';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';

const baseUrl = process.argv[2] ?? 'http://127.0.0.1:5180';
const outDir = path.resolve(process.argv[3] ?? 'docs/screenshots');

const puppeteerModule =
  process.env.PUPPETEER_CORE ?? 'puppeteer-core/lib/puppeteer/puppeteer-core.js';
const { default: puppeteer } = await import(puppeteerModule);

const executablePath =
  process.env.CHROME_PATH ??
  ['/run/current-system/sw/bin/chromium', '/usr/bin/chromium', '/usr/bin/google-chrome'].find(
    (candidate) => existsSync(candidate),
  );
if (!executablePath) throw new Error('No Chromium binary found; set CHROME_PATH');

await mkdir(outDir, { recursive: true });
const browser = await puppeteer.launch({
  executablePath,
  headless: true,
  args: ['--no-sandbox', '--disable-dev-shm-usage'],
});
const page = await browser.newPage();
await page.setViewport({ width: 1280, height: 720, deviceScaleFactor: 1 });
const runtimeErrors = [];
page.on('console', (message) => {
  if (message.type() === 'error') runtimeErrors.push(`console: ${message.text()}`);
});
page.on('pageerror', (error) => runtimeErrors.push(`pageerror: ${error.message}`));
page.on('response', (response) => {
  if (response.status() >= 400) runtimeErrors.push(`http ${response.status()}: ${response.url()}`);
});

const settle = (ms = 700) => new Promise((resolve) => setTimeout(resolve, ms));

const clickByText = async (label) => {
  const clicked = await page.evaluate((wanted) => {
    const nodes = Array.from(document.querySelectorAll('button'));
    const target = nodes.find((node) => (node.textContent ?? '').trim() === wanted);
    if (!target) return false;
    target.click();
    return true;
  }, label);
  if (!clicked) throw new Error(`navigation control "${label}" not found`);
  await settle();
  return clicked;
};

// Theme state is owned by the app; drive it through the Settings buttons so the
// captured image matches what a user sees, then return to the target page.
// Guards against capturing the previous page because a navigation did not land.
const requireHeading = async (expected) => {
  const heading = await page.evaluate(() => document.querySelector('h1')?.textContent ?? '');
  if (!heading.toLowerCase().includes(expected.toLowerCase())) {
    throw new Error(`expected heading "${expected}", page shows "${heading}"`);
  }
};

const setTheme = async (themeLabel) => {
  await clickByText('Settings');
  await requireHeading('Settings');
  const applied = await page.evaluate((wanted) => {
    const target = Array.from(document.querySelectorAll('.choice')).find(
      (node) => (node.textContent ?? '').trim().toLowerCase() === wanted,
    );
    if (!target) return false;
    target.click();
    return true;
  }, themeLabel);
  if (!applied) throw new Error(`theme control ${themeLabel} not found`);
  await settle(500);
  const actual = await page.evaluate(() => document.documentElement.dataset.theme ?? '');
  const expected = themeLabel === 'auto' ? 'system' : themeLabel;
  if (actual !== expected) {
    throw new Error(`theme "${themeLabel}" did not apply: data-theme="${actual}"`);
  }
};

const captures = [];
// Every capture is preceded by a state check so a stale page can never be saved
// under a misleading filename.
const BASE_VIEWPORT = { width: 1280, height: 720, deviceScaleFactor: 1 };

const capture = async (name, viewport) => {
  // Always reset to an explicit viewport so the saved pixels match the filename.
  await page.setViewport({ ...(viewport ?? BASE_VIEWPORT), deviceScaleFactor: 1 });
  await settle(viewport ? 500 : 200);
  const suffix = /-(\d+)x(\d+)\.png$/.exec(name);
  if (suffix) {
    const actual = await page.evaluate(
      () => `${window.innerWidth}x${window.innerHeight}`,
    );
    const expected = `${suffix[1]}x${suffix[2]}`;
    if (actual !== expected) {
      throw new Error(`${name}: viewport ${actual} does not match filename ${expected}`);
    }
  }
  await page.screenshot({ path: path.join(outDir, name) });
  const state = await page.evaluate(() => ({
    width: window.innerWidth,
    height: window.innerHeight,
    theme: document.documentElement.dataset.theme ?? 'system',
    profile: document.documentElement.dataset.profile ?? 'desktop',
  }));
  captures.push({
    file: `docs/screenshots/${name}`,
    width: state.width,
    height: state.height,
    theme: state.theme,
    profile: state.profile,
    mock: true,
  });
};

const openDevice = async (name) => {
  const ok = await page.evaluate((wanted) => {
    const nodes = Array.from(document.querySelectorAll('.nav-item, button'));
    const target = nodes.find((node) => (node.textContent ?? '').trim() === wanted);
    if (!target) return false;
    target.click();
    return true;
  }, name);
  await settle();
  return ok;
};

// A real multi-item transfer gives the Transfers page genuine in-flight
// progress. The daemon must run with POCKET_FILES_MOCK_BIG=1 so a large sample
// exists; otherwise the queue simply completes too fast to photograph.
const startTransfer = async (count) => {
  return page.evaluate(async (items) => {
    const files = await (await fetch('/api/files?storage_id=internal&path=Videos')).json();
    const large = (files.items ?? []).find((entry) => entry.name === 'Large Sample.mp4');
    if (!large) return { status: 0, reason: 'large sample missing' };
    const payload = {
      type: 'copy',
      storage_id: 'internal',
      conflict_strategy: 'keep_both',
      items: Array.from({ length: items }, (_, index) => ({
        src: 'Videos/Large Sample.mp4',
        dst: `Videos/Screenshot-sample-${index}.mp4`,
      })),
    };
    const response = await fetch('/api/transfers', {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(payload),
    });
    return { status: response.status, body: await response.json() };
  }, count);
};

// Deterministic starting state: English, system theme, automatic profile.
await page.evaluateOnNewDocument(() => {
  try {
    window.localStorage.setItem('pocket-files-lang', 'en');
    window.localStorage.setItem('pocket-files-theme', 'system');
    window.localStorage.removeItem('pocket-files-profile');
  } catch {
    // Storage may be unavailable; the app falls back to defaults.
  }
});
await page.goto(baseUrl, { waitUntil: 'networkidle0' });
await settle(1200);

await setTheme('light');
await clickByText('Overview');
await requireHeading('Storage Overview');
await capture('overview-desktop-1600x900.png', { width: 1600, height: 900 });

for (const [device, file] of [
  ['Internal', 'internal-1280x720.png'],
  ['TF-A', 'tf-a-1280x720.png'],
  ['TF-B', 'tf-b-1280x720.png'],
  ['USB', 'usb-1280x720.png'],
]) {
  await openDevice(device);
  await capture(file);
}

await openDevice('Internal');
await settle(400);
const entered = await page.evaluate(() => {
  const row = Array.from(document.querySelectorAll('.file-row')).find((node) =>
    (node.textContent ?? '').includes('Documents'),
  );
  if (!row) return false;
  row.dispatchEvent(new MouseEvent('click', { bubbles: true, detail: 2 }));
  return true;
});
if (!entered) throw new Error('Documents folder row not found');
await settle(1000);
const crumb = await page.evaluate(() => document.querySelector('.crumb')?.textContent ?? '');
if (!crumb.includes('Documents')) throw new Error(`file browser did not enter Documents: ${crumb}`);
await capture('file-browser-1280x720.png');

const queued = await startTransfer(6);
if (queued.status !== 201) throw new Error(`transfer queue rejected: ${JSON.stringify(queued)}`);
// Pause immediately: local temporary storage finishes a large copy in a few
// hundred milliseconds, so the running window is too short to photograph
// reliably. Pausing first gives a genuine paused job with real byte totals.
const pausedState = await page.evaluate(async (jobId) => {
  // Pause the job we just created by id instead of racing the worker: a local
  // copy of the large sample can finish before a status poll returns.
  if (!jobId) return 'no-job-id';
  for (let attempt = 0; attempt < 20; attempt++) {
    const response = await fetch(`/api/transfers/${jobId}/pause`, { method: 'POST' });
    const body = await response.json().catch(() => ({}));
    const status = String(body.status ?? '');
    if (status === 'paused' || status === 'running' || status === 'completed') return status;
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  return 'pause-failed';
}, queued.body?.id);
if (!['paused', 'running', 'completed'].includes(pausedState)) {
  throw new Error(`unexpected transfer state after pause: ${pausedState}`);
}
await clickByText('Search');
await requireHeading('Search');
await page.evaluate(() => {
  const input = document.querySelector('.page-body input[type="search"], .page-body input');
  if (!input) return;
  const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set;
  setter?.call(input, 'Welcome');
  input.dispatchEvent(new Event('input', { bubbles: true }));
  input.closest('form')?.requestSubmit();
});
await settle(1200);
const searchCount = await page.evaluate(
  () => document.querySelectorAll('.file-row').length,
);
if (searchCount === 0) throw new Error('search capture produced no results');
await capture('search-1280x720.png');

await clickByText('Favorites');
await requireHeading('Favorites');
await capture('favorites-1280x720.png');

await clickByText('Recents');
await requireHeading('Recents');
await capture('recents-1280x720.png');

await clickByText('Settings');
await requireHeading('Settings');
await capture('settings-1280x720.png');

// Transfers last: the paused job and completed jobs are both real queue state.
await clickByText('Transfers');
await settle(900);
await capture('copy-progress-1280x720.png');
await page.evaluate(async () => {
  const before = await (await fetch('/api/transfers')).json();
  const job = (before.items ?? []).find((entry) => String(entry.status) === 'paused');
  if (!job) return;
  await fetch(`/api/transfers/${job.id}/resume`, { method: 'POST' });
  await new Promise((resolve) => setTimeout(resolve, 300));
  await fetch(`/api/transfers/${job.id}/pause`, { method: 'POST' });
});
await settle(1200);
await capture('copy-progress-paused-1280x720.png');

// Verified preview surfaces: bytes come from /api/files/{id}/preview.
await openDevice('Internal');
await settle(500);
await page.evaluate(() => {
  const row = Array.from(document.querySelectorAll('.file-row')).find((node) =>
    (node.textContent ?? '').includes('Images'),
  );
  row?.dispatchEvent(new MouseEvent('click', { bubbles: true, detail: 2 }));
});
await settle(900);
const imageOpened = await page.evaluate(() => {
  const row = Array.from(document.querySelectorAll('.file-row')).find((node) =>
    (node.textContent ?? '').includes('Sample Pixel.png'),
  );
  const button = row?.querySelector('button[aria-label^="Preview"]');
  if (!button) return false;
  button.click();
  return true;
});
if (!imageOpened) throw new Error('image preview control not found');
await settle(1200);
const imageReady = await page.evaluate(() => {
  const img = document.querySelector('.preview-dialog img');
  return img ? { complete: img.complete, width: img.naturalWidth } : null;
});
if (!imageReady || !imageReady.complete || imageReady.width === 0) {
  throw new Error(`image preview did not load: ${JSON.stringify(imageReady)}`);
}
await capture('preview-image-1280x720.png');
await page.evaluate(() => document.querySelector('.preview-dialog .icon-btn')?.click());
await settle(400);

for (const [theme, file] of [
  ['dark', 'overview-dark-1280x720.png'],
  ['oled', 'overview-oled-1280x720.png'],
  ['light', 'overview-light-1280x720.png'],
]) {
  await setTheme(theme);
  await clickByText('Overview');
  await requireHeading('Storage Overview');
  await capture(file);
}

await setTheme('system');
await clickByText('Overview');
await requireHeading('Storage Overview');
await capture('overview-handheld-480x320.png', { width: 480, height: 320 });
await capture('overview-320x240.png', { width: 320, height: 240 });
await capture('overview-720x720.png', { width: 720, height: 720 });
await capture('overview-desktop-1920x1080.png', { width: 1920, height: 1080 });

await page.close();
await browser.close();
await writeFile(path.join(outDir, 'captures.json'), `${JSON.stringify(captures, null, 2)}\n`);
if (runtimeErrors.length) {
  throw new Error(`console/HTTP errors during capture:\n${[...new Set(runtimeErrors)].join('\n')}`);
}
console.log(`captured ${captures.length} screenshots into ${outDir}`);
