# Architecture

```text
React/Vite UI → REST API → filesd → Provider interface → storaged providers → volumes
                                     └→ SQLite (modernc)
```

The UI receives storage IDs, relative provider URIs, friendly names, metadata, and capabilities. It never receives a host root or device node. `filesd` owns search, persistent user data, virtual views, and operation orchestration; `storaged` owns device discovery and storage operations. The Go provider contract is currently defined in `backend/filesd/storage.go`; separating the daemon into its final package/process is roadmap work.

Mock provider data is generated in a fresh OS temporary directory. Provider sandbox resolution rejects absolute paths, `..`, and symlinks escaping a configured root. Current API write support is limited to mock folder creation. Linux discovery and real mount/write paths are not enabled. This ensures the development host's production `/storage` is never accessed or altered.

Future real Linux discovery reads mountinfo/sysfs/udisks metadata read-only. Mount and unmount must be explicit, separately confirmed requests. Stable provider IDs—not device labels or guessed slot names—identify devices; labels are editable presentation values. App launches use PJAppRegistry/AppLaunchAPI; unavailable apps must be reported as unavailable.
