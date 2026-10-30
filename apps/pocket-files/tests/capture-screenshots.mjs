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

const settle = (ms = 700) => new Promise((resolve) => setTimeout(resolve, ms));

const clickByText = async (label) => {
  const clicked = await page.evaluate((wanted) => {
    const nodes = Array.from(document.querySelectorAll('button'));
    const target = nodes.find((node) => (node.textContent ?? '').trim() === wanted);
    if (!target) return false;
    target.click();
    return true;
  }, label);
  await settle();
  return clicked;
};

// Theme state is owned by the app; drive it through the Settings buttons so the
// captured image matches what a user sees, then return to the target page.
const setTheme = async (themeLabel) => {
  await clickByText('Settings');
  const applied = await page.evaluate((wanted) => {
    const buttons = Array.from(document.querySelectorAll('.choice, button'));
    const target = buttons.find((node) => (node.textContent ?? '').trim().toLowerCase() === wanted);
    if (!target) return false;
    target.click();
    return true;
  }, themeLabel);
  if (!applied) throw new Error(`theme control ${themeLabel} not found`);
  await settle(400);
};

const captures = [];
const capture = async (name, viewport) => {
  if (viewport) {
    await page.setViewport({ ...viewport, deviceScaleFactor: 1 });
    await settle(500);
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

await setTheme('system');
await clickByText('Overview');
await capture('overview-desktop-1280x720.png');

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
const pausedState = await page.evaluate(async () => {
  await new Promise((resolve) => setTimeout(resolve, 150));
  const body = await (await fetch('/api/transfers')).json();
  const job = (body.items ?? []).find((entry) =>
    ['queued', 'running', 'paused'].includes(String(entry.status)),
  );
  if (!job) return 'no-active-job';
  await fetch(`/api/transfers/${job.id}/pause`, { method: 'POST' });
  await new Promise((resolve) => setTimeout(resolve, 250));
  const after = await (await fetch('/api/transfers')).json();
  const updated = (after.items ?? []).find((entry) => entry.id === job.id);
  return updated ? String(updated.status) : 'job-vanished';
});
if (!['paused', 'running', 'completed'].includes(pausedState)) {
  throw new Error(`unexpected transfer state after pause: ${pausedState}`);
}
await clickByText('Search');
await page.type('.search input', 'Notes');
await settle(900);
await capture('search-1280x720.png');

await clickByText('Favorites');
await capture('favorites-1280x720.png');

await clickByText('Recents');
await capture('recents-1280x720.png');

await clickByText('Settings');
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

await setTheme('dark');
await clickByText('Overview');
await capture('overview-dark-1280x720.png');

await setTheme('light');
await capture('overview-light-1280x720.png');

await setTheme('oled');
await capture('overview-oled-1280x720.png');

await setTheme('system');
await capture('overview-handheld-480x320.png', { width: 480, height: 320 });
await capture('overview-320x240.png', { width: 320, height: 240 });
await capture('overview-720x720.png', { width: 720, height: 720 });
await capture('overview-1280x720.png', { width: 1280, height: 720 });
await capture('overview-desktop-1920x1080.png', { width: 1920, height: 1080 });

await page.close();
await browser.close();
await writeFile(path.join(outDir, 'captures.json'), `${JSON.stringify(captures, null, 2)}\n`);
console.log(`captured ${captures.length} screenshots into ${outDir}`);
