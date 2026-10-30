# Pocket Files

Pocket Files is PJOS's storage API and user file manager. The reference implementation uses a React/Vite UI, Go `filesd`, a provider boundary for storage, and SQLite via modernc.org/sqlite. It is not Electron. The initial runnable configuration uses generated temporary mock storage only; no host filesystem root is exposed to the UI.

## Run

```sh
make dev
```

Open http://127.0.0.1:5173. `filesd` listens on `127.0.0.1:8787`; Vite proxies `/api` to it. `make build` builds UI and daemon; `make test` runs Go checks.

Mock volumes Internal, TF-A, TF-B, and USB are created under a fresh temporary directory each daemon start. They are explicitly labeled mock and contain generated example folders/files. SQLite state defaults to `pocket-files.db`; set `POCKET_FILES_DB` to change it. This initial implementation does not enable real write operations against host volumes.

## Documents

See ARCHITECTURE, STORAGE, DATABASE, API, ROADMAP, DEPENDENCIES, and SCREENSHOTS in this directory.
