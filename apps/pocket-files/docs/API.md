# API

Base URL: `http://127.0.0.1:8787`. Errors use `{"error":{"code":"...","message":"..."}}`.

Implemented endpoints: `GET /api/health`, `GET /api/storage`, `GET /api/files?storage_id=&path=&q=&ext=&page=&limit=`, `POST /api/folders`, `GET /api/favorites`, `GET /api/recents`, `GET /api/views/{name}`, `GET /api/downloads`, `GET /api/settings`, and `GET /api/transfers`. `downloads` currently returns 503 with unavailable status; a downloadsd client is not connected.

The remaining endpoints in the product contract—storage detail/usage and confirmed lifecycle operations, file metadata/preview, rename, safe copy/move/delete/undo and bulk actions, durable transfer controls, favorites/tags/recents mutations, settings writes—are not implemented yet. Clients must not infer support from this document; inspect endpoint availability.

Entry paths are provider-relative IDs and host paths are never part of the response schema. Current writes affect only generated mock storage.
