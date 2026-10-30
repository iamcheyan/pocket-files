package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestServer(t *testing.T) (*Server, *MockProvider, string) {
	t.Helper()
	p, err := newMockProvider()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(p.roots["internal"])) })
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	be := newBackend(p, db)
	be.recoverTransfers()
	return &Server{p: p, db: db, be: be}, p, filepath.Dir(p.roots["internal"])
}

func doJSON(t *testing.T, s *Server, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func doRaw(t *testing.T, s *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w
}

func writeFile(t *testing.T, root, rel string, data []byte) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, data, 0600); err != nil {
		t.Fatal(err)
	}
}

// waitJob polls until predicate is true or times out.
func waitJob(t *testing.T, s *Server, id string, pred func(map[string]any) bool, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, body := doJSON(t, s, http.MethodGet, "/api/transfers", nil)
		items, _ := body["items"].([]any)
		for _, it := range items {
			m, _ := it.(map[string]any)
			if m["id"] == id && pred(m) {
				return m
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("transfer %s did not reach expected state", id)
	return nil
}

func TestOperationsCopyMoveDeleteUndo(t *testing.T) {
	s, p, _ := newTestServer(t)
	writeFile(t, p.roots["internal"], "Documents/a.txt", []byte("hello world"))
	code, body := doJSON(t, s, http.MethodPost, "/api/operations", map[string]any{"action": "copy", "storage_id": "internal", "source": "Documents/a.txt", "destination": "Documents/b.txt"})
	if code != 200 {
		t.Fatalf("copy %d %v", code, body)
	}
	if b, e := os.ReadFile(filepath.Join(p.roots["internal"], "Documents/b.txt")); e != nil || string(b) != "hello world" {
		t.Fatalf("copy content %q %v", b, e)
	}
	code, _ = doJSON(t, s, http.MethodPost, "/api/operations", map[string]any{"action": "move", "storage_id": "internal", "source": "Documents/b.txt", "destination": "Documents/c.txt"})
	if code != 200 {
		t.Fatalf("move %d", code)
	}
	if _, e := os.Stat(filepath.Join(p.roots["internal"], "Documents/b.txt")); !os.IsNotExist(e) {
		t.Fatalf("source still exists")
	}
	code, delBody := doJSON(t, s, http.MethodPost, "/api/operations", map[string]any{"action": "delete", "storage_id": "internal", "source": "Documents/c.txt"})
	if code != 200 {
		t.Fatalf("delete %d %v", code, delBody)
	}
	if _, e := os.Stat(filepath.Join(p.roots["internal"], "Documents/c.txt")); !os.IsNotExist(e) {
		t.Fatalf("deleted file still present")
	}
	tp, _ := delBody["trash_path"].(string)
	if tp == "" {
		t.Fatalf("missing trash_path: %v", delBody)
	}
	if strings.Contains(tp, "/tmp/") {
		t.Fatalf("trash path leaked host dir: %q", tp)
	}
	code, undoBody := doJSON(t, s, http.MethodPost, "/api/operations/undo", map[string]any{"storage_id": "internal", "source": "Documents/c.txt", "trash_path": tp})
	if code != 200 {
		t.Fatalf("undo %d %v", code, undoBody)
	}
	if b, e := os.ReadFile(filepath.Join(p.roots["internal"], "Documents/c.txt")); e != nil || string(b) != "hello world" {
		t.Fatalf("restored content %q %v", b, e)
	}
}

func TestBulkDeleteRequiresConfirm(t *testing.T) {
	s, p, _ := newTestServer(t)
	writeFile(t, p.roots["internal"], "Documents/x.txt", []byte("x"))
	writeFile(t, p.roots["internal"], "Documents/y.txt", []byte("y"))
	code, body := doJSON(t, s, http.MethodPost, "/api/operations/bulk", map[string]any{"action": "delete", "storage_id": "internal", "sources": []string{"Documents/x.txt", "Documents/y.txt"}})
	if code != 400 {
		t.Fatalf("expected 400 without confirm, got %d %v", code, body)
	}
	if _, e := os.Stat(filepath.Join(p.roots["internal"], "Documents/x.txt")); e != nil {
		t.Fatal("file deleted without confirm")
	}
	code, body = doJSON(t, s, http.MethodPost, "/api/operations/bulk", map[string]any{"action": "delete", "storage_id": "internal", "confirm": true, "sources": []string{"Documents/x.txt", "Documents/y.txt"}})
	if code != 200 {
		t.Fatalf("bulk delete %d %v", code, body)
	}
	results, _ := body["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		m, _ := r.(map[string]any)
		if m["status"] != "ok" {
			t.Fatalf("bulk item failed: %v", m)
		}
	}
	if _, e := os.Stat(filepath.Join(p.roots["internal"], "Documents/x.txt")); !os.IsNotExist(e) {
		t.Fatalf("file still present")
	}
}

func TestFavoritesTagsRecentsSettingsPersistence(t *testing.T) {
	s, _, _ := newTestServer(t)
	// favorites
	if code, _ := doJSON(t, s, http.MethodPost, "/api/favorites", map[string]any{"storage_id": "internal", "uri": "Documents/a.txt"}); code != 200 {
		t.Fatalf("add favorite %d", code)
	}
	_, body := doJSON(t, s, http.MethodGet, "/api/favorites", nil)
	items, _ := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("favorites=%v", body)
	}
	if code, _ := doJSON(t, s, http.MethodDelete, "/api/favorites", map[string]any{"storage_id": "internal", "uri": "Documents/a.txt"}); code != 200 {
		t.Fatalf("del favorite %d", code)
	}
	_, body = doJSON(t, s, http.MethodGet, "/api/favorites", nil)
	if items, _ := body["items"].([]any); len(items) != 0 {
		t.Fatalf("favorites not cleared: %v", body)
	}
	// tags
	if code, _ := doJSON(t, s, http.MethodPost, "/api/tags", map[string]any{"storage_id": "internal", "uri": "Documents/a.txt", "tag": "important"}); code != 200 {
		t.Fatalf("add tag")
	}
	if code, _ := doJSON(t, s, http.MethodPut, "/api/tags", map[string]any{"storage_id": "internal", "uri": "Documents/a.txt", "tag": "archive"}); code != 200 {
		t.Fatalf("put tag")
	}
	_, body = doJSON(t, s, http.MethodGet, "/api/tags?storage_id=internal&uri=Documents/a.txt", nil)
	if items, _ := body["items"].([]any); len(items) != 2 {
		t.Fatalf("tags=%v", body)
	}
	if code, _ := doJSON(t, s, http.MethodDelete, "/api/tags", map[string]any{"storage_id": "internal", "uri": "Documents/a.txt", "tag": "important"}); code != 200 {
		t.Fatalf("del tag")
	}
	_, body = doJSON(t, s, http.MethodGet, "/api/tags?storage_id=internal&uri=Documents/a.txt", nil)
	if items, _ := body["items"].([]any); len(items) != 1 {
		t.Fatalf("tags after delete=%v", body)
	}
	// recents
	if e := s.be.recordRecent("internal", "Documents/a.txt", "file"); e != nil {
		t.Fatal(e)
	}
	_, body = doJSON(t, s, http.MethodGet, "/api/recents?kind=file", nil)
	if items, _ := body["items"].([]any); len(items) == 0 {
		t.Fatalf("recents empty")
	}
	if code, _ := doJSON(t, s, http.MethodDelete, "/api/recents", nil); code != 200 {
		t.Fatalf("clear recents")
	}
	_, body = doJSON(t, s, http.MethodGet, "/api/recents", nil)
	if items, _ := body["items"].([]any); len(items) != 0 {
		t.Fatalf("recents not cleared: %v", body)
	}
	// settings
	if code, _ := doJSON(t, s, http.MethodPut, "/api/settings", map[string]any{"privacy_mode": true, "theme": "dark"}); code != 200 {
		t.Fatalf("put settings")
	}
	_, body = doJSON(t, s, http.MethodGet, "/api/settings", nil)
	if body["theme"] != "dark" {
		t.Fatalf("settings not persisted: %v", body)
	}
	if body["mode"] != "mock" {
		t.Fatalf("settings mode missing: %v", body)
	}
}

func TestStorageFriendlyNameAndUsage(t *testing.T) {
	s, _, _ := newTestServer(t)
	code, body := doJSON(t, s, http.MethodPut, "/api/storage/internal", map[string]any{"friendly_name": "My Internal"})
	if code != 200 {
		t.Fatalf("rename %d %v", code, body)
	}
	_, body = doJSON(t, s, http.MethodGet, "/api/storage", nil)
	items, _ := body["items"].([]any)
	var found bool
	for _, it := range items {
		m, _ := it.(map[string]any)
		if m["id"] == "internal" {
			found = true
			if m["name"] != "My Internal" {
				t.Fatalf("friendly name not reflected: %v", m)
			}
		}
	}
	if !found {
		t.Fatal("internal not listed")
	}
	_, det := doJSON(t, s, http.MethodGet, "/api/storage/internal", nil)
	if det["friendly_name"] != "My Internal" {
		t.Fatalf("detail friendly_name=%v", det)
	}
	if det["mock"] != true {
		t.Fatalf("detail mock flag missing")
	}
	code, usage := doJSON(t, s, http.MethodGet, "/api/storage/internal/usage", nil)
	if code != 200 {
		t.Fatalf("usage %d %v", code, usage)
	}
	if _, ok := usage["total"].(float64); !ok {
		t.Fatalf("usage total missing: %v", usage)
	}
	if _, ok := usage["free"].(float64); !ok {
		t.Fatalf("usage free missing: %v", usage)
	}
	code, _ = doJSON(t, s, http.MethodPost, "/api/storage/internal/mount", nil)
	if code != 501 {
		t.Fatalf("mount expected 501, got %d", code)
	}
}

func TestFileMetadataPreviewViewsAndApps(t *testing.T) {
	s, p, _ := newTestServer(t)
	writeFile(t, p.roots["internal"], "Documents/note.md", []byte("# hi"))
	writeFile(t, p.roots["internal"], "Documents/pic.png", []byte("\x89PNG\r\n\x1a\n"))
	writeFile(t, p.roots["internal"], "Books/book.epub", []byte("book"))
	code, meta := doJSON(t, s, http.MethodGet, "/api/files/metadata/internal:Documents/note.md", nil)
	if code != 200 {
		t.Fatalf("metadata %d %v", code, meta)
	}
	if meta["name"] != "note.md" {
		t.Fatalf("metadata=%v", meta)
	}
	if _, ok := meta["id"]; !ok {
		t.Fatalf("metadata missing id")
	}
	w := doRaw(t, s, http.MethodGet, "/api/files/preview/internal:Documents/note.md", nil)
	if w.Code != 200 {
		t.Fatalf("preview status %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/markdown") {
		t.Fatalf("preview content-type %q", ct)
	}
	w = doRaw(t, s, http.MethodGet, "/api/files/preview/internal:Documents/pic.png", nil)
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("png content-type %q", ct)
	}
	w = doRaw(t, s, http.MethodGet, "/api/files/preview/internal:Books/book.epub", nil)
	if w.Code != 400 {
		t.Fatalf("unsupported preview should 400, got %d", w.Code)
	}
	for _, v := range []string{"books", "music", "videos", "games", "downloads", "recent", "favorites"} {
		code, body := doJSON(t, s, http.MethodGet, "/api/views/"+v, nil)
		if code != 200 {
			t.Fatalf("view %s => %d %v", v, code, body)
		}
	}
	_, body := doJSON(t, s, http.MethodGet, "/api/views/books", nil)
	if items, _ := body["items"].([]any); len(items) == 0 {
		t.Fatalf("books view empty")
	}
	_, body = doJSON(t, s, http.MethodGet, "/api/apps/open?storage_id=internal&path=Books/book.epub", nil)
	if body["status"] != "unavailable" {
		t.Fatalf("apps open=%v", body)
	}
}

func TestSearchFilterSort(t *testing.T) {
	s, p, _ := newTestServer(t)
	writeFile(t, p.roots["internal"], "Documents/small.txt", []byte("a"))
	writeFile(t, p.roots["internal"], "Documents/large.txt", bytes.Repeat([]byte("b"), 4096))
	code, body := doJSON(t, s, http.MethodGet, "/api/files?storage_id=internal&path=Documents&ext=txt&sort=size&order=desc", nil)
	if code != 200 {
		t.Fatalf("search %d %v", code, body)
	}
	items, _ := body["items"].([]any)
	if len(items) < 2 {
		t.Fatalf("expected >=2 items, got %d", len(items))
	}
	first, _ := items[0].(map[string]any)
	if first["name"] != "large.txt" {
		t.Fatalf("sort desc wrong: %v", first)
	}
	prev := -1.0
	for _, it := range items {
		m, _ := it.(map[string]any)
		sz, _ := m["size"].(float64)
		if prev >= 0 && sz > prev {
			t.Fatalf("sort desc not monotonic: %v", items)
		}
		prev = sz
	}
	// min_size filter
	code, body = doJSON(t, s, http.MethodGet, "/api/files?storage_id=internal&path=Documents&min_size=1000", nil)
	items, _ = body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("min_size filter: %v", body)
	}
	// type filter
	code, body = doJSON(t, s, http.MethodGet, "/api/files?storage_id=internal&path=Documents&type=file", nil)
	if code != 200 {
		t.Fatalf("type filter %d", code)
	}
	// name sort ascending stable
	code, body = doJSON(t, s, http.MethodGet, "/api/files?storage_id=internal&path=Documents&sort=name&order=asc", nil)
	items, _ = body["items"].([]any)
	names := make([]string, 0, len(items))
	for _, it := range items {
		m, _ := it.(map[string]any)
		names = append(names, m["name"].(string))
	}
	if !sort.StringsAreSorted(names) {
		t.Fatalf("name sort not sorted: %v", names)
	}
	// created field present
	_, body = doJSON(t, s, http.MethodGet, "/api/files?storage_id=internal&path=Documents", nil)
	items, _ = body["items"].([]any)
	if _, ok := items[0].(map[string]any)["created"]; !ok {
		t.Fatalf("created field missing")
	}
}

