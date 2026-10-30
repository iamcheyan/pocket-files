package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProviderConfig is the on-disk operator configuration for real local volumes.
// It is read from POCKET_FILES_CONFIG; without it the service runs on mock
// storage only. Roots are validated once at startup and never accepted from an
// API caller.
type ProviderConfig struct {
	Storages []StorageEntry `json:"storages"`
}

type StorageEntry struct {
	ID           string `json:"id"`
	FriendlyName string `json:"friendly_name"`
	Kind         string `json:"kind"`
	Root         string `json:"root"`
	ReadOnly     bool   `json:"read_only"`
}

// forbiddenRoots are host paths that must never be mounted as an app-managed
// volume: system trees and the production data disk.
var forbiddenRoots = []string{"/", "/nix", "/nix/store", "/boot", "/home", "/storage", "/etc", "/usr", "/var"}

func loadProviderConfig(path string) (ProviderConfig, error) {
	if path == "" {
		return ProviderConfig{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ProviderConfig{}, fmt.Errorf("read provider config: %w", err)
	}
	var cfg ProviderConfig
	if err = json.Unmarshal(data, &cfg); err != nil {
		return ProviderConfig{}, fmt.Errorf("parse provider config: %w", err)
	}
	if err = cfg.validate(); err != nil {
		return ProviderConfig{}, err
	}
	return cfg, nil
}

func (c ProviderConfig) validate() error {
	seen := map[string]bool{}
	for _, entry := range c.Storages {
		if entry.ID == "" {
			return fmt.Errorf("storage entry requires an id")
		}
		if seen[entry.ID] {
			return fmt.Errorf("duplicate storage id %q", entry.ID)
		}
		seen[entry.ID] = true
		if entry.Root == "" || !filepath.IsAbs(entry.Root) {
			return fmt.Errorf("storage %q root must be an absolute path", entry.ID)
		}
		clean := filepath.Clean(entry.Root)
		for _, forbidden := range forbiddenRoots {
			if clean == forbidden || strings.HasPrefix(clean, forbidden+"/") {
				if forbidden == "/" && clean != "/" {
					continue
				}
				return fmt.Errorf("storage %q root %q is a protected system path", entry.ID, clean)
			}
		}
		info, err := os.Stat(clean)
		if err != nil {
			return fmt.Errorf("storage %q root unavailable: %w", entry.ID, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("storage %q root is not a directory", entry.ID)
		}
		switch entry.Kind {
		case "internal", "tf", "usb":
		default:
			return fmt.Errorf("storage %q has unsupported kind %q", entry.ID, entry.Kind)
		}
	}
	return nil
}
