package workspace

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
)

// WorkspaceConfig is the merged result of `.gitw` and `.gitw.local`.
// Repos and Groups maps are always non-nil after loading.
type WorkspaceConfig struct {
	Workspace WorkspaceMeta             `toml:"workspace"`
	Context   ContextConfig             `toml:"context"` // sourced from .gitw.local
	Repos     map[string]RepoConfig     `toml:"repos"`
	Groups    map[string]GroupConfig    `toml:"groups"`
	Worktrees map[string]WorktreeConfig `toml:"worktrees"`
}

// WorkspaceMeta holds top-level workspace settings.
type WorkspaceMeta struct {
	Name          string `toml:"name"`
	AutoGitignore *bool  `toml:"auto_gitignore"` // nil means true (default on)
}

// RepoConfig represents one tracked repository.
type RepoConfig struct {
	Path  string   `toml:"path"`
	URL   string   `toml:"url,omitempty"`
	Flags []string `toml:"flags,omitempty"`
}

// WorktreeConfig describes one shared bare-repo + branch worktree set.
type WorktreeConfig struct {
	URL      string            `toml:"url"`
	BarePath string            `toml:"bare_path"`
	Branches map[string]string `toml:"branches"`
}

// GroupConfig is a named set of repos.
type GroupConfig struct {
	Repos []string `toml:"repos"`
	Path  string   `toml:"path,omitempty"` // optional; used for auto-context detection
}

// ContextConfig holds the active context (stored in .gitw.local).
type ContextConfig struct {
	Active string `toml:"active"`
}

// AutoGitignoreEnabled reports whether auto-gitignore is on (nil means default true).
func (c WorkspaceConfig) AutoGitignoreEnabled() bool {
	return c.Workspace.AutoGitignore == nil || *c.Workspace.AutoGitignore
}

// AddRepoToGroup appends name to the named group, creating the group if absent.
// It is idempotent: if name is already in the group, it is not added again.
func (c *WorkspaceConfig) AddRepoToGroup(group, name string) {
	g := c.Groups[group]

	for _, r := range g.Repos {
		if r == name {
			return
		}
	}

	g.Repos = append(g.Repos, name)
	c.Groups[group] = g
}

// RepoName returns the base-name of absPath and errors if it is already registered.
func (c *WorkspaceConfig) RepoName(absPath string) (string, error) {
	name := filepath.Base(absPath)

	if _, exists := c.Repos[name]; exists {
		return "", fmt.Errorf("repo %q is already registered", name)
	}

	return name, nil
}

// WorktreeRepoName returns the synthesized repo name for a set+branch.
func WorktreeRepoName(setName, branch string) string {
	return fmt.Sprintf("%s-%s", setName, branch)
}

// RemoveRepoFromManualGroups removes repoName from every group that is not
// synthesized from a worktree set. Must be called before deleting the set
// from cfg.Worktrees so the synthesized group can still be identified.
func (c *WorkspaceConfig) RemoveRepoFromManualGroups(repoName string) {
	for groupName, g := range c.Groups {
		if _, isSynth := c.Worktrees[groupName]; isSynth {
			continue
		}

		filtered := make([]string, 0, len(g.Repos))
		for _, r := range g.Repos {
			if r != repoName {
				filtered = append(filtered, r)
			}
		}

		if len(filtered) != len(g.Repos) {
			g.Repos = filtered
			c.Groups[groupName] = g
		}
	}
}

// SortedStringKeys returns string map keys in deterministic order.
func SortedStringKeys[V any](values map[string]V) []string {
	return slices.Sorted(maps.Keys(values))
}

// SortedWorktreeBranchNames returns branch names in deterministic order.
// It is an alias for SortedStringKeys, kept for semantic clarity at call sites.
func SortedWorktreeBranchNames(branches map[string]string) []string {
	return SortedStringKeys(branches)
}
