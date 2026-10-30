# API

Base URL: `http://127.0.0.1:8787`. Errors use `{"error":{"code":"...","message":"..."}}`. Entry paths are provider-relative IDs (`storage_id:path` where an absolute file id is required) and host paths are never part of any JSON schema. All mock storage is generated under a fresh temporary directory per daemon start; the mock flag is surfaced as `mock:true`.

## Health and storage

- `GET /api/health` → `{"status":"ok","mode":"mock"}`.
- `GET /api/storage` → `{"items":[{id,name,kind,status,mock,read_only,total,used}]}`. `name` reflects a persisted friendly name when set.
- `GET /api/storage/{id}` → storage detail (`id,name,kind,status,mock,read_only,total,used,friendly_name`).
- `GET /api/storage/{id}/usage` → `{id,mock,total,used,free}` computed with `syscall.Statfs` on the provider root (mock roots are real temporary directories).
- `PUT /api/storage/{id}` body `{"friendly_name":"..."}` → persists the friendly name in SQLite (`storage_devices`); the new name is reflected by `GET /api/storage` and detail. Empty name → 400.
- `POST /api/storage/{id}/mount` and `POST /api/storage/{id}/unmount` → `501 not_implemented` in mock mode with no side effects.

Unknown storage ids return 404 `storage_not_found`.

## Browsing, search and pagination

- `GET /api/files?storage_id=&path=&q=&ext=&type=&min_size=&max_size=&sort=&order=&page=&limit=`
  - `q` searches by name (recursive); otherwise `path` is browsed.
  - `ext` filters by extension; `type` filters `file`/`directory`.
  - `min_size`/`max_size` accept bytes or `KB`/`MB`/`GB` suffixes.
  - `sort` ∈ `name,size,modified,created,type`; `order` ∈ `asc,desc` (stable). `created` is the filesystem `ctime`.
  - `limit` defaults to 100 (max 500); `page` is zero-based.
  - Response `{items,page,limit,total}` where items carry `id,storage_id,name,parent,kind,size,modified,created,extension`.
  - Browsing a non-root path also records a recent entry.
- `GET /api/files/metadata/{storage_id}:{path}` → a single Entry object (records a recent).
- `GET /api/files/preview/{storage_id}:{path}` → streams file content with the correct `Content-Type` for images (png/jpeg/gif/webp/bmp), text/csv/log, markdown, json, yaml and pdf. Files larger than 8 MiB return 400 `unsupported_type`; directories and other types return 400.

## Folders and operations

- `POST /api/folders` body `{storage_id,path}` → 201.
- `POST /api/operations` body `{action,storage_id,source,destination?,trash_path?}` where `action` ∈ `copy,move,delete,undo`.
  - `delete` moves the entry to a per-storage Trash (`<root>/.Trash/<id>/<path>`) and returns `trash_path` (relative, never a host path).
  - `undo` restores `source` from `trash_path`.
  - Response `{status,action,storage_id,source,destination?,trash_path?}`.
- `POST /api/operations/undo` behaves like `POST /api/operations` with `action:"undo"`.
- `POST /api/operations/bulk` body `{action,storage_id,sources,destination?,confirm?}` with `action` ∈ `copy,move,delete`.
  - Bulk `delete` requires `confirm:true`; otherwise 400 `confirm_required`.
  - Response `{status,action,storage_id,results:[{status,action,storage_id,source,destination?,trash_path?,error?}]}`.
  - Bulk `undo` is rejected (400).

All operations are sandboxed: absolute paths, `..`, and symlink escapes are rejected.

## Transfer manager

Durable queue in SQLite (`transfer_jobs`, `transfer_items`, `transfer_order`). Job types: `copy`, `move`, `delete`. Statuses: `queued, running, paused, failed, completed, cancelled, conflict`.

