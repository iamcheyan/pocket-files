package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type opResult struct {
	Status      string `json:"status"`
	Action      string `json:"action"`
	StorageID   string `json:"storage_id"`
	Source      string `json:"source"`
	Destination string `json:"destination,omitempty"`
	TrashPath   string `json:"trash_path,omitempty"`
	Error       string `json:"error,omitempty"`
}

func (s *Server) handleOperations(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 2 && r.Method == "POST" {
		s.handleOperation(w, r)
		return
	}
	if len(parts) == 3 {
		switch parts[2] {
		case "bulk":
			if r.Method == "POST" {
				s.handleBulkOperation(w, r)
				return
			}
		case "copy", "move", "delete", "undo":
			if r.Method == "POST" {
				s.handleOperationKind(w, r, parts[2])
				return
			}
		}
	}
	s.error(w, 404, "not_found", "endpoint not found")
}

// handleOperationKind serves the per-action contract paths: the action comes from
// the route, so clients do not have to repeat it in the request body.
func (s *Server) handleOperationKind(w http.ResponseWriter, r *http.Request, action string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		s.error(w, 400, "invalid_json", "invalid request")
		return
	}
	var payload map[string]any
	if len(bytes.TrimSpace(body)) > 0 {
		if json.Unmarshal(body, &payload) != nil {
			s.error(w, 400, "invalid_json", "invalid request")
			return
		}
	}
	payload["action"] = action
	r.Body = io.NopCloser(bytes.NewReader(mustJSON(payload)))
	s.handleOperation(w, r)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func (s *Server) handleOperation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action      string `json:"action"`
		StorageID   string `json:"storage_id"`
		Source      string `json:"source"`
		Destination string `json:"destination"`
		TrashPath   string `json:"trash_path"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		s.error(w, 400, "invalid_json", "invalid request")
		return
	}
	if req.Action == "" && strings.HasSuffix(strings.Trim(r.URL.Path, "/"), "/undo") {
		req.Action = "undo"
	}
	req.Action = strings.ToLower(req.Action)
	switch req.Action {
	case "copy":
		if req.StorageID == "" || req.Source == "" || req.Destination == "" {
			s.error(w, 400, "invalid_request", "storage_id, source and destination are required")
			return
		}
		if e := s.be.copyFile(req.StorageID, req.Source, req.Destination); e != nil {
			s.storageError(w, "operation_failed", e)
			return
		}
		s.json(w, 200, opResult{Status: "ok", Action: "copy", StorageID: req.StorageID, Source: req.Source, Destination: req.Destination})
	case "move":
		if req.StorageID == "" || req.Source == "" || req.Destination == "" {
			s.error(w, 400, "invalid_request", "storage_id, source and destination are required")
			return
		}
		if e := s.be.moveFile(req.StorageID, req.Source, req.Destination); e != nil {
			s.storageError(w, "operation_failed", e)
			return
		}
		s.json(w, 200, opResult{Status: "ok", Action: "move", StorageID: req.StorageID, Source: req.Source, Destination: req.Destination})
	case "delete":
		if req.StorageID == "" || req.Source == "" {
			s.error(w, 400, "invalid_request", "storage_id and source are required")
			return
		}
		tp, e := s.be.trashEntry(req.StorageID, req.Source)
		if e != nil {
			s.storageError(w, "operation_failed", e)
			return
		}
		s.json(w, 200, opResult{Status: "ok", Action: "delete", StorageID: req.StorageID, Source: req.Source, TrashPath: tp})
	case "undo":
		if req.StorageID == "" || req.Source == "" || req.TrashPath == "" {
			s.error(w, 400, "invalid_request", "storage_id, source and trash_path are required")
			return
		}
		if e := s.be.restoreEntry(req.StorageID, req.Source, req.TrashPath); e != nil {
			s.storageError(w, "operation_failed", e)
			return
		}
		s.json(w, 200, opResult{Status: "ok", Action: "undo", StorageID: req.StorageID, Source: req.Source})
	default:
		s.error(w, 400, "invalid_action", "action must be copy, move, delete or undo")
	}
}

func (s *Server) handleBulkOperation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action             string   `json:"action"`
		StorageID          string   `json:"storage_id"`
		Sources            []string `json:"sources"`
		Destination        string   `json:"destination"`
		DestinationStorage string   `json:"destination_storage"`
		Confirm            bool     `json:"confirm"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		s.error(w, 400, "invalid_json", "invalid request")
		return
	}
	req.Action = strings.ToLower(req.Action)
	if req.DestinationStorage != "" && req.DestinationStorage == req.StorageID {
		req.DestinationStorage = ""
	}
	if req.StorageID == "" {
		s.error(w, 400, "invalid_request", "storage_id is required")
		return
	}
	if len(req.Sources) == 0 {
		s.error(w, 400, "invalid_request", "at least one source is required")
		return
	}
	if req.Action == "delete" && !req.Confirm {
		s.error(w, 400, "confirm_required", "bulk delete requires confirm=true")
		return
	}
	results := make([]opResult, 0, len(req.Sources))
	var firstErr string
	for _, src := range req.Sources {
		var res opResult
		switch req.Action {
		case "copy":
			if req.Destination == "" {
				firstErr = "destination required"
				s.error(w, 400, "invalid_request", "destination is required for bulk copy")
				return
			}
			dir, base := resolveRelativeParent(req.Destination)
			dst := strings.TrimSuffix(dir, "/") + "/" + base + "_" + srcName(src)
			if req.DestinationStorage != "" {
				// Cross-device copies run through the persistent transfer queue so
				// progress, pause and crash recovery behave like any other job.
				id, idErr := randID()
				if idErr != nil {
					s.error(w, 400, "operation_failed", "unable to queue transfer")
					return
				}
				jobErr := s.be.tc.create(id, "copy", req.StorageID, conflictKeepBoth,
					[]transferItem{{Src: src, Dst: dst, DstStorage: req.DestinationStorage}}, 0)
				if jobErr != nil {
					res = opResult{Status: "error", Action: "copy", StorageID: req.StorageID, Source: src, Error: jobErr.Error()}
					break
				}
				if job, jobLookup := s.be.tc.get(id); jobLookup == nil {
					go s.be.tc.start(job)
				}
				res = opResult{Status: "queued", Action: "copy", StorageID: req.StorageID, Source: src, Destination: dst}
				break
			}
			if e := s.be.copyFile(req.StorageID, src, dst); e != nil {
				res = opResult{Status: "error", Action: "copy", StorageID: req.StorageID, Source: src, Error: e.Error()}
			} else {
				res = opResult{Status: "ok", Action: "copy", StorageID: req.StorageID, Source: src, Destination: dst}
			}
		case "move":
			if req.Destination == "" {
				s.error(w, 400, "invalid_request", "destination is required for bulk move")
				return
			}
			dir, base := resolveRelativeParent(req.Destination)
			dst := strings.TrimSuffix(dir, "/") + "/" + base + "_" + srcName(src)
			if e := s.be.moveFile(req.StorageID, src, dst); e != nil {
				res = opResult{Status: "error", Action: "move", StorageID: req.StorageID, Source: src, Error: e.Error()}
			} else {
				res = opResult{Status: "ok", Action: "move", StorageID: req.StorageID, Source: src, Destination: dst}
			}
		case "delete":
			tp, e := s.be.trashEntry(req.StorageID, src)
			if e != nil {
				res = opResult{Status: "error", Action: "delete", StorageID: req.StorageID, Source: src, Error: e.Error()}
			} else {
				res = opResult{Status: "ok", Action: "delete", StorageID: req.StorageID, Source: src, TrashPath: tp}
			}
		case "undo":
			s.error(w, 400, "invalid_action", "bulk undo is not supported")
			return
		default:
			s.error(w, 400, "invalid_action", "action must be copy, move or delete")
			return
		}
		results = append(results, res)
		if res.Status == "error" && firstErr == "" {
			firstErr = res.Error
		}
	}
	s.json(w, 200, map[string]any{"status": "ok", "action": req.Action, "storage_id": req.StorageID, "results": results})
}

