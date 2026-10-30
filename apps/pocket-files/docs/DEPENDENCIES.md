# Dependencies

Verified versions are those actually used to build and test this repository on
the NixOS HX90 workstation.

## Backend (`apps/pocket-files/backend/filesd`)

| Dependency | Version | Why |
|---|---|---|
| Go | 1.26.6 (module declares `go 1.26.0`) | Daemon implementation |
| `modernc.org/sqlite` | v1.38.2 | Pure-Go SQLite; no CGO, no system libsqlite |
| `modernc.org/libc` | v1.66.3 (indirect) | SQLite port support |
| `modernc.org/memory`, `mathutil` | v1.11.0 / v1.7.1 (indirect) | SQLite port support |
| `golang.org/x/sys` | v0.34.0 (indirect) | Platform syscalls |
| `github.com/google/uuid` | v1.6.0 (indirect) | SQLite dependency |
| `github.com/mattn/go-isatty` | v0.0.20 (indirect) | SQLite dependency |
| `github.com/ncruces/go-strftime` | v0.1.9 (indirect) | SQLite dependency |
| `github.com/remyoudompheng/bigfft` | v0.0.0-20230129092748 (indirect) | SQLite dependency |
| `github.com/dustin/go-humanize` | v1.0.1 (indirect) | SQLite dependency |
| `golang.org/x/exp` | v0.0.0-20250620022241 (indirect) | SQLite dependency |

Standard library only for HTTP, filesystem, `syscall.Statfs`, and JSON. No
udisks2 client, no CGO, no SQLite CLI required.

## Frontend (`apps/pocket-files/frontend`)

| Dependency | Version | Why |
|---|---|---|
| React | 19.3.1 | UI runtime |
| React DOM | 19.3.1 | DOM renderer |
| Vite | 8.3.5 | Dev server and production bundler |
| TypeScript | 7.0.2 | Type checking |
| `@vitejs/plugin-react` | 6.1.2 | JSX transform / fast refresh |
| `lucide-react` | 1.64.0 | Icon set (only icon source) |
| `@types/react` | 19.2.14 (dev) | Types |
| `@types/react-dom` | 19.2.3 (dev) | Types |

`bun.lock` pins the resolved tree. There is **no Electron**, no Node filesystem
access from the UI, and no direct SQLite access from the browser — the frontend
speaks HTTP only.

## Tooling

| Tool | Version used |
|---|---|
| Bun | 1.4.2 (install, scripts, i18n check) |
| Node.js | 24.19.0 (Vite under Bun-installed modules) |
| Chromium | host NixOS build (`/run/current-system/sw/bin/chromium`) for screenshot capture |
| `puppeteer-core` | resolved from the host Bun global install for `tests/capture-screenshots.mjs` |

`puppeteer-core` is intentionally not a project dependency: screenshot capture is
a maintainer step and resolves the host copy via `PUPPETEER_CORE`/`NODE_PATH`.

## Deliberately absent

- `Electron` — the product is a web UI plus a Go daemon.
- `udisks2`/D-Bus client — detection currently reads mountinfo/sysfs only.
- `downloadsd` and `PJAppRegistry` clients — both are reported as `unavailable`
  rather than faked.
- Native image/PDF/EPUB parsers — previews stream bytes with verified
  content types; metadata extraction is on the roadmap.
