package storaged

// DetectedDTO is the API shape of a discovered removable volume. It reports only
// a stable id, friendly kind label and capacity: kernel device names and mount
// points never leave the process.
type DetectedDTO struct {
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

// DetectedItems converts discovery results into API DTOs.
func DetectedItems(devices []RemovableDevice) []DetectedDTO {
	out := make([]DetectedDTO, 0, len(devices))
	for _, d := range devices {
		used := d.Total - d.Free
		if used < 0 {
			used = 0
		}
		status := "ready"
		if d.ReadOnly {
			status = "read_only"
		}
		out = append(out, DetectedDTO{
			ID: d.ID, Name: d.Name, Kind: d.Kind, Status: status,
			Mock: false, ReadOnly: true, Detected: true,
			Total: d.Total, Used: used, Filesystem: d.Filesystem,
		})
	}
	return out
}
