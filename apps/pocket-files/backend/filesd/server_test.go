package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMockVolumesAndGeneratedExamples(t *testing.T) {
	p, err := newMockProvider()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(p.MockRoot("internal")))
	devices, err := p.Discover(t.Context())
	if err != nil || len(devices) != 4 {
		t.Fatalf("devices=%d err=%v", len(devices), err)
	}
	seen := map[string]bool{}
	for _, d := range devices {
		seen[d.ID] = d.Mock
		if strings.Contains(d.Name, "mmcblk") || strings.Contains(d.Name, "/dev/") {
			t.Fatalf("device node leaked: %s", d.Name)
		}
	}
	for _, id := range []string{"internal", "tf-a", "tf-b", "usb"} {
		if !seen[id] {
			t.Fatalf("missing mock %s", id)
		}
		items, e := p.Browse(t.Context(), id, "")
		if e != nil || len(items) < 6 {
			t.Fatalf("%s examples: %d %v", id, len(items), e)
		}
	}
}
func TestSQLiteSchemaStartup(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string]bool{}
	for rows.Next() {
		var n string
		if err = rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		got[n] = true
	}
	for _, name := range []string{"storage_devices", "directories", "favorites", "recent_files", "transfer_jobs", "settings", "file_tags", "custom_locations"} {
		if !got[name] {
			t.Errorf("missing table %s", name)
		}
	}
}
func TestHealthStorageBrowseAndUnavailableDownloads(t *testing.T) {
	p, err := newMockProvider()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(p.MockRoot("internal")))
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{p: p, db: db}
	for _, tc := range []struct {
		path   string
		status int
		needle string
	}{{"/api/health", 200, "mock"}, {"/api/storage", 200, "tf-a"}, {"/api/files?storage_id=internal", 200, "Books"}, {"/api/downloads", 503, "unavailable"}} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.needle) {
			t.Errorf("%s => %d %s", tc.path, w.Code, w.Body.String())
		}
	}
	body := bytes.NewBufferString(`{"storage_id":"internal","path":"NewFolder"}`)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/folders", body))
	if w.Code != 201 {
		t.Fatalf("create folder status=%d body=%s", w.Code, w.Body.String())
	}
	if _, err = os.Stat(filepath.Join(p.MockRoot("internal"), "NewFolder")); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationsAreVersionedAndIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migrate.db")
	db, err := openDB(path)
	if err != nil {
		t.Fatal(err)
	}
	var version int
	if err = db.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != schemaVersion {
		t.Fatalf("schema version = %d, want %d", version, schemaVersion)
	}
	var indexes int
	if err = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%'`).Scan(&indexes); err != nil {
		t.Fatal(err)
	}
	if indexes < 5 {
		t.Fatalf("expected created indexes, found %d", indexes)
	}
	db.Close()

	// Re-opening must not duplicate version rows or fail on existing indexes.
	db, err = openDB(path)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer db.Close()
	var rows int
	if err = db.QueryRow(`SELECT COUNT(*) FROM schema_version`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != schemaVersion {
		t.Fatalf("schema_version rows = %d, want %d", rows, schemaVersion)
	}
	if err = db.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&version); err != nil || version != schemaVersion {
		t.Fatalf("version after reopen = %d (%v)", version, err)
	}
}

func TestMigrationRefusesNewerDatabase(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "future.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`INSERT INTO schema_version(version) VALUES(?)`, schemaVersion+5); err != nil {
		t.Fatal(err)
	}
	if err = migrate(db); err == nil {
		t.Fatalf("expected refusal for a newer schema version")
	}
}
