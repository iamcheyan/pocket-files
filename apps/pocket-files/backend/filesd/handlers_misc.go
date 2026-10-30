package main

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.error(w, 405, "method_not_allowed", "use POST")
		return
	}
	var body struct {
		StorageID string `json:"storage_id"`
		Path      string `json:"path"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		s.error(w, 400, "invalid_json", "invalid request")
		return
	}
	if err := s.p.CreateDir(r.Context(), body.StorageID, body.Path); err != nil {
		s.storageError(w, "operation_failed", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleStorageItem(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == http.MethodGet {
		d, ok := s.be.getDevice(id)
		if !ok {
			s.error(w, 404, "storage_not_found", "unknown storage")
			return
		}
		name := s.be.getFriendlyName(id)
		s.json(w, 200, map[string]any{"id": d.ID, "name": name, "friendly_name": name, "kind": d.Kind, "status": d.Status, "mock": d.Mock, "read_only": d.ReadOnly, "total": d.Total, "used": d.Used})
		return
	}
	if r.Method == http.MethodPut {
		var body struct {
			FriendlyName string `json:"friendly_name"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			s.error(w, 400, "invalid_json", "invalid request")
			return
		}
		if err := s.be.updateFriendlyName(id, body.FriendlyName); err != nil {
			s.storageError(w, "storage_error", err)
			return
		}
		s.json(w, 200, map[string]any{"id": id, "friendly_name": body.FriendlyName})
		return
	}
	s.error(w, 405, "method_not_allowed", "unsupported storage method")
}

func (s *Server) handleStorageUsage(w http.ResponseWriter, r *http.Request, id string) {
	if _, known := s.be.getDevice(id); !known {
		s.error(w, http.StatusNotFound, "storage_not_found", "unknown storage")
		return
	}
	total, used, free, err := s.be.storageUsage(id)
	if err != nil {
		s.storageError(w, "storage_error", err)
		return
	}
	d, _ := s.be.getDevice(id)
	s.json(w, 200, map[string]any{"id": id, "mock": d.Mock, "total": total, "used": used, "free": free})
}

func (s *Server) handleViews(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) != 3 || r.Method != http.MethodGet {
		s.error(w, 405, "method_not_allowed", "use GET")
		return
	}
	items, err := s.be.getView(r.Context(), parts[2])
	if err != nil {
		s.storageError(w, "view_error", err)
		return
	}
	s.json(w, 200, map[string]any{"view": parts[2], "items": items})
}

func (s *Server) handleApps(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) >= 3 && parts[2] == "open" {
		s.json(w, 200, map[string]any{"status": "unavailable", "reason": "pj_app_registry_not_installed"})
		return
	}
	s.json(w, 200, map[string]any{"status": "unavailable", "reason": "pj_app_registry_not_installed"})
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.error(w, 405, "method_not_allowed", "use GET")
		return
	}
	items := []map[string]any{
		{"id": "smb", "kind": "smb", "status": "not_implemented"},
		{"id": "webdav", "kind": "webdav", "status": "not_implemented"},
		{"id": "cloud", "kind": "cloud", "status": "not_implemented"},
	}
	s.json(w, 200, map[string]any{"providers": items})
}
