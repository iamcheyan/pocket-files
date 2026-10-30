package main

import (
	"io/fs"
	"path/filepath"
)

// directoryUsage sums file sizes under root. It is used for mock volumes, whose
// declared capacity is nominal and must not be confused with the host disk.
func directoryUsage(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, entry fs.DirEntry, err error) error {
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
