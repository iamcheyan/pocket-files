# Storage

All of this lives in the standalone `backend/storaged` package; `filesd` consumes
it through the `storaged.Backend` interface and never touches a filesystem itself.

## Provider contract

```go
type Provider interface {
    Discover(ctx) ([]Device, error)
    Mount(ctx, id) error
    Unmount(ctx, id) error
    Browse(ctx, id, rel) ([]Entry, error)
    Search(ctx, id, query) ([]Entry, error)
    Stat(ctx, id, rel) (Entry, error)
    Copy(ctx, srcID, src, dst) error
    Move(ctx, srcID, src, dst) error
    Delete(ctx, id, rel) error
    CreateDir(ctx, id, rel) error
    AvailableSpace(ctx, id) (total, used int64, err error)
    // recoverable delete used by the transfer controller
    Trash(ctx, id, rel) (journal string, err error)
    Restore(ctx, id, src, journal string) error
}
```

`providerRouter` implements `storageBackend` (the same surface plus `allDevices`
and `resolve`) and dispatches each call to the provider that owns the storage ID.

| Provider | Status | Notes |
|---|---|---|
| `MockProvider` | **real, mock data** | Four generated volumes in an OS temp dir, seeded with small sample files. Declared nominal capacity; `used` is the byte total of its own files. `mock:true`. |
| `InternalStorageProvider` | real, configuration-gated | `LocalProvider` over one configured root. Real `statfs` capacity. |
| `TFCardProvider` | real, configuration-gated | Same code path, `kind:"tf"`. Identity comes from config/stable ID, never from a guessed slot name. |
| `USBStorageProvider` | real, configuration-gated | Same code path, `kind:"usb"`. |
| SMB / WebDAV / Cloud | **`not_implemented` stubs** | Advertised by `GET /api/providers`; operations return `501`. |

Configured volumes (all fields required except `friendly_name`/`read_only`):

```json
{
  "storages": [
    {"id": "onboard", "friendly_name": "Onboard storage", "kind": "internal",
     "root": "/home/me/PocketLibrary", "read_only": false},
    {"id": "card-a", "friendly_name": "Reading card", "kind": "tf",
     "root": "/run/media/me/CARD-A", "read_only": true}
  ]
}
```

Set `POCKET_FILES_CONFIG=/path/to/providers.json`. Startup refuses `/`, `/nix`,
`/boot`, `/home`, `/storage`, `/etc`, `/usr`, `/var`, relative paths, duplicate
IDs, missing directories and unknown kinds.

## Linux device detection

`discoverRemovableStorage` parses `/proc/self/mountinfo` and confirms removability
through `/sys/class/block/<dev>/removable` (falling back to the parent disk for
partitions). It:

- skips virtual/pseudo filesystems (proc, sysfs, tmpfs, overlay, squashfs, nfs,
  cifs, sshfs, zfs, …) and `/`, `/nix…`, `/boot…`, `/home`, anything under
  `/storage`;
- requires a real `/dev/*` source;
- identifies each volume by filesystem UUID from `/dev/disk/by-uuid`, skipping
  partitions without one;
- decodes mountinfo escapes (`\040`) and reports `ro` mounts as read-only;
- reports capacity via `statfs` when the mount point is reachable.

Results are exposed read-only at `GET /api/storage/detected` with
`mode:"read_only"`. Kernel device names and absolute mount points are internal
correlation fields and never serialized. In this environment no removable media
is attached, so that endpoint correctly returns an empty list.

Fixture tests (`discovery_test.go`) cover: mmc partitions, USB disk partitions, a
non-removable disk, the production `/storage` mount, a virtual filesystem, an
unlabelled partition, an escaped mount point, missing sysfs, and missing
mountinfo.

## Deletion, trash and undo

`delete` moves the entry to `<root>/.Trash/<journal-id>/<original-path>` and
returns the journal path. `undo` takes `{storage_id, source, trash_path}` and
restores it. Journals never leave the provider root, and `trash_path` is always
relative. Bulk delete additionally requires `confirm:true`; the UI shows a
confirmation dialog before sending it.

## Transfers

The queue lives in SQLite (`transfer_jobs`, `transfer_items`, `transfer_order`).
States: `queued`, `running`, `paused`, `failed`, `completed`, `cancelled`,
`conflict`. Conflict strategies: `skip`, `replace`, `keep_both`, `ask`.

Copies stream in 256 KiB chunks with pause/cancel checks per chunk and persist
`bytes_done`, `total_bytes`, `speed`, `eta`. A `destination_storage` on the job
(or on a bulk copy) makes the destination resolve through a different provider,
so Internal → TF-A/USB is a real streamed cross-device copy. Same-device moves use
the provider's rename path; cross-device moves copy then delete the source. Failed
items roll back by removing a partially written destination.

On startup, jobs left in `running` are marked `failed` with item state
`interrupted`; nothing is reported as completed that did not finish.

## Boundaries to respect

- Never format, partition, mount or unmount a host volume from this service.
- Never copy real host or NAS content into the repository or into mock fixtures.
- Never expose kernel device names or absolute paths in API responses.
- Never infer `TF-A`/`TF-B` from hardware; use stable IDs plus friendly names.
