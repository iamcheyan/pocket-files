package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// appRegistry models the PJOS AppLaunchAPI. Pocket Files never launches a host
// binary itself: it asks the registry, which reports whether the target PJOS app
// is installed. When the registry is absent every request resolves to an
// explicit `unavailable` result so the UI can show that state instead of
// silently doing nothing.
type appRegistry struct {
	entrypoint string
	client     *http.Client
}

type appLaunchResult struct {
	Status  string `json:"status"`
	App     string `json:"app,omitempty"`
	Target  string `json:"target,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

// appForExtension maps a file family to the Pocket app that should open it.
func appForExtension(ext string) (string, bool) {
	switch strings.ToLower(ext) {
	case "epub", "pdf", "mobi", "azw", "azw3", "cbz", "cbr", "txt", "md":
		return "pocket-reader", true
	case "mp3", "flac", "wav", "ogg", "m4a", "aac", "opus":
		return "pocket-music", true
	case "mp4", "mkv", "webm", "avi", "mov", "m4v":
		return "pocket-video", true
	case "gba", "gb", "nes", "sfc", "iso", "rom", "n64", "z64":
		return "pocket-gamehub", true
	case "html", "htm":
		return "pocket-web", true
	}
	return "", false
}

func newAppRegistry() *appRegistry {
	return &appRegistry{
		entrypoint: os.Getenv("POCKET_FILES_APP_REGISTRY"),
		client:     &http.Client{Timeout: 2 * time.Second},
	}
}

// launch asks the configured registry to open the target. Any transport failure,
// missing configuration or unexpected status resolves to unavailable.
func (r *appRegistry) launch(storageID, rel, app string) appLaunchResult {
	result := appLaunchResult{App: app, Target: storageID + ":" + rel}
	if r.entrypoint == "" {
		result.Status = "unavailable"
		result.Reason = "app_registry_not_configured"
		result.Message = "No PJAppRegistry endpoint is configured for this host."
		return result
	}
	body, err := json.Marshal(map[string]string{
		"app": app, "storage_id": storageID, "path": rel,
	})
	if err != nil {
		result.Status = "unavailable"
		result.Reason = "encode_failed"
		return result
	}
	req, err := http.NewRequest(http.MethodPost, r.entrypoint+"/launch", strings.NewReader(string(body)))
	if err != nil {
		result.Status = "unavailable"
		result.Reason = "invalid_registry_endpoint"
		return result
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		result.Status = "unavailable"
		result.Reason = "app_registry_unreachable"
		result.Message = "The PJAppRegistry service is not reachable."
		return result
	}
	defer resp.Body.Close()
	var launched appLaunchResult
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&launched) != nil {
		result.Status = "unavailable"
		result.Reason = "app_not_installed"
		result.Message = "The requested app is not installed on this device."
		return result
	}
	if launched.Status == "" {
		launched.Status = "ok"
	}
	launched.App = app
	launched.Target = storageID + ":" + rel
	return launched
}

// openEntryHandler implements GET /api/apps/open?storage_id=&path= or ?id=.
func (s *Server) openEntryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.error(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	sid := r.URL.Query().Get("storage_id")
	rel := r.URL.Query().Get("path")
	if id := r.URL.Query().Get("id"); id != "" {
		parsedStorage, parsedPath, err := s.be.splitID(id)
		if err != nil {
			s.error(w, http.StatusBadRequest, "invalid_id", err.Error())
			return
		}
		sid, rel = parsedStorage, parsedPath
	}
	if sid == "" || rel == "" {
		s.error(w, http.StatusBadRequest, "invalid_request", "storage_id and path are required")
		return
	}
	entry, err := s.be.getEntry(sid, rel)
	if err != nil {
		s.error(w, http.StatusNotFound, "not_found", "entry not found")
		return
	}
	app, known := appForExtension(entry.Extension)
	if !known {
		s.json(w, http.StatusOK, appLaunchResult{
			Status: "unavailable", Reason: "no_app_for_type",
			Message: "No Pocket app handles this file type.",
		})
		return
	}
	s.json(w, http.StatusOK, s.apps.launch(sid, rel, app))
}
