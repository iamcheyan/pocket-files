# Screenshots

All images below are real Chromium renders of the running Pocket Files UI
(`Vite` dev server) talking to a live `filesd` on `127.0.0.1:8787`. Every device
shown is a generated **mock** volume; `MOCK` badges and the in-app note are part
of the captured UI. No design mock-ups, no fabricated capacity, no real host data.

## How they were taken

```sh
# 1. Backend with mock volumes (large sample enables visible transfer progress)
cd apps/pocket-files/backend/filesd
POCKET_FILES_MOCK_BIG=1 POCKET_FILES_DB=/tmp/pocket-files-shots.db go run . &

# 2. Frontend
node <vite>/bin/vite.js "$PWD/apps/pocket-files/frontend" --host 127.0.0.1 --port 5180 &

# 3. Capture (Chromium from the host; puppeteer-core resolves from NODE_PATH)
NODE_PATH=/home/tetsuya/.bun/install/global/node_modules \
PUPPETEER_CORE=/home/tetsuya/.bun/install/global/node_modules/puppeteer-core/lib/puppeteer/puppeteer-core.js \
CHROME_PATH=/run/current-system/sw/bin/chromium \
node apps/pocket-files/tests/capture-screenshots.mjs http://127.0.0.1:5180 apps/pocket-files/docs/screenshots
```

The script sets `localStorage` language to `en` and theme to `system` before
navigating, drives real UI controls (navigation, folder open, theme/language
buttons), creates a real queued transfer for the transfer screens, and fails
loudly if a state is not what the capture claims.

## Captures

| File | Resolution | Mode / profile | SHA-256 (first 12) |
|---|---|---|---|
| `overview-desktop-1280x720.png` | 1280×720 | system theme, desktop | `7b9a97437d52` |
| `overview-1280x720.png` | 1280×720 | system theme, desktop | `e75ec884963f` |
| `internal-1280x720.png` | 1280×720 | system, desktop, Internal root | `64d4f11829f6` |
| `tf-a-1280x720.png` | 1280×720 | system, desktop, TF-A root | `5298ac5c939e` |
| `tf-b-1280x720.png` | 1280×720 | system, desktop, TF-B root | `813831be71fa` |
| `usb-1280x720.png` | 1280×720 | system, desktop, USB root | `fc15ed9c11f9` |
| `file-browser-1280x720.png` | 1280×720 | system, desktop, Internal/Documents | `bccda7137cb7` |
| `copy-progress-1280x720.png` | 1280×720 | system, Transfers with live job | `fdef0d167098` |
| `copy-progress-paused-1280x720.png` | 1280×720 | system, Transfers, paused job | `da73f518eeae` |
| `search-1280x720.png` | 1280×720 | system, Search "Notes" | `ed116d6604df` |
| `favorites-1280x720.png` | 1280×720 | system, Favorites | `7a4b8526277c` |
| `recents-1280x720.png` | 1280×720 | system, Recents | `81937c2e0c17` |
| `settings-1280x720.png` | 1280×720 | system, Settings | `d6a48dd890ef` |
| `overview-dark-1280x720.png` | 1280×720 | dark theme | `3541c5aa2975` |
| `overview-light-1280x720.png` | 1280×720 | light theme | `df409f08d9b1` |
| `overview-oled-1280x720.png` | 1280×720 | OLED theme (#000) | `a49122801e2c` |
| `overview-handheld-480x320.png` | 480×320 | handheld profile | `9820bc18bcc7` |
| `overview-320x240.png` | 320×240 | handheld profile (smallest) | `a4f7caee9a3d` |
| `overview-720x720.png` | 720×720 | reader profile | `81d3d90eaa2f` |
| `overview-desktop-1920x1080.png` | 1920×1080 | desktop | `6aff3378369f` |

`captures.json` records the same matrix in machine-readable form (width, height,
theme, resolved device profile, mock flag).

## Verification performed

- PNG signature `89 50 4E 47 0D 0A 1A 0A` present in all 20 files.
- IHDR dimensions decoded from bytes 16–24 match every filename suffix.
- All 20 SHA-256 digests are distinct (20/20), so dark/light/OLED and each page
  are genuinely different renders rather than duplicates.
- OLED capture shows a pure `#000000` body background; light shows `#f7f8fa`.
