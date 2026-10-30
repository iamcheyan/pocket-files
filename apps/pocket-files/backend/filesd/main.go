package main

import (
	"database/sql"
	"encoding/json"
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
	p  *MockProvider
	db *sql.DB
	be *Backend
}

func (s *Server) storageItems(devices []Device) []storageDTO {
	out := make([]storageDTO, 0, len(devices))
	for _, d := range devices {
		name := d.Name
		if s.be != nil {
			name = s.be.getFriendlyName(d.ID)
		}
		out = append(out, storageDTO{ID: d.ID, Name: name, Kind: d.Kind, Status: d.Status, Mock: d.Mock, ReadOnly: d.ReadOnly, Total: d.Total, Used: d.Used})
	}
	return out
}
func entryItems(entries []Entry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		out = append(out, map[string]any{
			"id": e.ID, "storage_id": e.StorageID, "name": e.Name, "parent": e.Parent, "kind": e.Kind,
			"size": e.Size, "modified": e.Modified, "created": e.Created, "extension": e.Extension,
		})
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
			s.json(w, 503, map[string]any{"status": "unavailable", "source": "downloadsd"})
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
		}
	}
	s.error(w, 404, "not_found", "endpoint not found")
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 2 && r.Method == "GET" {
		id := r.URL.Query().Get("storage_id")
		q := r.URL.Query().Get("q")
		pathQ := r.URL.Query().Get("path")
		var items []Entry
		var err error
		if q != "" {
			items, err = s.p.Search(r.Context(), id, q)
		} else {
			items, err = s.p.Browse(r.Context(), id, pathQ)
		}
		if err != nil {
			s.error(w, 400, "storage_error", err.Error())
			return
		}
		if q == "" && pathQ != "" {
			_ = s.be.recordRecent(id, pathQ, "directory")
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
	// Canonical contract: GET /api/files/{id}/metadata and /api/files/{id}/preview,
	// where {id} is the URL-escaped "storage:path" file identifier.
	if len(parts) >= 4 && r.Method == "GET" && (parts[len(parts)-1] == "metadata" || parts[len(parts)-1] == "preview") {
		action := parts[len(parts)-1]
		raw := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/files/"), "/"+action)
		if id, err := url.PathUnescape(raw); err == nil {
			raw = id
		}
		if strings.Contains(raw, ":") {
			if action == "metadata" {
				s.handleFileMetadata(w, r, raw)
			} else {
				s.handleFilePreview(w, r, raw)
			}
			return
		}
	}
	if len(parts) >= 4 && (parts[2] == "metadata" || parts[2] == "preview") && r.Method == "GET" {
		raw := strings.TrimPrefix(r.URL.Path, "/api/files/"+parts[2]+"/")
		id, err := url.PathUnescape(raw)
		if err != nil {
			id = raw
		}
		if parts[2] == "metadata" {
			s.handleFileMetadata(w, r, id)
		} else {
			s.handleFilePreview(w, r, id)
		}
		return
	}
	s.error(w, 404, "not_found", "endpoint not found")
}

func (s *Server) handleStorage(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 3 && parts[2] == "detected" && r.Method == "GET" {
		s.handleDetectedStorage(w, r)
		return
	}
	if len(parts) == 2 && r.Method == "GET" {
		d, _ := s.p.Discover(r.Context())
		s.json(w, 200, map[string]any{"items": s.storageItems(d)})
		return
	}
	if len(parts) == 4 && r.Method == "GET" && parts[3] == "usage" {
		s.handleStorageUsage(w, r, parts[2])
		return
	}
	if len(parts) == 4 && r.Method == "POST" && (parts[3] == "mount" || parts[3] == "unmount") {
		s.error(w, 501, "not_implemented", "mock storage mount/unmount is not implemented")
		return
	}
	if len(parts) == 3 {
		s.handleStorageItem(w, r, parts[2])
		return
	}
	s.error(w, 404, "not_found", "endpoint not found")
}

func (s *Server) handleStorageItem(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == "GET" {
		d, ok := s.be.getDevice(id)
		if !ok {
			s.error(w, 404, "storage_not_found", "unknown storage")
			return
		}
		detail := map[string]any{
			"id": d.ID, "name": d.Name, "kind": d.Kind, "status": d.Status, "mock": d.Mock,
			"read_only": d.ReadOnly, "total": d.Total, "used": d.Used, "friendly_name": s.be.getFriendlyName(id),
		}
		s.json(w, 200, detail)
		return
	}
	if r.Method == "PUT" {
		var body struct {
			FriendlyName string `json:"friendly_name"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			s.error(w, 400, "invalid_json", "invalid request")
			return
		}
		if e := s.be.updateFriendlyName(id, body.FriendlyName); e != nil {
			s.error(w, 400, "storage_error", e.Error())
			return
		}
		s.json(w, 200, map[string]any{"id": id, "friendly_name": body.FriendlyName})
		return
	}
	if r.Method == "POST" {
		action := strings.Trim(strings.TrimPrefix(r.URL.Path, "api/storage/"+id+"/"), "/")
		if action == "mount" || action == "unmount" {
			s.error(w, 501, "not_implemented", "mock storage mount/unmount is not implemented")
			return
		}
	}
	s.error(w, 404, "not_found", "endpoint not found")
}

func (s *Server) handleStorageUsage(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != "GET" {
		s.error(w, 405, "method_not_allowed", "use GET")
		return
	}
	total, used, free, e := s.be.storageUsage(id)
	if e != nil {
		s.error(w, 400, "storage_error", e.Error())
		return
	}
	d, _ := s.be.getDevice(id)
	s.json(w, 200, map[string]any{"id": id, "mock": d.Mock, "total": total, "used": used, "free": free})
}

func (s *Server) handleFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.error(w, 405, "method_not_allowed", "use POST")
		return
	}
	var v struct {
		StorageID string `json:"storage_id"`
		Path      string `json:"path"`
	}
	if json.NewDecoder(r.Body).Decode(&v) != nil {
		s.error(w, 400, "invalid_json", "invalid request")
		return
	}
	if e := s.p.CreateDir(r.Context(), v.StorageID, v.Path); e != nil {
		s.error(w, 400, "operation_failed", e.Error())
		return
	}
	w.WriteHeader(201)
}

func (s *Server) handleViews(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 3 && r.Method == "GET" {
		items, err := s.be.getView(r.Context(), parts[2])
		if err != nil {
			s.error(w, 400, "view_error", err.Error())
			return
		}
		s.json(w, 200, map[string]any{"view": parts[2], "items": items})
		return
	}
	s.error(w, 501, "not_implemented", "virtual view is not available for this provider yet")
}

func (s *Server) handleApps(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 2 && r.Method == "GET" {
		s.json(w, 200, map[string]any{"status": "unavailable", "reason": "pj_app_registry_not_installed"})
		return
	}
	if len(parts) >= 3 && parts[2] == "open" && r.Method == "GET" {
		s.json(w, 200, map[string]any{"status": "unavailable"})
		return
	}
	s.error(w, 404, "not_found", "endpoint not found")
}

func openDB(path string) (*sql.DB, error) {
	db, e := sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	queries := []string{
		`PRAGMA busy_timeout=5000`,
		`PRAGMA journal_mode=WAL`,
		`PRAGMA foreign_keys=ON`,
		`CREATE TABLE IF NOT EXISTS schema_version(version INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS storage_devices(id TEXT PRIMARY KEY,friendly_name TEXT NOT NULL,kind TEXT NOT NULL,status TEXT NOT NULL,mock INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE IF NOT EXISTS directories(id TEXT PRIMARY KEY,storage_id TEXT NOT NULL,parent_id TEXT,name TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS favorites(id INTEGER PRIMARY KEY,storage_id TEXT NOT NULL,uri TEXT NOT NULL,UNIQUE(storage_id,uri))`,
		`CREATE TABLE IF NOT EXISTS recent_files(id INTEGER PRIMARY KEY,storage_id TEXT NOT NULL,uri TEXT NOT NULL,kind TEXT NOT NULL,accessed_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS transfer_jobs(id TEXT PRIMARY KEY,type TEXT NOT NULL,storage_id TEXT NOT NULL,status TEXT NOT NULL,conflict_strategy TEXT NOT NULL DEFAULT 'skip',extra TEXT NOT NULL DEFAULT '{}',created_at TEXT NOT NULL,updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS transfer_items(job_id TEXT NOT NULL,item_index INTEGER NOT NULL,src TEXT NOT NULL,dst TEXT NOT NULL,state TEXT NOT NULL,error TEXT NOT NULL DEFAULT '',PRIMARY KEY(job_id,item_index))`,
		`CREATE TABLE IF NOT EXISTS transfer_order(position INTEGER PRIMARY KEY,id TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY,value TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS file_tags(id INTEGER PRIMARY KEY,storage_id TEXT NOT NULL,uri TEXT NOT NULL,tag TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS custom_locations(id INTEGER PRIMARY KEY,name TEXT NOT NULL,storage_id TEXT NOT NULL,uri TEXT NOT NULL)`,
	}
	for _, q := range queries {
		if _, e = db.Exec(q); e != nil {
			db.Close()
			return nil, e
		}
	}
	return db, nil
}

func main() {
	p, e := newMockProvider()
	if e != nil {
		log.Fatal(e)
	}
	db, e := openDB(env("POCKET_FILES_DB", "pocket-files.db"))
	if e != nil {
		log.Fatal(e)
	}
	defer db.Close()
	be := newBackend(p, db)
	be.recoverTransfers()
	// wire storage usage routes via a custom mux on top of ServeHTTP is unnecessary: handled in handleStorageItem region.
	addr := env("POCKET_FILES_ADDR", "127.0.0.1:8787")
	log.Printf("filesd listening on %s (mock mode)", addr)
	if e = http.ListenAndServe(addr, &Server{p: p, db: db, be: be}); e != nil {
		log.Fatal(e)
	}
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func parts(p string) []string { return strings.Split(strings.Trim(p, "/"), "/") }
