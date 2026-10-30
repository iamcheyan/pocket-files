# Verification — pocket-files @ 2026-10-31

- Verdict: PASS
- Commit: `__COMMIT__` (fixes + screenshots; this document is the following commit)
- Remote: `__REMOTE__`
- Runtime: `filesd` on `127.0.0.1:8787` with `POCKET_FILES_MOCK_BIG=1` + fresh temp DB, Vite dev server `127.0.0.1:5180`; `make build` production bundle
- Toolchain: go 1.26.0, bun 1.4.4, node v22.23.3, Chrome 155.0.8059.84

## Test gate
| command | result | notes |
|---|---|---|
| `go vet ./...` (storaged + filesd) | PASS | |
| `go test -race -count=1 ./...` | PASS | 9 storaged tests + 35 filesd tests, 0 failures |
| `bun run check:i18n` | PASS | 3 languages × 204 keys |
| `bun run check:profiles` | PASS | 7 viewports |
| `bun run typecheck` | PASS | `tsc --noEmit` |
| `bun run build` | PASS | production Vite bundle |
| `make e2e` (`tests/smoke.sh` against a fresh daemon) | PASS | 56 API checks, including traversal rejection, trash/undo, transfers, favorites/tags, settings, apps, pins and 501 stubs |

`make test` runs the whole gate; it finished green.

## Screenshot matrix
Captured this run with the repo's own `apps/pocket-files/tests/capture-screenshots.mjs`
(puppeteer-core + system Chrome) against the live UI and API. The script drives real
navigation/theme/search controls, creates a real queued transfer and pauses it by job id,
and aborts on a missing heading, wrong viewport, empty search, duplicate hash, or any
console/HTTP error. `captures.json` holds the machine-readable matrix.

| screen | viewport | theme | file | note |
|---|---|---|---|---|
| overview-desktop-1600x900.png | 1600×900 | light | `apps/pocket-files/docs/screenshots/overview-desktop-1600x900.png` | profile desktop, mock True |
| internal-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/internal-1280x720.png` | profile desktop, mock True |
| tf-a-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/tf-a-1280x720.png` | profile desktop, mock True |
| tf-b-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/tf-b-1280x720.png` | profile desktop, mock True |
| usb-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/usb-1280x720.png` | profile desktop, mock True |
| file-browser-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/file-browser-1280x720.png` | profile desktop, mock True |
| search-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/search-1280x720.png` | profile desktop, mock True |
| favorites-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/favorites-1280x720.png` | profile desktop, mock True |
| recents-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/recents-1280x720.png` | profile desktop, mock True |
| settings-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/settings-1280x720.png` | profile desktop, mock True |
| copy-progress-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/copy-progress-1280x720.png` | profile desktop, mock True |
| copy-progress-paused-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/copy-progress-paused-1280x720.png` | profile desktop, mock True |
| preview-image-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/preview-image-1280x720.png` | profile desktop, mock True |
| overview-dark-1280x720.png | 1280×720 | dark | `apps/pocket-files/docs/screenshots/overview-dark-1280x720.png` | profile desktop, mock True |
| overview-oled-1280x720.png | 1280×720 | oled | `apps/pocket-files/docs/screenshots/overview-oled-1280x720.png` | profile desktop, mock True |
| overview-light-1280x720.png | 1280×720 | light | `apps/pocket-files/docs/screenshots/overview-light-1280x720.png` | profile desktop, mock True |
| overview-handheld-480x320.png | 480×320 | system | `apps/pocket-files/docs/screenshots/overview-handheld-480x320.png` | profile handheld, mock True |
| overview-320x240.png | 320×240 | system | `apps/pocket-files/docs/screenshots/overview-320x240.png` | profile handheld, mock True |
| overview-720x720.png | 720×720 | system | `apps/pocket-files/docs/screenshots/overview-720x720.png` | profile reader, mock True |
| overview-desktop-1920x1080.png | 1920×1080 | system | `apps/pocket-files/docs/screenshots/overview-desktop-1920x1080.png` | profile desktop, mock True |

Count: 20 screenshots, 20 unique hashes (0 duplicates). `tools/check_shots.py`: all PNGs valid,
dimensions match, minimum per-image RGB stddev above the blank threshold, no blank frames.
OLED background is pure #000, dark/light differ; `SCREENSHOTS.md` hash column was refreshed to
the bytes produced by this run.

## Fixed
- The capture script raced the transfer worker: it slept 150 ms, then looked for an active job,
  but a local 300 MB mock copy finishes sooner on this machine, so the run died with
  `unexpected transfer state after pause: no-active-job` and no screenshots were produced.
  → pause the job by the id returned from `POST /api/transfers` immediately, retrying the pause
  call until the job is paused/running/completed → `apps/pocket-files/tests/capture-screenshots.mjs`
  → the capture now yields a genuine paused job (`copy-progress-paused-1280x720.png`).
- Every page logged a console 404 for `/favicon.ico`. → added
  `apps/pocket-files/frontend/public/favicon.svg` and `<link rel="icon">` in `index.html`;
  the capture script now also collects console/pageerror/HTTP≥400 events and fails the run,
  so the current green run proves the console is clean.
- `make test` failed on a fresh clone with `tsc: command not found` because the frontend
  dependencies are installed only by `make build`. → verified that `bun install` is required
  first (documented in the README already); no Makefile change needed, but the run order is
  now recorded here: `bun install` → `make test`.

## Blocked / open questions
- SMB/WebDAV/Cloud providers and mount lifecycle remain explicit `not_implemented` stubs, as
  documented in `STORAGE.md`; they are not exercised and not required for this verification.

## Not verified here
- 掌机真机（RG34XX / PJOS-LINUX）未验证；本轮只在本机 headless Chrome 里跑真实进程。
- Real removable media / NAS mounts: only generated mock volumes were used.
