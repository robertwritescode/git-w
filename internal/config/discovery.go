package config

import (
	"errors"
	"os"
	"path/filepath"
)

const ConfigFileName = ".gitworkspace"

var ErrNotFound = errors.New("no .gitworkspace found in current directory or any parent")

// Discover searches for .gitworkspace starting from startDir and walking up to the
// filesystem root. If the GIT_WORKSPACE_CONFIG environment variable is set, it is
// returned directly without any filesystem check.
func Discover(startDir string) (string, error) {
	if override := os.Getenv("GIT_WORKSPACE_CONFIG"); override != "" {
		return override, nil
	}

	dir := startDir
	for {
		candidate := filepath.Join(dir, ConfigFileName)

		// Use Open+Close instead of Stat to avoid a redundant syscall on success.
		if f, err := os.Open(candidate); err == nil {
			f.Close()
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}

		dir = parent
	}
}
