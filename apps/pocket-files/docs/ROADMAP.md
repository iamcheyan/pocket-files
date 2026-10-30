# Roadmap

## Shipped (this repository)

- Mock storage platform: Internal, TF-A, TF-B, USB generated in a temp directory,
  each with its own root, nominal capacity, real usage bytes and seeded
  Books/Music/Videos/ROMs/Documents/Downloads samples, always labelled `mock`.
- Provider abstraction with `InternalStorageProvider`, `TFCardProvider`,
  `USBStorageProvider` over operator-configured roots, plus SMB/WebDAV/Cloud
  `not_implemented` stubs answering `501`.
- Read-only Linux device discovery with fixture tests and a `/api/storage/detected`
  endpoint; kernel names and mount points never leave the process.
- Root sandbox with traversal and symlink-escape rejection on every operation.
- Browse, search (`q`, `ext`, `type`, min/max size), stable sort
  (`name/size/modified/created/type` × asc/desc), pagination.
- Folder create, rename, copy, move, delete-to-Trash, undo, bulk copy/move/delete
  with `confirm` enforcement.
- Persistent transfer queue: pause/resume/retry/cancel, conflict strategies
  (skip/replace/keep_both/ask), priority ordering, streaming 256 KiB copies,
  cross-device copies and startup crash recovery.
- Favorites, tags, recents, settings, friendly device names in SQLite with
  versioned migrations.
- Virtual views (books/music/videos/games/downloads/recent/favorites), analytics
  (statfs capacity, category counts, largest files/directories, cache timestamp).
- React UI: storage overview, device pages, file browser with multi-select,
  transfers, search, favorites, recents, libraries, analytics, settings; four
  themes incl. pure-black OLED; en/zh/ja with enforced key parity; device profiles
  and 1x/1.5x/2x/3x DPI buckets; keyboard, D-pad and Gamepad navigation.
- `make test` (Go race tests, i18n parity, tsc, Vite build, live API smoke) and
  `make build`; one-command `make dev`.
- 20 verified screenshots in `docs/screenshots/` with a documented matrix.

## Next

1. **Split daemons**: extract `storaged` as its own process with a versioned
   internal API; `filesd` keeps search/index/orchestration.
2. **Real mount lifecycle**: implement explicit, separately confirmed
   mount/unmount against a whitelisted device list, with policy prompts and audit
   logging. Until then these endpoints stay `501` on purpose.
3. **udisks2/D-Bus discovery** alongside mountinfo/sysfs, with the same fixture
   approach and identical redaction rules.
4. **Move rollback**: finish the copy+fsync+atomic-rename rollback path for
   cross-filesystem moves, including partial-write cleanup on power loss.
5. **PJOS integration**: `PJAppRegistry`/`AppLaunchAPI` client replacing the
   `unavailable` app stub; `downloadsd` client replacing the 503 stub.
6. **Previews**: thumbnails, EPUB metadata extraction, PDF page previews, ranged
   streaming for large media.
7. **Analytics at scale**: background scan jobs with progress reporting for very
   large trees instead of a single cached pass.
8. **Privacy mode**: persisted opt-in that suppresses recents/favorites writes and
   purges existing history on enable.
9. **Network providers**: SMB and WebDAV behind the existing provider interface,
   with credential handling outside API responses.
10. **Accessibility and i18n depth**: screen-reader pass, RTL groundwork, and
    per-device profile tuning on real handheld/reader hardware.
