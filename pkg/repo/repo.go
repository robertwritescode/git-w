package repo

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/robertwritescode/git-w/pkg/config"
)

// Repo is a resolved, ready-to-use repository with an absolute path.
type Repo struct {
	Name    string
	AbsPath string
	Flags   []string
}

// FromConfig returns a slice of Repos resolved from cfg, using cfgPath as the
// base for relative path resolution. Repos are returned in sorted name order.
func FromConfig(cfg *config.WorkspaceConfig, cfgPath string) []Repo {
	repos := make([]Repo, 0, len(cfg.Repos))

	for _, name := range config.SortedStringKeys(cfg.Repos) {
		if r, ok := resolveRepo(name, cfg.Repos[name], cfgPath); ok {
			repos = append(repos, r)
		}
	}

	return repos
}

// FromNames returns Repos for the given names, looked up in cfg.
// Names not found in cfg.Repos are silently skipped.
// Results are in sorted name order.
func FromNames(cfg *config.WorkspaceConfig, cfgPath string, names []string) []Repo {
	sorted := make([]string, len(names))
	copy(sorted, names)
	slices.Sort(sorted)

	repos := make([]Repo, 0, len(sorted))
	for _, name := range sorted {
		rc, ok := cfg.Repos[name]
		if !ok {
			continue
		}

		if r, ok := resolveRepo(name, rc, cfgPath); ok {
			repos = append(repos, r)
		}
	}
	return repos
}

// resolveRepo converts a config entry into a Repo with an absolute path.
// Returns false if the path cannot be resolved.
func resolveRepo(name string, rc config.RepoConfig, cfgPath string) (Repo, bool) {
	absPath, err := config.ResolveRepoPath(cfgPath, rc.Path)
	if err != nil {
		return Repo{}, false
	}

	return Repo{Name: name, AbsPath: absPath, Flags: rc.Flags}, true
}

// IsGitRepo reports whether path is a git repository (contains a .git entry).
func IsGitRepo(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}