func TestAnalyticsCounts(t *testing.T) {
	s, p, _ := newTestServer(t)
	writeFile(t, p.roots["internal"], "Books/b.epub", []byte("book"))
	writeFile(t, p.roots["internal"], "Music/m.mp3", []byte("music"))
	writeFile(t, p.roots["internal"], "Videos/v.mp4", []byte("video"))
	writeFile(t, p.roots["internal"], "ROMs/r.gba", []byte("game"))
	writeFile(t, p.roots["internal"], "Documents/d.pdf", []byte("doc"))
	code, body := doJSON(t, s, http.MethodGet, "/api/analytics?storage_id=internal", nil)
	if code != 200 {
		t.Fatalf("analytics %d %v", code, body)
	}
	if body["mock"] != true {
		t.Fatalf("analytics mock flag missing")
	}
	cats, ok := body["category_counts"].(map[string]any)
	if !ok {
		t.Fatalf("category_counts missing: %v", body)
	}
	for _, c := range []string{"Books", "Music", "Videos", "Games", "Documents"} {
		if _, ok := cats[c]; !ok {
			t.Fatalf("category %s missing: %v", c, cats)
		}
	}
	if body["cache_timestamp"] == "" {
		t.Fatalf("cache_timestamp missing")
	}
}

func TestTransferConflictSkipReplaceKeepBothAndAsk(t *testing.T) {
	s, p, _ := newTestServer(t)
	writeFile(t, p.roots["internal"], "Documents/src.txt", []byte("new"))
	writeFile(t, p.roots["internal"], "Documents/dst.txt", []byte("old"))
	// skip: leave destination untouched
	code, body := doJSON(t, s, http.MethodPost, "/api/transfers", map[string]any{
		"id": "job-skip", "type": "copy", "storage_id": "internal", "conflict_strategy": "skip",
		"items": []map[string]any{{"src": "Documents/src.txt", "dst": "Documents/dst.txt"}},
	})
	if code != 201 {
		t.Fatalf("create skip %d %v", code, body)
	}
	waitJob(t, s, "job-skip", func(m map[string]any) bool { return m["status"] == "completed" }, 2*time.Second)
	if b, _ := os.ReadFile(filepath.Join(p.roots["internal"], "Documents/dst.txt")); string(b) != "old" {
		t.Fatalf("skip overwrote: %q", b)
	}
	// replace: destination becomes new
	code, _ = doJSON(t, s, http.MethodPost, "/api/transfers", map[string]any{
		"id": "job-replace", "type": "copy", "storage_id": "internal", "conflict_strategy": "replace",
		"items": []map[string]any{{"src": "Documents/src.txt", "dst": "Documents/dst.txt"}},
	})
	if code != 201 {
		t.Fatalf("create replace %d", code)
	}
	waitJob(t, s, "job-replace", func(m map[string]any) bool { return m["status"] == "completed" }, 2*time.Second)
	if b, _ := os.ReadFile(filepath.Join(p.roots["internal"], "Documents/dst.txt")); string(b) != "new" {
		t.Fatalf("replace failed: %q", b)
	}
	// keep_both: new file created
	code, _ = doJSON(t, s, "POST", "/api/transfers", map[string]any{
		"id": "job-both", "type": "copy", "storage_id": "internal", "conflict_strategy": "keep_both",
		"items": []map[string]any{{"src": "Documents/src.txt", "dst": "Documents/dst.txt"}},
	})
	if code != 201 {
		t.Fatalf("create keep_both %d", code)
	}
	waitJob(t, s, "job-both", func(m map[string]any) bool { return m["status"] == "completed" }, 2*time.Second)
	matches, _ := filepath.Glob(filepath.Join(p.roots["internal"], "Documents", "dst(*.txt"))
	if len(matches) == 0 {
		t.Fatalf("keep_both did not create a copy")
	}
	// ask: conflict state, resumable
	code, body = doJSON(t, s, http.MethodPost, "/api/transfers", map[string]any{
		"id": "job-ask", "type": "copy", "storage_id": "internal", "conflict_strategy": "ask",
		"items": []map[string]any{{"src": "Documents/src.txt", "dst": "Documents/dst.txt"}},
	})
	if code != 201 {
		t.Fatalf("create ask %d %v", code, body)
	}
	got := waitJob(t, s, "job-ask", func(m map[string]any) bool { return m["status"] == "conflict" }, 2*time.Second)
	if got["status"] != "conflict" {
		t.Fatalf("ask did not enter conflict: %v", got)
	}
	// resolve by switching to replace via retry after setting strategy through a fresh job is not exposed;
	// assert the file was not overwritten
	if b, _ := os.ReadFile(filepath.Join(p.roots["internal"], "Documents/dst.txt")); string(b) != "new" {
		t.Fatalf("ask overwrote unexpectedly: %q", b)
	}
}

