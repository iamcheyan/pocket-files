package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownloadsUnavailableWithoutService(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.downloads = newDownloadsClient()
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/downloads", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	body := decodeJSON(t, w)
	if body["status"] != "unavailable" || body["reason"] != "downloadsd_not_configured" {
		t.Fatalf("unexpected payload: %v", body)
	}
	if body["source"] != "downloadsd" {
		t.Fatalf("source must name downloadsd: %v", body)
	}
}

func TestDownloadsReadsFromService(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/downloads" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"id":"d1","name":"Clip.mp4","state":"downloading","progress":42}]}`))
	}))
	defer service.Close()

	s, _, _ := newTestServer(t)
	s.downloads = &downloadsClient{baseURL: service.URL, client: service.Client()}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/downloads", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", w.Code, w.Body.String())
	}
	body := decodeJSON(t, w)
	if body["status"] != "ok" {
		t.Fatalf("status payload: %v", body)
	}
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items: %v", body["items"])
	}
	first, _ := items[0].(map[string]any)
	if first["name"] != "Clip.mp4" {
		t.Fatalf("first item = %v", first)
	}
}

func TestDownloadsServiceFailureIsUnavailable(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer service.Close()
	s, _, _ := newTestServer(t)
	s.downloads = &downloadsClient{baseURL: service.URL, client: service.Client()}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/downloads", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if decodeJSON(t, w)["reason"] != "downloadsd_error" {
		t.Fatalf("unexpected payload: %s", w.Body.String())
	}
}

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v (%s)", err, w.Body.String())
	}
	return body
}
