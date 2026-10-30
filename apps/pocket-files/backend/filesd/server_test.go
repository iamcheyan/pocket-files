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
	defer os.RemoveAll(filepath.Dir(p.roots["internal"]))
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
func TestSandboxRejectsTraversalAndSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	s := sandbox{map[string]string{"x": root}}
	for _, rel := range []string{"../outside", "escape/file"} {
		if _, err := s.resolve("x", rel, true); err == nil {
			t.Errorf("accepted unsafe path %q", rel)
		}
	}
	if _, err := s.resolve("x", "safe/new", true); err != nil {
		t.Fatalf("safe destination rejected: %v", err)
	}
}
func TestMockCreateRenameCopyMoveDelete(t *testing.T) {
	p, err := newMockProvider()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(p.roots["internal"]))
	ctx := t.Context()
	if err = p.CreateDir(ctx, "internal", "Documents/Test"); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(p.roots["internal"], "Documents/Test/a.txt"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = p.Copy(ctx, "internal", "Documents/Test/a.txt", "Documents/Test/copy.txt"); err != nil {
		t.Fatal(err)
	}
	if err = p.Move(ctx, "internal", "Documents/Test/copy.txt", "Documents/Test/moved.txt"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(p.roots["internal"], "Documents/Test/moved.txt"))
	if err != nil || string(data) != "hello" {
		t.Fatalf("copy content %q err %v", data, err)
	}
	if err = p.Delete(ctx, "internal", "Documents/Test/moved.txt"); err != nil {
		t.Fatal(err)
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
	defer os.RemoveAll(filepath.Dir(p.roots["internal"]))
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
	if _, err = os.Stat(filepath.Join(p.roots["internal"], "NewFolder")); err != nil {
		t.Fatal(err)
	}
}
