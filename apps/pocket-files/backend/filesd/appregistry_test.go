package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAppForExtensionMapping(t *testing.T) {
	cases := map[string]string{
		"epub": "pocket-reader", "pdf": "pocket-reader", "md": "pocket-reader",
		"mp3": "pocket-music", "flac": "pocket-music",
		"mp4": "pocket-video", "mkv": "pocket-video",
		"gba": "pocket-gamehub", "nes": "pocket-gamehub",
		"html": "pocket-web",
	}
	for ext, want := range cases {
		got, ok := appForExtension(ext)
		if !ok || got != want {
			t.Errorf("appForExtension(%q) = %q,%v want %q", ext, got, ok, want)
		}
	}
	if _, ok := appForExtension("zip"); ok {
		t.Errorf("zip should have no associated app")
	}
}

func TestAppOpenWithoutRegistryIsExplicitlyUnavailable(t *testing.T) {
	s, p, _ := newTestServer(t)
	writeFile(t, p.MockRoot("internal"), "Books/book.epub", []byte("book"))
	s.apps = newAppRegistry()

	code, body := doJSON(t, s, http.MethodGet, "/api/apps/open?storage_id=internal&path=Books/book.epub", nil)
	if code != 200 {
		t.Fatalf("status %d", code)
	}
	if body["status"] != "unavailable" {
		t.Fatalf("expected unavailable, got %v", body)
	}
	if body["reason"] != "app_registry_not_configured" {
		t.Fatalf("unexpected reason: %v", body)
	}
	if body["app"] != "pocket-reader" {
		t.Fatalf("app mapping missing: %v", body)
	}
}

func TestAppOpenCallsConfiguredRegistry(t *testing.T) {
	var received string
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer registry.Close()

	s, p, _ := newTestServer(t)
	writeFile(t, p.MockRoot("internal"), "Videos/clip.mp4", []byte("video"))
	s.apps = &appRegistry{entrypoint: registry.URL, client: registry.Client()}

	code, body := doJSON(t, s, http.MethodGet, "/api/apps/open?storage_id=internal&path=Videos/clip.mp4", nil)
	if code != 200 || body["status"] != "ok" {
		t.Fatalf("registry launch failed: %d %v", code, body)
	}
	if received != "/launch" {
		t.Fatalf("registry path = %q", received)
	}
	if body["app"] != "pocket-video" {
		t.Fatalf("app = %v", body["app"])
	}
}

func TestAppOpenUnknownTypeAndMissingEntry(t *testing.T) {
	s, p, _ := newTestServer(t)
	writeFile(t, p.MockRoot("internal"), "Documents/a.zip", []byte("zip"))
	s.apps = newAppRegistry()
	code, body := doJSON(t, s, http.MethodGet, "/api/apps/open?storage_id=internal&path=Documents/a.zip", nil)
	if code != 200 || body["status"] != "unavailable" || body["reason"] != "no_app_for_type" {
		t.Fatalf("unexpected: %d %v", code, body)
	}
	code, _ = doJSON(t, s, http.MethodGet, "/api/apps/open?storage_id=internal&path=Documents/missing.txt", nil)
	if code != 404 {
		t.Fatalf("missing entry should 404, got %d", code)
	}
	code, _ = doJSON(t, s, http.MethodGet, "/api/apps/open?storage_id=internal", nil)
	if code != 400 {
		t.Fatalf("missing path should 400, got %d", code)
	}
}
