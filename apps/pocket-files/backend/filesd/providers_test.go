package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestRouter(t *testing.T) (*providerRouter, string, *MockProvider) {
	t.Helper()
	mock, err := newMockProvider()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(mock.roots["internal"])) })
	root := t.TempDir()
	cfg := ProviderConfig{Storages: []StorageEntry{
		{ID: "local-internal", FriendlyName: "Onboard", Kind: "internal", Root: root},
	}}
	router, err := newProviderRouter(mock, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return router, root, mock
}

func TestProviderContractAcrossFamilies(t *testing.T) {
	router, root, _ := newTestRouter(t)
	ctx := context.Background()

	for _, id := range []string{"internal", "tf-a", "tf-b", "usb", "local-internal"} {
		if _, err := router.Browse(ctx, id, ""); err != nil {
			t.Fatalf("%s browse: %v", id, err)
		}
		if err := router.CreateDir(ctx, id, "ContractTest"); err != nil {
			t.Fatalf("%s mkdir: %v", id, err)
		}
		rootPath, err := router.resolve(id, "ContractTest", false)
		if err != nil {
			t.Fatalf("%s resolve: %v", id, err)
		}
		if err = os.WriteFile(filepath.Join(rootPath, "a.txt"), []byte("payload"), 0600); err != nil {
			t.Fatal(err)
		}
		entry, err := router.Stat(ctx, id, "ContractTest/a.txt")
		if err != nil || entry.Size != 7 {
			t.Fatalf("%s stat: %+v %v", id, entry, err)
		}
		if err = router.Copy(ctx, id, "ContractTest/a.txt", "ContractTest/b.txt"); err != nil {
			t.Fatalf("%s copy: %v", id, err)
		}
		if err = router.Move(ctx, id, "ContractTest/b.txt", "ContractTest/c.txt"); err != nil {
			t.Fatalf("%s move: %v", id, err)
		}
		if _, err = router.Trash(ctx, id, "ContractTest/c.txt"); err != nil {
			t.Fatalf("%s trash: %v", id, err)
		}
		if err = router.Delete(ctx, id, "ContractTest"); err != nil {
			t.Fatalf("%s delete: %v", id, err)
		}
	}
	if _, err := router.Browse(ctx, "smb", ""); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("smb browse should be not implemented, got %v", err)
	}
	if _, _, err := router.AvailableSpace(ctx, "webdav"); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("webdav space should be not implemented, got %v", err)
	}
	if _, err := router.resolve("cloud", "x", false); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("cloud resolve should be not implemented, got %v", err)
	}
	if _, err := router.resolve("local-internal", "../escape", false); err == nil {
		t.Fatalf("local provider accepted traversal")
	}
	_ = root
}

func TestProviderStubsAreAdvertised(t *testing.T) {
	router, _, _ := newTestRouter(t)
	devices, err := router.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, d := range devices {
		status[d.ID] = d.Status
	}
	for _, id := range []string{"smb", "webdav", "cloud"} {
		if status[id] != "not_implemented" {
			t.Errorf("%s status = %q, want not_implemented", id, status[id])
		}
	}
}

func TestLocalProviderUsesConfiguredRootNotCallerPath(t *testing.T) {
	router, root, _ := newTestRouter(t)
	ctx := context.Background()
	entry, err := router.Stat(ctx, "local-internal", "")
	if err != nil {
		t.Fatal(err)
	}
	if entry.StorageID != "local-internal" {
		t.Fatalf("storage id = %q", entry.StorageID)
	}
	if _, err = router.resolve("local-internal", filepath.Join(root, "x"), false); err == nil {
		t.Fatalf("absolute caller path should be rejected")
	}
	// A symlink pointing outside the configured root must be refused.
	outside := t.TempDir()
	if err = os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err = router.resolve("local-internal", "link/file", true); err == nil {
		t.Fatalf("symlink escape should be rejected")
	}
}

func TestProviderConfigValidation(t *testing.T) {
	dir := t.TempDir()
	ok := ProviderConfig{Storages: []StorageEntry{{ID: "a", Kind: "usb", Root: dir}}}
	if err := ok.validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	write := func(body string) string {
		path := filepath.Join(t.TempDir(), "providers.json")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	cases := map[string]string{
		"root /":           `{"storages":[{"id":"a","kind":"usb","root":"/"}]}`,
		"storage path":     `{"storages":[{"id":"a","kind":"usb","root":"/storage/data"}]}`,
		"nix path":         `{"storages":[{"id":"a","kind":"usb","root":"/nix/store"}]}`,
		"relative root":    `{"storages":[{"id":"a","kind":"usb","root":"relative/dir"}]}`,
		"duplicate ids":    `{"storages":[{"id":"a","kind":"usb","root":"` + dir + `"},{"id":"a","kind":"usb","root":"` + dir + `"}]}`,
		"unsupported kind": `{"storages":[{"id":"a","kind":"smb","root":"` + dir + `"}]}`,
	}
	for name, body := range cases {
		if _, err := loadProviderConfig(write(body)); err == nil {
			t.Errorf("%s: config should be rejected", name)
		}
	}
}

func TestAPIReportsNotImplementedForStubProviders(t *testing.T) {
	router, _, _ := newTestRouter(t)
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := &Server{p: router, db: db, be: newBackend(router, db)}

	cases := []struct{ method, path string }{
		{http.MethodPost, "/api/storage/smb/mount"},
		{http.MethodGet, "/api/files?storage_id=webdav"},
		{http.MethodGet, "/api/storage/cloud/usage"},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		server.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != http.StatusNotImplemented {
			t.Errorf("%s %s => %d %s", tc.method, tc.path, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "not_implemented") {
			t.Errorf("%s %s: missing not_implemented code: %s", tc.method, tc.path, w.Body.String())
		}
	}
}

func TestStorageResponsesHideConfiguredRoots(t *testing.T) {
	router, root, _ := newTestRouter(t)
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := &Server{p: router, db: db, be: newBackend(router, db)}
	for _, path := range []string{"/api/storage", "/api/storage/local-internal", "/api/storage/local-internal/usage"} {
		w := httptest.NewRecorder()
		server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if strings.Contains(w.Body.String(), root) {
			t.Errorf("%s leaked configured root: %s", path, w.Body.String())
		}
	}
}