func srcName(rel string) string {
	rel = strings.Trim(rel, "/")
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[i+1:]
	}
	return rel
}

func (s *Server) handleFavorites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		sid := r.URL.Query().Get("storage_id")
		s.json(w, 200, map[string]any{"items": s.be.listFavorites(sid)})
	case "POST":
		var v struct {
			StorageID string `json:"storage_id"`
			URI       string `json:"uri"`
		}
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			s.error(w, 400, "invalid_json", "invalid request")
			return
		}
		if e := s.be.addFavorite(v.StorageID, v.URI); e != nil {
			s.error(w, 400, "invalid_request", e.Error())
			return
		}
		s.json(w, 200, map[string]any{"status": "ok", "storage_id": v.StorageID, "uri": v.URI})
	case "DELETE":
		var v struct {
			StorageID string `json:"storage_id"`
			URI       string `json:"uri"`
		}
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			s.error(w, 400, "invalid_json", "invalid request")
			return
		}
		if e := s.be.removeFavorite(v.StorageID, v.URI); e != nil {
			s.error(w, 400, "invalid_request", e.Error())
			return
		}
		s.json(w, 200, map[string]any{"status": "ok", "storage_id": v.StorageID, "uri": v.URI})
	default:
		s.error(w, 405, "method_not_allowed", "use GET, POST or DELETE")
	}
}

