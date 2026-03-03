package config

import (
	"errors"
	"os"
	"path/filepath"
)

const ConfigFileName = ".gitw"

var ErrNotFound = errors.New("no .gitw found in current directory or any parent")

// Discover searches for .gitw starting from startDir and walking up to the
// filesystem root. If the GIT_W_CONFIG environment variable is set, it is
// returned directly without any filesystem check.
func Discover(startDir string) (string, error) {
	if override := os.Getenv("GIT_W_CONFIG"); override != "" {
		return override, nil
	}

	dir := startDir
	for {
		candidate := filepath.Join(dir, ConfigFileName)

		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}

		dir = parent
	}
}
