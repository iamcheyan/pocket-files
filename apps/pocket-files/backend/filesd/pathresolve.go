package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveLocalPath resolves a relative path inside a configured absolute root,
// rejecting absolute paths, `..` traversal and symlink escapes. It mirrors the
// sandbox behaviour already used by MockProvider but is reusable by every
// local provider.
func resolveLocalPath(root, rel string, allowMissing bool) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute paths are forbidden")
	}
	clean := filepath.Clean(rel)
	if clean == "." {
		clean = ""
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes storage root")
	}
	base, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(base, clean)
	cur := candidate
	for {
		real, evalErr := filepath.EvalSymlinks(cur)
		if evalErr == nil {
			if real != base && !strings.HasPrefix(real, base+string(filepath.Separator)) {
				return "", fmt.Errorf("symlink escapes storage root")
			}
			break
		}
		if !allowMissing || !errors.Is(evalErr, os.ErrNotExist) {
			return "", evalErr
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", evalErr
		}
		cur = parent
	}
	// Every existing ancestor was already checked in the loop above, so the
	// candidate cannot escape through a symlinked parent.
	return candidate, nil
}