func TestTransferPauseResumeCancelAndRecovery(t *testing.T) {
	s, p, _ := newTestServer(t)
	big := bytes.Repeat([]byte("z"), 8<<20) // 8MB, multi-chunk
	writeFile(t, p.roots["internal"], "Documents/big.bin", big)
	writeFile(t, p.roots["internal"], "Documents/second.bin", big)
	code, body := doJSON(t, s, http.MethodPost, "/api/transfers", map[string]any{
		"id": "job-pause", "type": "copy", "storage_id": "internal", "conflict_strategy": "replace",
		"items": []map[string]any{{"src": "Documents/big.bin", "dst": "Documents/copy1.bin"}},
	})
	if code != 201 {
		t.Fatalf("create pause job %d %v", code, body)
	}
	// pause quickly
	time.Sleep(2 * time.Millisecond)
	doJSON(t, s, http.MethodPost, "/api/transfers/job-pause/pause", nil)
	// wait for paused state or completion
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, b := doJSON(t, s, http.MethodGet, "/api/transfers", nil)
		st := statusOf(b, "job-pause")
		if st == "paused" || st == "completed" {
			break
		}
		time.Sleep(3 * time.Millisecond)
	}
	_, b := doJSON(t, s, http.MethodGet, "/api/transfers", nil)
	st := statusOf(b, "job-pause")
	if st != "paused" && st != "completed" {
		t.Fatalf("pause did not take effect: %s", st)
	}
	if st == "paused" {
		doJSON(t, s, http.MethodPost, "/api/transfers/job-pause/resume", nil)
		waitJob(t, s, "job-pause", func(m map[string]any) bool { return m["status"] == "completed" }, 5*time.Second)
	}
	if fi, e := os.Stat(filepath.Join(p.roots["internal"], "Documents/copy1.bin")); e != nil || fi.Size() != int64(len(big)) {
		t.Fatalf("paused copy incomplete: %v size=%d", e, fi.Size())
	}
	_ = big
	// cancel a second job
	code, _ = doJSON(t, s, http.MethodPost, "/api/transfers", map[string]any{
		"id": "job-cancel", "type": "copy", "storage_id": "internal", "conflict_strategy": "replace",
		"items": []map[string]any{{"src": "Documents/second.bin", "dst": "Documents/copy2.bin"}},
	})
	if code != 201 {
		t.Fatalf("create cancel job %d", code)
	}
	doJSON(t, s, http.MethodPost, "/api/transfers/job-cancel/cancel", nil)
	waitJob(t, s, "job-cancel", func(m map[string]any) bool { return m["status"] == "cancelled" }, 2*time.Second)
	// crash recovery: insert a running job directly, then recover
	now := time.Now().UTC().Format(time.RFC3339)
	if _, e := s.db.Exec(`INSERT INTO transfer_jobs(id,type,storage_id,status,conflict_strategy,extra,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
		"job-crash", "copy", "internal", "running", "skip", "{}", now, now); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.Exec(`INSERT INTO transfer_items(job_id,item_index,src,dst,state,error) VALUES(?,?,?,?,?,?)`,
		"job-crash", 0, "Documents/big.bin", "Documents/crash.bin", "running", ""); e != nil {
		t.Fatal(e)
	}
	s.be.tc.recover()
	_, b = doJSON(t, s, http.MethodGet, "/api/transfers", nil)
	st = statusOf(b, "job-crash")
	if st != "failed" {
		t.Fatalf("crash recovery status=%s want failed", st)
	}
}

func statusOf(body map[string]any, id string) string {
	items, _ := body["items"].([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		if m["id"] == id {
			s, _ := m["status"].(string)
			return s
		}
	}
	return ""
}

func TestNoPathLeakAcrossEndpoints(t *testing.T) {
	s, p, base := newTestServer(t)
	writeFile(t, p.roots["internal"], "Documents/a.txt", []byte("hello"))
	_ = base
	writeFile(t, p.roots["internal"], "Books/b.epub", []byte("book"))
	paths := []string{
		"/api/health", "/api/storage", "/api/storage/internal", "/api/storage/internal/usage",
		"/api/files?storage_id=internal&path=Documents", "/api/files?storage_id=internal&q=a",
		"/api/folders", "/api/favorites", "/api/tags", "/api/recents", "/api/settings",
		"/api/downloads", "/api/transfers", "/api/analytics?storage_id=internal",
		"/api/views/books", "/api/views/recent", "/api/views/favorites",
		"/api/apps/open?storage_id=internal&path=Documents/a.txt",
	}
	var mu sync.Mutex
	for _, path := range paths {
		w := doRaw(t, s, http.MethodGet, path, nil)
		mu.Lock()
		body := w.Body.String()
		mu.Unlock()
		if strings.Contains(body, "/tmp/") || strings.Contains(body, p.roots["internal"]) || strings.Contains(body, "pocket-files-mock") {
			t.Errorf("path leak at %s: %s", path, body)
		}
		if strings.Contains(body, "\"Root\"") || strings.Contains(body, "\"root\"") {
			t.Errorf("root field leaked at %s", path)
		}
	}
	// metadata/preview via raw path
	for _, path := range []string{"/api/files/metadata/internal:Documents/a.txt", "/api/files/preview/internal:Documents/a.txt"} {
		w := doRaw(t, s, http.MethodGet, path, nil)
		body := w.Body.String()
		if strings.Contains(body, p.roots["internal"]) {
			t.Errorf("path leak at %s", path)
		}
		if strings.Contains(w.Header().Get("Content-Type"), p.roots["internal"]) {
			t.Errorf("header leak at %s", path)
		}
	}
	// operations + transfers create with bad path must not echo host paths
	code, body := doJSON(t, s, http.MethodPost, "/api/operations", map[string]any{"action": "copy", "storage_id": "internal", "source": "Documents/a.txt", "destination": "Documents/a2.txt"})
	if code != 200 {
		t.Fatalf("op copy %d", code)
	}
	if js, _ := json.Marshal(body); strings.Contains(string(js), p.roots["internal"]) {
		t.Errorf("operation leaked path")
	}
}

var _ = fmt.Sprintf

func TestFileRenameContract(t *testing.T) {
	s, p, _ := newTestServer(t)
	writeFile(t, p.roots["internal"], "Documents/old.txt", []byte("payload"))
	code, body := doJSON(t, s, http.MethodPut, "/api/files/internal%3ADocuments%2Fold.txt/rename", map[string]any{"name": "new.txt"})
	if code != 200 {
		t.Fatalf("rename status %d %v", code, body)
	}
	if _, err := os.Stat(filepath.Join(p.roots["internal"], "Documents", "new.txt")); err != nil {
		t.Fatalf("renamed file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(p.roots["internal"], "Documents", "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old name still present: %v", err)
	}
	for _, bad := range []string{"../escape.txt", "a/b.txt", ""} {
		code, _ = doJSON(t, s, http.MethodPut, "/api/files/internal%3ADocuments%2Fnew.txt/rename", map[string]any{"name": bad})
		if code != 400 {
			t.Errorf("rename %q should be rejected, got %d", bad, code)
		}
	}
}
