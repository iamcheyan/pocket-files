# Pocket Files

PJOS storage API and file manager: React/Vite UI → Go `filesd` REST API →
provider-backed storage → SQLite (`modernc.org/sqlite`). Not Electron; the browser
never reads the filesystem, mounts devices or touches SQLite.

```sh
make dev     # filesd on :8787 + UI on :5180 (Ctrl-C stops both)
make build   # frontend bundle + filesd binary
make test    # go vet + race tests, i18n parity, tsc, Vite build, live API smoke
```

## What runs today

The default runnable configuration uses **four generated mock volumes**
(Internal, TF-A, TF-B, USB) created in a fresh temporary directory. They are
labelled `MOCK` in the UI and report mock capacity with real byte counts of their
own sample files. Real configured volumes, read-only Linux detection, trash/undo,
the persistent transfer queue, favorites/tags/recents and analytics are all
implemented; SMB/WebDAV/Cloud and mount lifecycle are explicit `not_implemented`
stubs. Nothing here formats, partitions, mounts or unmounts a host volume.

See [STORAGE.md](apps/pocket-files/docs/STORAGE.md) for exactly what is real,
what is mock and what is future work.

## Layout

```
apps/pocket-files/frontend/        React 19 + Vite UI (i18n, themes, profiles, keyboard nav)
apps/pocket-files/backend/filesd/  HTTP API, transfer queue, SQLite persistence
apps/pocket-files/backend/storaged/ device discovery, providers, root sandbox
apps/pocket-files/docs/            architecture, storage, database, API, roadmap, dependencies, screenshots
apps/pocket-files/tests/           live API smoke test, dev launcher, screenshot capture
```

## Security posture

- Root sandbox rejects absolute paths, `..` and symlink escapes for every read and
  write; tests cover traversal per provider.
- Real volumes only come from operator configuration (`POCKET_FILES_CONFIG`),
  validated at startup against protected host paths.
- API JSON never contains absolute paths or kernel device names; tests assert it.
- Deletes land in `<root>/.Trash/<journal>/…` and `undo` restores from that exact
  journal; bulk delete requires `confirm:true`.

## Documentation

[ARCHITECTURE](apps/pocket-files/docs/ARCHITECTURE.md) ·
[STORAGE](apps/pocket-files/docs/STORAGE.md) ·
[DATABASE](apps/pocket-files/docs/DATABASE.md) ·
[API](apps/pocket-files/docs/API.md) ·
[ROADMAP](apps/pocket-files/docs/ROADMAP.md) ·
[DEPENDENCIES](apps/pocket-files/docs/DEPENDENCIES.md) ·
[SCREENSHOTS](apps/pocket-files/docs/SCREENSHOTS.md)
