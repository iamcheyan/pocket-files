package main

import "net/http"

// detectedDTO is the API shape of a real, read-only discovered volume. It never
// carries a kernel device node or an absolute mount path.
type detectedDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	Mock       bool   `json:"mock"`
	ReadOnly   bool   `json:"read_only"`
	Detected   bool   `json:"detected"`
	Total      int64  `json:"total"`
	Used       int64  `json:"used"`
	Filesystem string `json:"filesystem"`
}

func detectedItems(devices []RemovableDevice) []detectedDTO {
	out := make([]detectedDTO, 0, len(devices))
	for _, d := range devices {
		used := d.Total - d.Free
		if used < 0 {
			used = 0
		}
		status := "ready"
		if d.ReadOnly {
			status = "read_only"
		}
		out = append(out, detectedDTO{
			ID: d.ID, Name: d.Name, Kind: d.Kind, Status: status,
			Mock: false, ReadOnly: true, Detected: true,
			Total: d.Total, Used: used, Filesystem: d.Filesystem,
		})
	}
	return out
}

// handleDetectedStorage reports real removable media discovered read-only.
// Mounting, unmounting and writing are deliberately not offered for these
// devices: the host's production volumes must never be mutated by this service.
func (s *Server) handleDetectedStorage(w http.ResponseWriter, _ *http.Request) {
	devices := discoverRemovableStorage(defaultDiscoveryEnv())
	s.json(w, 200, map[string]any{
		"items": detectedItems(devices),
		"mode":  "read_only",
		"note":  "Detected removable media are reported for display only; mount and write operations are disabled.",
	})
}
