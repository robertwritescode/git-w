package config

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
	Name              string `toml:"name"`
	AutoGitignore     *bool  `toml:"auto_gitignore"` // nil means true (default on)
	SyncPush          *bool  `toml:"sync_push"`      // nil means true (default on)
	DefaultBranch     string `toml:"default_branch"`
	BranchSyncSource  *bool  `toml:"branch_sync_source"`  // nil means true (default on)
	BranchSetUpstream *bool  `toml:"branch_set_upstream"` // nil means true (default on)
	BranchPush        *bool  `toml:"branch_push"`         // nil means true (default on)
}

// RepoConfig represents one tracked repository.
type RepoConfig struct {
	Path          string   `toml:"path"`
	URL           string   `toml:"url,omitempty"`
	Flags         []string `toml:"flags,omitempty"`
	DefaultBranch string   `toml:"default_branch,omitempty"`
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

// SyncPushEnabled reports whether sync runs push by default (nil means true).
func (c WorkspaceConfig) SyncPushEnabled() bool {
	return c.Workspace.SyncPush == nil || *c.Workspace.SyncPush
}

// BranchSyncSourceEnabled reports whether branch creation syncs the source branch (nil means true).
func (c WorkspaceConfig) BranchSyncSourceEnabled() bool {
	return c.Workspace.BranchSyncSource == nil || *c.Workspace.BranchSyncSource
}

// BranchSetUpstreamEnabled reports whether branch creation sets upstream (nil means true).
func (c WorkspaceConfig) BranchSetUpstreamEnabled() bool {
	return c.Workspace.BranchSetUpstream == nil || *c.Workspace.BranchSetUpstream
}

// BranchPushEnabled reports whether branch creation pushes by default (nil means true).
func (c WorkspaceConfig) BranchPushEnabled() bool {
	return c.Workspace.BranchPush == nil || *c.Workspace.BranchPush
}

// ResolveDefaultBranch returns the source branch for a repo.
func (c WorkspaceConfig) ResolveDefaultBranch(repoName string) string {
	if repoCfg, ok := c.Repos[repoName]; ok && repoCfg.DefaultBranch != "" {
		return repoCfg.DefaultBranch
	}

	if c.Workspace.DefaultBranch != "" {
		return c.Workspace.DefaultBranch
	}

	return "main"
}

// WorktreeBranchForRepo returns the worktree branch for a synthesized repo name.
func (c WorkspaceConfig) WorktreeBranchForRepo(repoName string) (string, bool) {
	for setName, wt := range c.Worktrees {
		for branch := range wt.Branches {
			if WorktreeRepoName(setName, branch) == repoName {
				return branch, true
			}
		}
	}

	return "", false
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

// WorktreeRepoToSetIndex returns a map of synthesized repo name to worktree set name.
func WorktreeRepoToSetIndex(c *WorkspaceConfig) map[string]string {
	result := make(map[string]string)

	for setName, wt := range c.Worktrees {
		for _, branch := range SortedStringKeys(wt.Branches) {
			result[WorktreeRepoName(setName, branch)] = setName
		}
	}

	return result
}

// RemoveRepoFromManualGroups removes repoName from every group that is not
// synthesized from a worktree set. Must be called before deleting the set
// from cfg.Worktrees so the synthesized group can still be identified.
func (c *WorkspaceConfig) RemoveRepoFromManualGroups(repoName string) {
	for groupName, g := range c.Groups {
		if _, isSynth := c.Worktrees[groupName]; isSynth {
			continue
		}

		c.updateGroupWithoutRepo(groupName, g, repoName)
	}
}

func (c *WorkspaceConfig) updateGroupWithoutRepo(groupName string, g GroupConfig, repoName string) {
	filtered := filterGroupRepos(g.Repos, repoName)
	if len(filtered) == len(g.Repos) {
		return
	}

	g.Repos = filtered
	c.Groups[groupName] = g
}

func filterGroupRepos(repos []string, exclude string) []string {
	filtered := make([]string, 0, len(repos))

	for _, r := range repos {
		if r != exclude {
			filtered = append(filtered, r)
		}
	}

	return filtered
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
