# Database

`filesd` uses **`modernc.org/sqlite`** (pure Go, no CGO) via `database/sql`.
Default path: `pocket-files.db`; override with `POCKET_FILES_DB`. Connection
pragmas: `journal_mode=WAL`, `busy_timeout=5000`, `foreign_keys=ON`, and
`SetMaxOpenConns(1)` so the transfer workers cannot deadlock each other.

## Schema versioning and migrations

`openDB` creates the tables, then `migrate` applies every migration newer than the
highest recorded version inside a transaction and inserts that version into
`schema_version`. Current `schemaVersion = 2`:

| Version | Contents |
|---|---|
| 1 | `idx_favorites_storage`, `idx_recents_accessed`, `idx_tags_storage_uri` |
| 2 | `idx_transfer_jobs_updated`, `idx_transfer_items_job` |

Migrations are idempotent (`CREATE INDEX IF NOT EXISTS`) so reopening a database
does not duplicate rows, and a database whose version is newer than the binary
supported is refused rather than silently downgraded. `TestMigrationsAreVersionedAndIdempotent`
and `TestMigrationRefusesNewerDatabase` cover both behaviors.

## Tables

| Table | Purpose |
|---|---|
| `schema_version` | Applied migration versions (one row per version). |
| `storage_devices` | Persisted `friendly_name` per storage ID (never a host path). |
| `directories` | Optional directory index rows (`storage_id`, `parent_id`, `name`). |
| `favorites` | Favorite files/folders/devices keyed by `(storage_id, uri)`, unique. |
| `recent_files` | `(storage_id, uri, kind, accessed_at)` opened/browsed entries. |
| `transfer_jobs` | Durable job: type, storage, status, conflict strategy, `extra` JSON (paused/priority/bytes_done/total_bytes/speed/eta), timestamps. |
| `transfer_items` | One row per job item: `item_index`, `src`, `dst`, `state`, `error`. |
| `transfer_order` | Explicit user ordering of queued jobs. |
| `settings` | `key`/`value` JSON settings persisted from `PUT /api/settings`. |
| `file_tags` | `(storage_id, uri, tag)` user tags. |
| `custom_locations` | User-pinned locations (`name`, `storage_id`, `uri`). |

## Rules

- Every query uses parameterized SQL; no string concatenation of user values.
- Only provider-relative URIs are stored. Absolute host roots, device nodes and
  kernel names are never persisted or indexed.
- Transfer state is written through `transferController.persist` under a mutex;
  on startup `recoverTransfers` marks orphaned `running` jobs `failed` with item
  state `interrupted` so a crash never leaves a phantom completed job.
- Indexing must never include an unauthorized root: search and analytics walk the
  provider API, which enforces the root sandbox.

## Privacy and cleanup

There is no network telemetry and no external database. `free`/`used` figures for
real volumes come from `statfs`; mock volumes report the byte total of their own
generated files against a declared nominal capacity and always carry
`mock:true`.

Privacy mode is a product roadmap item: today `GET /api/settings` reports
`privacy_mode:false` as computed state and does not persist it. To clear local
state manually, stop the daemon and delete the database file (plus `-wal`/`-shm`
siblings); the schema is recreated on next start. `DELETE /api/recents` clears
recent history without touching files, and favorites/tags can be removed through
their endpoints.
