# Pocket Files

PJOS storage API and file manager. React/Vite UI → Go `filesd` REST API → provider-backed storage → SQLite (`modernc.org/sqlite`). No Electron; the browser never reads the filesystem directly.

```sh
make dev     # filesd :8787 + Vite :5173
make build   # frontend production build + Go binaries
make test    # Go unit/integration tests, API smoke, frontend type-check/build
```

The runnable configuration uses four generated mock volumes (Internal, TF-A, TF-B, USB) created in an OS temporary directory. The UI labels them as mock; no host volume, `/storage`, or production mount is read or mutated. Linux device discovery and real mount/write support are described honestly in [docs/STORAGE.md](apps/pocket-files/docs/STORAGE.md).

Layout:

```
apps/pocket-files/frontend/        React + Vite UI
apps/pocket-files/backend/filesd/  API, storage providers, queue, SQLite
apps/pocket-files/docs/            architecture, storage, database, API, roadmap, dependencies, screenshots
apps/pocket-files/tests/           API integration smoke tests
```

Documentation: [ARCHITECTURE](apps/pocket-files/docs/ARCHITECTURE.md) · [STORAGE](apps/pocket-files/docs/STORAGE.md) · [DATABASE](apps/pocket-files/docs/DATABASE.md) · [API](apps/pocket-files/docs/API.md) · [ROADMAP](apps/pocket-files/docs/ROADMAP.md) · [DEPENDENCIES](apps/pocket-files/docs/DEPENDENCIES.md) · [SCREENSHOTS](apps/pocket-files/docs/SCREENSHOTS.md)
