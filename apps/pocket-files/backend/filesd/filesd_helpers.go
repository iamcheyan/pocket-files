package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// randID returns a stable-format random identifier for jobs and trash journals.
func randID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// handleDetectedStorage reports removable media discovered read-only. These
// devices are display-only: no operation route accepts them as a storage id.
func (s *Server) handleDetectedStorage(w http.ResponseWriter, _ *http.Request) {
	devices := discoverRemovableStorage(defaultDiscoveryEnv())
	s.json(w, http.StatusOK, map[string]any{
		"items": detectedItems(devices),
		"mode":  "read_only",
		"note":  "Detected removable media are reported for display only; mount and write operations are disabled.",
	})
}
