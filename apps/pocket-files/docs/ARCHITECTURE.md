# Architecture

```text
React + Vite UI (apps/pocket-files/frontend)
        │  REST/JSON over /api (no filesystem, SQLite or device access)
        ▼
filesd (apps/pocket-files/backend/filesd — Go, package main)
        │  routing · backend services · transfer queue · SQLite persistence
        ▼
providerRouter  ──► MockProvider        (Internal, TF-A, TF-B, USB)
                ├─► InternalStorageProvider / TFCardProvider / USBStorageProvider
                │       (LocalProvider over an operator-configured root)
                └─► UnimplementedProvider (SMB, WebDAV, Cloud → 501)
        ▼
filesystem (sandboxed, root-confined) + modernc.org/sqlite
```

## Process boundaries

`filesd` is a single HTTP daemon today. The provider layer is the storage
boundary that `storaged` will occupy: every storage operation goes through the
`Provider` interface (`Discover/Mount/Unmount/Browse/Search/Stat/Copy/Move/
Delete/CreateDir/AvailableSpace`, plus `Trash`/`Restore` for recoverable deletes).
Handlers, the transfer controller and analytics all talk to `storageBackend`, not
to a concrete provider, so splitting `storaged` out is a packaging change rather
than a redesign.

The UI never receives an absolute path, a device node or a database row. It gets
storage IDs, friendly names, provider-relative entry IDs (`storage_id:path`),
metadata and capabilities. All of that is produced by `filesd`.

## Safety model

- **Root sandbox**: `resolveLocalPath` / `sandbox.resolve` reject absolute paths,
  `..` traversal and symlinks that resolve outside the configured root. Every
  read, write, rename, copy, move, trash and preview path goes through it.
- **Configured roots only**: real local volumes come from `POCKET_FILES_CONFIG`
  (`{"storages":[{id,friendly_name,kind,root,read_only}]}`). Startup validation
  refuses `/`, `/nix`, `/boot`, `/home`, `/storage`, `/etc`, `/usr`, `/var`,
  relative roots, duplicate IDs and non-directories. API callers can never supply
  or change a root.
- **No unrequested host mutation**: `mount`/`unmount` return `501`, SMB/WebDAV/
  Cloud return `501`, and Linux discovery is strictly read-only. Nothing in this
  repository formats, partitions or unmounts a host volume.
- **Recoverable deletes**: `delete` moves the entry into `<root>/.Trash/<journal>/…`
  and returns the relative journal path; `undo` restores from exactly that path.
  Bulk delete requires `confirm:true`.
- **Streaming transfers**: copies use a 256 KiB buffer, persist
  `bytes_done/total_bytes/speed/eta`, and cross-device copies resolve the source
  and destination through different providers. Move on the same device uses
  rename semantics via the provider; cross-device move copies then deletes.
- **Crash recovery**: on startup any job still marked `running` is marked `failed`
  with item state `interrupted`, so a restart never reports a phantom success.
- **No path leaks**: JSON responses, errors and headers carry provider-relative
  identifiers only; tests assert this across every endpoint.

## Storage identity

Devices are identified by stable IDs, never by guessed slot names. Mock volumes
use fixed IDs (`internal`, `tf-a`, `tf-b`, `usb`) that are explicitly labelled
`mock:true`. Detected Linux media are keyed by filesystem UUID from
`/dev/disk/by-uuid`; a partition without a UUID is skipped rather than being
labelled "TF-A". Friendly names are user-editable and persisted in SQLite.

## Frontend

A single React app with an explicit `NavTarget` union instead of a router
dependency. Preferences (theme `system|light|dark|oled`, language `en|zh|ja`,
device profile `auto|desktop|handheld|reader|dap`) drive `data-*` attributes on
`<html>`; CSS and the DPI buckets (`1x/1.5x/2x/3x`) respond to those attributes.
`useKeyboardNav` provides spatial arrow-key focus, Enter/Space activation,
Escape-as-back and Gamepad API mapping, so every flow is reachable without a
mouse. i18n has 185 keys per language with a load-time and CI-enforced parity
check.

## Testing shape

`make test` runs: `go vet`, Go tests with `-race` (provider contract across mock
and local families, sandbox escapes, config validation, operations/undo, bulk
confirm, transfer pause/resume/cancel/conflict/recovery, cross-device transfer,
search sort/filter, analytics, view, persistence, path-leak assertions,
discovery fixtures), i18n parity, `tsc --noEmit`, the production Vite build, and
a live API smoke test (`apps/pocket-files/tests/smoke.sh`) against a freshly
started daemon on port 8788.
