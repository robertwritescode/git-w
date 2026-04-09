package config

import (
	"fmt"
	"os"

	"github.com/robertwritescode/git-w/pkg/toml"
)

// LoadStream reads a .gitw-stream manifest from path, applies defaults,
// validates uniqueness constraints, and returns the parsed manifest.
// Returns os.ErrNotExist unwrapped if the file is missing (callers use errors.Is).
func LoadStream(path string) (*WorkstreamManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m WorkstreamManifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing stream %s: %w", path, err)
	}

	applyStreamDefaults(&m)

	if err := validateStream(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

// applyStreamDefaults sets name and path defaults on each WorktreeEntry:
// 1. Count repo occurrences across all entries.
// 2. For single-occurrence repos: if name is empty, default name = repo.
// 3. For all entries: if path is empty, default path = name.
func applyStreamDefaults(m *WorkstreamManifest) {
	repoCounts := make(map[string]int, len(m.Worktrees))
	for _, e := range m.Worktrees {
		repoCounts[e.Repo]++
	}

	for i := range m.Worktrees {
		e := &m.Worktrees[i]
		if repoCounts[e.Repo] == 1 && e.Name == "" {
			e.Name = e.Repo
		}
		if e.Path == "" {
			e.Path = e.Name
		}
	}
}

// validateStream checks uniqueness of name and path, and that multi-occurrence
// repos all have explicit names. Called after applyStreamDefaults.
func validateStream(m *WorkstreamManifest) error {
	repoCounts := make(map[string]int, len(m.Worktrees))
	for _, e := range m.Worktrees {
		repoCounts[e.Repo]++
	}

	for _, e := range m.Worktrees {
		if repoCounts[e.Repo] > 1 && e.Name == "" {
			return fmt.Errorf("worktree entry for repo %q requires a name when the repo appears multiple times", e.Repo)
		}
	}

	seenNames := make(map[string]bool, len(m.Worktrees))
	for _, e := range m.Worktrees {
		if e.Name != "" {
			if seenNames[e.Name] {
				return fmt.Errorf("worktree name %q appears more than once in workstream", e.Name)
			}
			seenNames[e.Name] = true
		}
	}

	seenPaths := make(map[string]bool, len(m.Worktrees))
	for _, e := range m.Worktrees {
		if e.Path != "" {
			if seenPaths[e.Path] {
				return fmt.Errorf("worktree path %q appears more than once in workstream", e.Path)
			}
			seenPaths[e.Path] = true
		}
	}

	return nil
}