func (s *Server) handleTags(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		sid := r.URL.Query().Get("storage_id")
		uri := r.URL.Query().Get("uri")
		s.json(w, 200, map[string]any{"items": s.be.listTags(sid, uri)})
	case "POST":
		var v struct {
			StorageID string `json:"storage_id"`
			URI       string `json:"uri"`
			Tag       string `json:"tag"`
		}
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			s.error(w, 400, "invalid_json", "invalid request")
			return
		}
		if e := s.be.upsertTag(v.StorageID, v.URI, v.Tag); e != nil {
			s.error(w, 400, "invalid_request", e.Error())
			return
		}
		s.json(w, 200, map[string]any{"status": "ok", "storage_id": v.StorageID, "uri": v.URI, "tag": v.Tag})
	case "PUT":
		var v struct {
			StorageID string `json:"storage_id"`
			URI       string `json:"uri"`
			Tag       string `json:"tag"`
		}
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			s.error(w, 400, "invalid_json", "invalid request")
			return
		}
		if e := s.be.upsertTag(v.StorageID, v.URI, v.Tag); e != nil {
			s.error(w, 400, "invalid_request", e.Error())
			return
		}
		s.json(w, 200, map[string]any{"status": "ok", "storage_id": v.StorageID, "uri": v.URI, "tag": v.Tag})
	case "DELETE":
		var v struct {
			StorageID string `json:"storage_id"`
			URI       string `json:"uri"`
			Tag       string `json:"tag"`
		}
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			s.error(w, 400, "invalid_json", "invalid request")
			return
		}
		if e := s.be.deleteTag(v.StorageID, v.URI, v.Tag); e != nil {
			s.error(w, 400, "invalid_request", e.Error())
			return
		}
		s.json(w, 200, map[string]any{"status": "ok", "storage_id": v.StorageID, "uri": v.URI, "tag": v.Tag})
	default:
		s.error(w, 405, "method_not_allowed", "use GET, POST, PUT or DELETE")
	}
}

func (s *Server) handleRecents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		kind := r.URL.Query().Get("kind")
		s.json(w, 200, map[string]any{"items": s.be.listRecents(kind, 0)})
	case "DELETE":
		if e := s.be.clearRecents(); e != nil {
			s.storageError(w, "operation_failed", e)
			return
		}
		s.json(w, 200, map[string]any{"status": "ok"})
	default:
		s.error(w, 405, "method_not_allowed", "use GET or DELETE")
	}
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.json(w, 200, s.be.getSettings())
	case "PUT":
		raw, err := readAll(r)
		if err != nil {
			s.error(w, 400, "invalid_json", "invalid request")
			return
		}
		var m map[string]json.RawMessage
		if e := json.Unmarshal(raw, &m); e != nil {
			s.error(w, 400, "invalid_json", e.Error())
			return
		}
		for k, v := range m {
			if k == "mode" || k == "privacy_mode" {
				continue
			}
			if e := s.be.putSetting(k, string(v)); e != nil {
				s.storageError(w, "operation_failed", e)
				return
			}
		}
		s.json(w, 200, s.be.getSettings())
	default:
		s.error(w, 405, "method_not_allowed", "use GET or PUT")
	}
}