- `GET /api/transfers` → `{items:[{id,type,storage_id,status,conflict_strategy,paused,priority,bytes_done,total_bytes,speed,eta,items:[{src,dst,state,error}]}]}` (highest priority first).
- `POST /api/transfers` body `{id?,type,storage_id,conflict_strategy?,priority?,items:[{src,dst?}]}` → 201 job DTO. `conflict_strategy` ∈ `skip,replace,keep_both,ask` (default `skip`).
  - `skip` leaves an existing destination untouched; `replace` overwrites; `keep_both` writes a uniquely named copy; `ask` enters status `conflict` (item state `waiting_conflict`) without writing.
- `POST /api/transfers/{id}/pause` → sets `paused` (workers also honour pause per 256 KiB chunk).
- `POST /api/transfers/{id}/resume` → clears pause and restarts the worker when not already active.
- `POST /api/transfers/{id}/retry` → requeues non-completed items and restarts.
- `POST /api/transfers/{id}/cancel` → cancels the worker; in-flight items are marked `cancelled`.
- `PUT /api/transfers/order` body `["id1","id2",...]` → persists `transfer_order` and assigns descending priorities.

Workers stream in 256 KiB buffers and persist progress (`bytes_done,total_bytes,speed,eta`). On startup `recover()` marks any jobs still `running` as `failed` (items marked `interrupted`).

## Favorites, tags, recents, settings

- `GET /api/favorites?storage_id=` → `{items:[{storage_id,uri}]}` (all when no filter).
- `POST /api/favorites` body `{storage_id,uri}` (idempotent).
- `DELETE /api/favorites` body `{storage_id,uri}`.
- `GET /api/tags?storage_id=&uri=` → `{items:[{storage_id,uri,tag}]}`.
- `POST /api/tags` and `PUT /api/tags` body `{storage_id,uri,tag}` → inserts a tag.
- `DELETE /api/tags` body `{storage_id,uri,tag?}` → deletes one tag, or all tags for the uri when `tag` is omitted.
- `GET /api/recents?kind=` → `{items:[{storage_id,uri,kind,accessed_at}]}` newest first. Recents are recorded on browse, metadata and preview.
- `DELETE /api/recents` → clears all recents.
- `GET /api/settings` → `{mode:"mock", privacy_mode:false, ...persistedKeys}`.
- `PUT /api/settings` body: a JSON object; each key is persisted and echoed back in `GET`. `mode` and `privacy_mode` are computed and not persisted.

## Virtual views

`GET /api/views/{books|music|videos|games|downloads|recent|favorites}` → `{view,items}`.

- `books|music|videos|games` scan all mock storages recursively and filter by extension.
- `downloads` lists the `Downloads` folder of each storage.
- `recent` and `favorites` resolve the recorded recents/favorites to Entry objects.

## Analytics

`GET /api/analytics?storage_id=` → `{storage_id,mock,total,used,free,category_counts,largest_files,largest_directories,cache_timestamp,backgrounded,progress}`.

- Total/used/free come from `statfs`.
- `category_counts` bucket files into `Books/Music/Videos/Games/Documents/Other` by extension.
- `largest_files` (top 20) and `largest_directories` (top 20, provider-relative uri) come from a recursive scan.
- Results are cached (~5 minutes) keyed by `storage_id`; `cache_timestamp` reports when the scan ran.

## Downloads and apps

- `GET /api/downloads` → `503 {"status":"unavailable","source":"downloadsd"}` (no downloadsd client).
- `GET /api/apps/open?storage_id=&path=` → `200 {"status":"unavailable"}` (PJAppRegistry not installed; documented stub).
- `GET /api/apps` → `200 {"status":"unavailable","reason":"pj_app_registry_not_installed"}`.

## Guarantees

- No JSON response, error message, transfer state or header exposes an absolute host path or a storage `Root`. Provider errors are rewritten to `not found`/`permission denied`/`io error` with the provider-relative path.
- SQL is parameterized throughout.
- The sandbox is enforced for every operation and for every metadata/preview/transfer read or write.
