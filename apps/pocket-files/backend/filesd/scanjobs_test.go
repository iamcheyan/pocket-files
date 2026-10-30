package main

import (
	"net/http"
	"testing"
	"time"
)

func TestAnalyticsIsSynchronousForSmallTrees(t *testing.T) {
	s, p, _ := newTestServer(t)
	writeFile(t, p.roots["internal"], "Documents/small.txt", []byte("hi"))
	code, body := doJSON(t, s, http.MethodGet, "/api/analytics?storage_id=internal", nil)
	if code != 200 {
		t.Fatalf("small tree should answer synchronously, got %d", code)
	}
	if body["backgrounded"] != false {
		t.Fatalf("backgrounded = %v", body["backgrounded"])
	}
	if counts, ok := body["category_counts"].(map[string]any); !ok || counts["Documents"] == nil {
		t.Fatalf("category counts missing: %v", body["category_counts"])
	}
}

func TestAnalyticsBackgroundsLargeTrees(t *testing.T) {
	s, p, _ := newTestServer(t)
	// Create more entries than the async threshold in one flat directory.
	for i := 0; i < asyncScanThreshold+50; i++ {
		name := "Documents/bulk-" + itoa(i) + ".txt"
		writeFile(t, p.roots["internal"], name, []byte("x"))
	}
	code, body := doJSON(t, s, http.MethodGet, "/api/analytics?storage_id=internal", nil)
	if code != http.StatusAccepted && code != http.StatusOK {
		t.Fatalf("unexpected status %d", code)
	}
	if code == http.StatusAccepted {
		if body["backgrounded"] != true {
			t.Fatalf("202 response must report backgrounded: %v", body)
		}
		if _, ok := body["progress"].(float64); !ok {
			t.Fatalf("progress missing: %v", body)
		}
	}
	// Poll until the scan finishes and the cached result is complete.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		code, body = doJSON(t, s, http.MethodGet, "/api/analytics?storage_id=internal", nil)
		if code == 200 && body["backgrounded"] == false {
			counts, _ := body["category_counts"].(map[string]any)
			if counts == nil || counts["Documents"] == nil {
				t.Fatalf("completed scan lacks counts: %v", body)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("background scan did not complete")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