// handleFileRename serves PUT /api/files/{id}/rename with {"name":"new-name"}.
// It renames in place, keeping the entry inside the same sandboxed parent.
func (s *Server) handleFileRename(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPut {
		s.error(w, 405, "method_not_allowed", "use PUT")
		return
	}
	sid, rel, err := s.be.splitID(id)
	if err != nil {
		s.error(w, 400, "invalid_id", err.Error())
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil || strings.TrimSpace(body.Name) == "" {
		s.error(w, 400, "invalid_request", "name is required")
		return
	}
	name := strings.TrimSpace(body.Name)
	if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
		s.error(w, 400, "invalid_name", "name must be a single path segment")
		return
	}
	parent := ""
	if idx := strings.LastIndex(rel, "/"); idx >= 0 {
		parent = rel[:idx]
	}
	destination := name
	if parent != "" {
		destination = parent + "/" + name
	}
	if err = s.be.moveFile(sid, rel, destination); err != nil {
		s.storageError(w, "operation_failed", err)
		return
	}
	s.json(w, 200, map[string]any{
		"status": "ok", "storage_id": sid, "source": rel, "destination": destination, "name": name,
	})
}

func (s *Server) handleFileMetadata(w http.ResponseWriter, r *http.Request, id string) {
	sid, rel, err := s.be.splitID(id)
	if err != nil {
		s.error(w, 400, "invalid_id", err.Error())
		return
	}
	e, err := s.be.getEntry(sid, rel)
	if err != nil {
		s.error(w, 404, "not_found", err.Error())
		return
	}
	s.be.recordRecentForEntry(e)
	s.json(w, 200, entryItem(e))
}

func (s *Server) handleFilePreview(w http.ResponseWriter, r *http.Request, id string) {
	sid, rel, err := s.be.splitID(id)
	if err != nil {
		s.error(w, 400, "invalid_id", err.Error())
		return
	}
	e, err := s.be.getEntry(sid, rel)
	if err != nil {
		s.error(w, 404, "not_found", err.Error())
		return
	}
	if e.Kind == "directory" {
		s.error(w, 400, "not_a_file", "cannot preview a directory")
		return
	}
	ct, ok := previewContentType(rel, e.Size)
	if !ok {
		s.error(w, 400, "unsupported_type", "preview not supported for this file type")
		return
	}
	full, err := s.p.Resolve(sid, rel, false)
	if err != nil {
		s.error(w, 400, "resolve_error", "unable to read file")
		return
	}
	f, err := openPreviewFile(full)
	if err != nil {
		s.error(w, 400, "read_error", "unable to read file")
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", ct)
	http.ServeContent(w, r, rel, e.Modified, f)
}
func previewContentType(rel string, size int64) (string, bool) {
	if size > 8<<20 {
		return "", false
	}
	lower := strings.ToLower(rel)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png", true
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg", true
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif", true
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp", true
	case strings.HasSuffix(lower, ".bmp"):
		return "image/bmp", true
	case strings.HasSuffix(lower, ".txt"), strings.HasSuffix(lower, ".csv"), strings.HasSuffix(lower, ".log"):
		return "text/plain; charset=utf-8", true
	case strings.HasSuffix(lower, ".md"), strings.HasSuffix(lower, ".markdown"):
		return "text/markdown; charset=utf-8", true
	case strings.HasSuffix(lower, ".json"):
		return "application/json", true
	case strings.HasSuffix(lower, ".yaml"), strings.HasSuffix(lower, ".yml"):
		return "text/yaml; charset=utf-8", true
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf", true
	}
	return "", false
}
