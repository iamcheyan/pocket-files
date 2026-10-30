# Database

`filesd` uses modernc.org/sqlite (pure Go). Startup creates version and application tables with foreign-key enforcement enabled: `storage_devices`, `directories`, `favorites`, `recent_files`, `transfer_jobs`, `settings`, `file_tags`, and `custom_locations`. SQL values are passed through parameters. Schema upgrades must increment `schema_version` transactionally and keep migrations repeatable.

Privacy mode and retention controls are not implemented yet. The database is local; it should store provider-relative references only. Do not store unapproved absolute roots or host paths in search indexes. The default DB path is `pocket-files.db`; configure with `POCKET_FILES_DB`.
