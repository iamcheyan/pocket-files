package main

import (
	"github.com/iamcheyan/pocket-files/apps/pocket-files/backend/storaged"
)

// Storage implementation lives in the storaged package. filesd keeps local
// aliases so handlers read naturally and the two daemons can later be split into
// separate processes without touching the HTTP layer.
type (
	Device          = storaged.Device
	Entry           = storaged.Entry
	Provider        = storaged.Provider
	MockProvider    = storaged.MockProvider
	ProviderConfig  = storaged.Config
	StorageEntry    = storaged.StorageEntry
	RemovableDevice = storaged.RemovableDevice
	detectedDTO     = storaged.DetectedDTO
	storageBackend  = storaged.Backend
)

var ErrNotImplemented = storaged.ErrNotImplemented

func newMockProvider() (*storaged.MockProvider, error) { return storaged.NewMockProvider() }

func newProviderRouter(mock *storaged.MockProvider, cfg ProviderConfig) (*storaged.Router, error) {
	return storaged.NewRouter(mock, cfg)
}

func loadProviderConfig(path string) (ProviderConfig, error) { return storaged.LoadConfig(path) }

func resolveLocalPath(root, rel string, allowMissing bool) (string, error) {
	return storaged.ResolvePath(root, rel, allowMissing)
}

func statfsNumbers(path string) (total, avail, free int64, err error) {
	return storaged.StatfsNumbers(path)
}

func directoryUsage(root string) int64 { return storaged.DirectoryUsage(root) }

func discoverRemovableStorage(env storaged.DiscoveryEnv) []RemovableDevice {
	return storaged.DiscoverRemovableStorage(env)
}

func defaultDiscoveryEnv() storaged.DiscoveryEnv { return storaged.DefaultDiscoveryEnv() }

func detectedItems(devices []RemovableDevice) []detectedDTO { return storaged.DetectedItems(devices) }

func isNotImplemented(err error) bool {
	return err != nil && err.Error() == ErrNotImplemented.Error()
}
