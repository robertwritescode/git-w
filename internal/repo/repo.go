package repo

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/robertwritescode/git-workspace/internal/config"
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
	keys := sortedKeys(cfg.Repos)
	root := config.ConfigDir(cfgPath)
	repos := make([]Repo, 0, len(keys))

	for _, name := range keys {
		rc := cfg.Repos[name]
		repos = append(repos, Repo{
			Name:    name,
			AbsPath: absPath(root, rc.Path),
			Flags:   rc.Flags,
		})
	}

	return repos
}

// IsGitRepo reports whether path is a git repository (contains a .git entry).
func IsGitRepo(path string) bool {
	f, err := os.Open(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func sortedKeys(m map[string]config.RepoConfig) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func absPath(root, relPath string) string {
	return filepath.Join(root, relPath)
}
