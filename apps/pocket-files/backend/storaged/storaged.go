// Package storaged owns device discovery and storage operations for Pocket
// Files. filesd never touches a filesystem directly: it asks this package for
// every read, write and capacity figure, and the package enforces the root
// sandbox on each request.
package storaged

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileID joins a storage id and a provider-relative path into one opaque id.
func FileID(storageID, rel string) string { return storageID + ":" + rel }

// SplitID parses an opaque file id back into storage id and relative path.
func SplitID(id string) (storageID, rel string, err error) {
	storageID, rel, found := strings.Cut(id, ":")
	if !found || storageID == "" {
		return "", "", fmt.Errorf("file id must be \"storage:path\"")
	}
	return storageID, strings.TrimPrefix(rel, "/"), nil
}

// DirectoryUsage sums regular-file bytes under root. Mock volumes use it so
// their reported usage is their own generated data, never the host disk.
func DirectoryUsage(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if info, statErr := entry.Info(); statErr == nil && info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// Backend is what filesd consumes. Router is the production implementation.
type Backend interface {
	Provider
	AllDevices() []Device
	Resolve(storageID, rel string, allowMissing bool) (string, error)
	ProviderFor(id string) (Provider, error)
	MockVolumeCount() int
	ConfiguredVolumeCount() int
}

var _ Backend = (*Router)(nil)

// MockRoot exposes a mock volume's generated directory. It exists for tests that
// assert real filesystem effects; it is never serialized by the API.
func (p *MockProvider) MockRoot(id string) string { return p.roots[id] }
