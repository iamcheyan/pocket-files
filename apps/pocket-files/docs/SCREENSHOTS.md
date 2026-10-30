# Screenshots

All images are real Chromium renders of the running Pocket Files UI (Vite dev
server) talking to a live `filesd` on `127.0.0.1:8787`. Every device shown is a
generated **mock** volume; the `MOCK` badges and the in-app "Preview mode" note are
part of the captured UI. No design mock-ups, no fabricated capacity, and no real
host or NAS content.

## How they are produced

```sh
# 1. Backend with mock volumes, large sample enables visible transfer progress
cd apps/pocket-files/backend/filesd
POCKET_FILES_MOCK_BIG=1 POCKET_FILES_DB=/tmp/pocket-files-shots.db \
  POCKET_FILES_ADDR=127.0.0.1:8787 go run . &

# 2. Frontend (port 5180 avoids the 5173 default used by sibling PJOS apps)
cd apps/pocket-files/frontend
bun install && bunx vite --host 127.0.0.1 --port 5180 --strictPort &

# 3. Capture
NODE_PATH=/home/tetsuya/.bun/install/global/node_modules \
PUPPETEER_CORE=/home/tetsuya/.bun/install/global/node_modules/puppeteer-core/lib/puppeteer/puppeteer-core.js \
CHROME_PATH=/run/current-system/sw/bin/chromium \
node apps/pocket-files/tests/capture-screenshots.mjs \
  http://127.0.0.1:5180 apps/pocket-files/docs/screenshots
```

The script pins language to `en` and theme to `system` before navigation, drives
real UI controls, creates a real queued transfer, and **fails loudly** when a
navigation did not land, a theme did not apply, a viewport does not match the
target filename, or a search produced no rows. A screenshot cannot be saved under
a misleading name.

## Captures

| File | Resolution | Theme / profile | State | SHA-256 (12) |
|---|---|---|---|---|
| `overview-desktop-1600x900.png` | 1600×900 | light / desktop | Storage Overview | `1ab0bcb9cc1f` |
| `internal-1280x720.png` | 1280×720 | light / desktop | Internal root | `df5666cfadeb` |
| `tf-a-1280x720.png` | 1280×720 | light / desktop | TF-A root | `027c0e692b57` |
| `tf-b-1280x720.png` | 1280×720 | light / desktop | TF-B root | `f4bdcbf45096` |
| `usb-1280x720.png` | 1280×720 | light / desktop | USB root | `eda6a84dd3d0` |
| `file-browser-1280x720.png` | 1280×720 | light / desktop | Internal/Documents | `52aa31e5feb2` |
| `search-1280x720.png` | 1280×720 | light / desktop | Search "Welcome" results | `387c9263df6a` |
| `favorites-1280x720.png` | 1280×720 | light / desktop | Favorites | `e8071338b0e2` |
| `recents-1280x720.png` | 1280×720 | light / desktop | Recents | `b290c3aa0efa` |
| `settings-1280x720.png` | 1280×720 | light / desktop | Settings (theme, language, profile, DPI) | `5e29a7f53d84` |
| `copy-progress-1280x720.png` | 1280×720 | light / desktop | Transfers with live queue | `467da1f303e2` |
| `copy-progress-paused-1280x720.png` | 1280×720 | light / desktop | Transfers, paused job | `a4dc63b912c4` |
| `overview-dark-1280x720.png` | 1280×720 | dark / desktop | Storage Overview | `96b1fae7c1e7` |
| `overview-oled-1280x720.png` | 1280×720 | OLED (#000) / desktop | Storage Overview | `ddee4c62512d` |
| `overview-light-1280x720.png` | 1280×720 | light / desktop | Storage Overview | `bf5f9e094d88` |
| `overview-handheld-480x320.png` | 480×320 | system / handheld | Storage Overview | `8abf27643e18` |
| `overview-320x240.png` | 320×240 | system / handheld | Storage Overview (smallest) | `44f589034f16` |
| `overview-720x720.png` | 720×720 | system / reader | Storage Overview | `627e276188e1` |
| `overview-desktop-1920x1080.png` | 1920×1080 | system / desktop | Storage Overview | `80541145c3f8` |

`captures.json` holds the same matrix in machine-readable form (width, height,
theme, resolved profile, mock flag).

Required sizes are covered: 320×240, 480×320, 720×720, 1280×720, 1600×900 and
1920×1080. Required themes are covered separately (light, dark, OLED) and the
transfers page is captured twice, running and paused.

## Verification performed

- PNG signature `89 50 4E 47 0D 0A 1A 0A` in all 19 files.
- IHDR width/height decoded from bytes 16–24 equal the filename suffix (0 mismatches).
- 19/19 distinct SHA-256 digests, so each page, size and theme is a genuinely
  different render.
- OLED body background measured `#000000`; dark `#101114` (`rgb(16,17,20)`);
  light `#f7f8fa`.
