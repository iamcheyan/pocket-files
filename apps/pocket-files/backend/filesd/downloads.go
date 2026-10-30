package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
)

// downloadsClient reads download state from the PJOS `downloadsd` service. Pocket
// Files does not run its own download engine: when downloadsd is not configured
// or not reachable, the API answers an explicit unavailable state and the UI
// shows it rather than inventing entries.
type downloadsClient struct {
	baseURL string
	client  *http.Client
}

func newDownloadsClient() *downloadsClient {
	return &downloadsClient{
		baseURL: os.Getenv("POCKET_FILES_DOWNLOADSD"),
		client:  &http.Client{Timeout: 2 * time.Second},
	}
}

type downloadsResult struct {
	Status  string           `json:"status"`
	Source  string           `json:"source"`
	Reason  string           `json:"reason,omitempty"`
	Message string           `json:"message,omitempty"`
	Items   []map[string]any `json:"items,omitempty"`
}

func (d *downloadsClient) list() downloadsResult {
	result := downloadsResult{Source: "downloadsd"}
	if d.baseURL == "" {
		result.Status = "unavailable"
		result.Reason = "downloadsd_not_configured"
		result.Message = "No downloadsd endpoint is configured for this host."
		return result
	}
	resp, err := d.client.Get(d.baseURL + "/downloads")
	if err != nil {
		result.Status = "unavailable"
		result.Reason = "downloadsd_unreachable"
		result.Message = "The downloadsd service is not reachable."
		return result
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		result.Status = "unavailable"
		result.Reason = "downloadsd_error"
		result.Message = "downloadsd returned an error."
		return result
	}
	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if json.NewDecoder(resp.Body).Decode(&payload) != nil {
		result.Status = "unavailable"
		result.Reason = "downloadsd_invalid_response"
		result.Message = "downloadsd returned an unexpected payload."
		return result
	}
	result.Status = "ok"
	result.Items = payload.Items
	if result.Items == nil {
		result.Items = []map[string]any{}
	}
	return result
}

// handleDownloads serves GET /api/downloads from the downloadsd client.
func (s *Server) handleDownloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.error(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	if s.downloads == nil {
		s.downloads = newDownloadsClient()
	}
	result := s.downloads.list()
	status := http.StatusOK
	if result.Status != "ok" {
		// 503 keeps "service unavailable" distinct from a validation error.
		status = http.StatusServiceUnavailable
	}
	s.json(w, status, result)
}
