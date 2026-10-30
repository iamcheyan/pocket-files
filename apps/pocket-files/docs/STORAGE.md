# Storage providers

`Provider` defines discovery, mount lifecycle, browse, search, stat, copy, move, delete, directory creation, and available-space methods. `MockProvider` implements the interface using temporary directories and generated examples for Internal, TF-A, TF-B, and USB. These names represent mock volumes only.

The mock provider applies a root sandbox on path resolution and refuses traversal and symlink escapes. Its reported capacity is illustrative mock metadata. It does not claim host capacity. The web client sees only friendly IDs/names and relative entry IDs.

`InternalStorageProvider`, `TFCardProvider`, and `USBStorageProvider` are not yet connected to Linux device discovery. SMB/WebDAV/cloud are future providers. Linux device discovery, if enabled, must remain read-only by default and derive device identity from stable system IDs, never guessed card names.

Deleting real files must be implemented through recoverable Trash/undo journaling. No real volume mutation is currently exposed. Transfer copies must stream in bounded buffers; cross-filesystem moves require copy, fsync, atomic destination rename, then source removal with rollback handling.
