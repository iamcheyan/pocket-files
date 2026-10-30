package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type storageDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Status   string `json:"status"`
	Mock     bool   `json:"mock"`
	ReadOnly bool   `json:"read_only"`
	Total    int64  `json:"total"`
	Used     int64  `json:"used"`
}
type Server struct {
	p         storageBackend
	db        *sql.DB
	be        *Backend
	apps      *appRegistry
	downloads *downloadsClient
}

func (s *Server) storageItems(devices []Device) []storageDTO {
	out := make([]storageDTO, 0, len(devices))
	for _, d := range devices {
		name := d.Name
		if s.be != nil {
			name = s.be.getFriendlyName(d.ID)
		}
		used := d.Used
		if d.Mock {
			used = directoryUsage(d.Root)
		}
		out = append(out, storageDTO{ID: d.ID, Name: name, Kind: d.Kind, Status: d.Status, Mock: d.Mock, ReadOnly: d.ReadOnly, Total: d.Total, Used: used})
	}
	return out
}
func entryItems(entries []Entry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		out = append(out, map[string]any{"id": e.ID, "storage_id": e.StorageID, "name": e.Name, "parent": e.Parent, "kind": e.Kind, "size": e.Size, "modified": e.Modified, "created": e.Created, "extension": e.Extension})
	}
	return out
}
func (s *Server) json(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func (s *Server) error(w http.ResponseWriter, status int, code, msg string) {
	s.json(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if path == "api/health" {
		s.json(w, 200, map[string]any{"status": "ok", "mode": "mock"})
		return
	}
	if len(parts) >= 2 && parts[0] == "api" {
		switch parts[1] {
		case "storage":
			s.handleStorage(w, r, parts)
			return
		case "files":
			s.handleFiles(w, r, parts)
			return
		case "folders":
			s.handleFolders(w, r)
			return
		case "favorites":
			s.handleFavorites(w, r)
			return
		case "tags":
			s.handleTags(w, r)
			return
		case "recents":
			s.handleRecents(w, r)
			return
		case "settings":
			s.handleSettings(w, r)
			return
		case "views":
			s.handleViews(w, r, parts)
			return
		case "downloads":
			s.handleDownloads(w, r)
			return
		case "transfers":
			s.handleTransfers(w, r, parts)
			return
		case "operations":
			s.handleOperations(w, r, parts)
			return
		case "analytics":
			s.handleAnalytics(w, r)
			return
		case "apps":
			s.handleApps(w, r, parts)
			return
		case "locations":
			s.handlePinnedLocations(w, r, parts)
			return
		case "providers":
			s.handleCapabilities(w, r)
			return
		}
	}
	s.error(w, 404, "not_found", "endpoint not found")
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 2 && r.Method == http.MethodGet {
		id := r.URL.Query().Get("storage_id")
		q := r.URL.Query().Get("q")
		var items []Entry
		var err error
		if q != "" {
			items, err = s.p.Search(r.Context(), id, q)
		} else {
			items, err = s.p.Browse(r.Context(), id, r.URL.Query().Get("path"))
		}
		if err != nil {
			s.storageError(w, "storage_error", err)
			return
		}
		if q == "" && r.URL.Query().Get("path") != "" {
			_ = s.be.recordRecent(id, r.URL.Query().Get("path"), "directory")
		}
		items = s.be.applyFileFilters(items, r.URL.Query())
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 500 {
			limit = 100
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 0 {
			page = 0
		}
		total := len(items)
		start := page * limit
		if start > total {
			start = total
		}
		items = items[start:]
		if len(items) > limit {
			items = items[:limit]
		}
		s.json(w, 200, map[string]any{"items": entryItems(items), "page": page, "limit": limit, "total": total})
		return
	}
	if action, id, ok := fileActionFromPath(r.URL.EscapedPath()); ok {
		switch action {
		case "metadata":
			if r.Method == http.MethodGet {
				s.handleFileMetadata(w, r, id)
				return
			}
		case "preview":
			if r.Method == http.MethodGet {
				s.handleFilePreview(w, r, id)
				return
			}
		case "rename":
			if r.Method == http.MethodPut {
				s.handleFileRename(w, r, id)
				return
			}
		}
	}
	s.error(w, 404, "not_found", "endpoint not found")
}

func (s *Server) handleStorage(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 3 && parts[2] == "detected" && r.Method == http.MethodGet {
		s.handleDetectedStorage(w, r)
		return
	}
	if len(parts) == 2 && r.Method == http.MethodGet {
		d, _ := s.p.Discover(r.Context())
		s.json(w, 200, map[string]any{"items": s.storageItems(d)})
		return
	}
	if len(parts) == 4 && r.Method == http.MethodGet && parts[3] == "usage" {
		s.handleStorageUsage(w, r, parts[2])
		return
	}
	if len(parts) == 4 && r.Method == http.MethodPost && (parts[3] == "mount" || parts[3] == "unmount") {
		// Lifecycle operations stay explicitly unimplemented: nothing may mount
		// or unmount a host volume on a user's behalf without confirmation.
		s.error(w, http.StatusNotImplemented, "not_implemented", "storage lifecycle operations are not implemented")
		return
	}
	if len(parts) == 3 {
		s.handleStorageItem(w, r, parts[2])
		return
	}
	s.error(w, 404, "not_found", "endpoint not found")
}

func openDB(path string) (*sql.DB, error) {
	db, e := sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}
	queries := []string{`PRAGMA foreign_keys=ON`, `PRAGMA journal_mode=WAL`, `PRAGMA busy_timeout=5000`, `CREATE TABLE IF NOT EXISTS schema_version(version INTEGER NOT NULL)`, `CREATE TABLE IF NOT EXISTS storage_devices(id TEXT PRIMARY KEY,friendly_name TEXT NOT NULL,kind TEXT NOT NULL,status TEXT NOT NULL,mock INTEGER NOT NULL DEFAULT 0)`, `CREATE TABLE IF NOT EXISTS directories(id TEXT PRIMARY KEY,storage_id TEXT NOT NULL,parent_id TEXT,name TEXT NOT NULL)`, `CREATE TABLE IF NOT EXISTS favorites(id INTEGER PRIMARY KEY,storage_id TEXT NOT NULL,uri TEXT NOT NULL,UNIQUE(storage_id,uri))`, `CREATE TABLE IF NOT EXISTS recent_files(id INTEGER PRIMARY KEY,storage_id TEXT NOT NULL,uri TEXT NOT NULL,kind TEXT NOT NULL,accessed_at TEXT NOT NULL)`, `CREATE TABLE IF NOT EXISTS transfer_jobs(id TEXT PRIMARY KEY,type TEXT NOT NULL,storage_id TEXT NOT NULL,status TEXT NOT NULL,conflict_strategy TEXT NOT NULL,extra TEXT NOT NULL,created_at TEXT NOT NULL,updated_at TEXT NOT NULL)`, `CREATE TABLE IF NOT EXISTS transfer_items(job_id TEXT NOT NULL,item_index INTEGER NOT NULL,src TEXT NOT NULL,dst TEXT NOT NULL,state TEXT NOT NULL,error TEXT NOT NULL DEFAULT '',PRIMARY KEY(job_id,item_index))`, `CREATE TABLE IF NOT EXISTS transfer_order(position INTEGER PRIMARY KEY,id TEXT NOT NULL)`, `CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY,value TEXT NOT NULL)`, `CREATE TABLE IF NOT EXISTS file_tags(id INTEGER PRIMARY KEY,storage_id TEXT NOT NULL,uri TEXT NOT NULL,tag TEXT NOT NULL)`, `CREATE TABLE IF NOT EXISTS custom_locations(id INTEGER PRIMARY KEY,name TEXT NOT NULL,storage_id TEXT NOT NULL,uri TEXT NOT NULL)`}
	for _, q := range queries {
		if _, e = db.Exec(q); e != nil {
			db.Close()
			return nil, e
		}
	}
	db.SetMaxOpenConns(1)
	if e = migrate(db); e != nil {
		db.Close()
		return nil, e
	}
	return db, nil
}

// schemaVersion is bumped whenever a migration is added. Migrations are applied
// in order inside a transaction and recorded in schema_version, so a partially
// initialized database upgrades safely and a newer database is refused.
const schemaVersion = 2

func migrate(db *sql.DB) error {
	var current int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_version`).Scan(&current); err != nil {
		return err
	}
	if current > schemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported %d", current, schemaVersion)
	}
	migrations := map[int][]string{
		1: {
			`CREATE INDEX IF NOT EXISTS idx_favorites_storage ON favorites(storage_id)`,
			`CREATE INDEX IF NOT EXISTS idx_recents_accessed ON recent_files(accessed_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_tags_storage_uri ON file_tags(storage_id, uri)`,
		},
		2: {
			`CREATE INDEX IF NOT EXISTS idx_transfer_jobs_updated ON transfer_jobs(updated_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_transfer_items_job ON transfer_items(job_id)`,
		},
	}
	for version := current + 1; version <= schemaVersion; version++ {
		statements, ok := migrations[version]
		if !ok {
			return fmt.Errorf("missing migration for schema version %d", version)
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		for _, statement := range statements {
			if _, err = tx.Exec(statement); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration %d: %w", version, err)
			}
		}
		if _, err = tx.Exec(`INSERT INTO schema_version(version) VALUES(?)`, version); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", version, err)
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	mock, e := newMockProvider()
	if e != nil {
		log.Fatal(e)
	}
	cfg, e := loadProviderConfig(env("POCKET_FILES_CONFIG", ""))
	if e != nil {
		log.Fatal(e)
	}
	providers, e := newProviderRouter(mock, cfg)
	if e != nil {
		log.Fatal(e)
	}
	db, e := openDB(env("POCKET_FILES_DB", "pocket-files.db"))
	if e != nil {
		log.Fatal(e)
	}
	defer db.Close()
	be := newBackend(providers, db)
	be.recoverTransfers()
	addr := env("POCKET_FILES_ADDR", "127.0.0.1:8787")
	log.Printf("filesd listening on %s (%d mock volumes, %d configured, smb/webdav/cloud not implemented)", addr, len(mock.devices), len(cfg.Storages))
	if e = http.ListenAndServe(addr, &Server{p: providers, db: db, be: be, apps: newAppRegistry(), downloads: newDownloadsClient()}); e != nil {
		log.Fatal(e)
	}
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// fileActionFromPath extracts the action and file identifier from both the
// canonical route (/api/files/{id}/metadata) and the legacy route
// (/api/files/metadata/{id}). The escaped path is used so an encoded ":" or "/"
// inside the identifier is never split by net/http's path decoding.
func fileActionFromPath(escapedPath string) (action, id string, ok bool) {
	rest, found := strings.CutPrefix(escapedPath, "/api/files/")
	if !found {
		return "", "", false
	}
	for _, candidate := range []string{"metadata", "preview", "rename"} {
		// Canonical form: {id}/{action}. Legacy form: {action}/{id}.
		if trimmed, matched := strings.CutSuffix(rest, "/"+candidate); matched {
			if decoded := decodeID(trimmed); decoded != "" {
				return candidate, decoded, true
			}
		}
		if trimmed, matched := strings.CutPrefix(rest, candidate+"/"); matched {
			if decoded := decodeID(trimmed); decoded != "" {
				return candidate, decoded, true
			}
		}
	}
	return "", "", false
}

func decodeID(raw string) string {
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return decoded
}
