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
	Metarepo   MetarepoConfig             `toml:"metarepo"`
	Workspaces []WorkspaceBlock           `toml:"workspace"`
	Remotes    []RemoteConfig             // in-memory; populated from [[remote]] list by loader
	SyncPairs  []SyncPairConfig           // in-memory; populated from [[sync_pair]] list by loader
	Context    ContextConfig              `toml:"context"` // sourced from .gitw.local
	Repos      map[string]RepoConfig      // in-memory only; populated from [[repo]] list by loader
	Groups     map[string]GroupConfig     `toml:"groups"`
	Worktrees  map[string]WorktreeConfig  `toml:"worktrees"`
	Workgroups map[string]WorkgroupConfig `toml:"workgroup"` // sourced from .gitw.local
	Warnings   []string                   // in-memory only; populated at load time
}

// WorkgroupConfig is a local workgroup entry (stored only in .gitw.local).
type WorkgroupConfig struct {
	Repos   []string `toml:"repos"`
	Branch  string   `toml:"branch"`
	Created string   `toml:"created,omitempty"`
}

// MetarepoConfig holds top-level metarepo settings (formerly WorkspaceMeta, TOML key: metarepo).
type MetarepoConfig struct {
	Name              string   `toml:"name"`
	DefaultRemotes    []string `toml:"default_remotes,omitempty"`
	AgenticFrameworks []string `toml:"agentic_frameworks,omitempty"`
	AutoGitignore     *bool    `toml:"auto_gitignore"` // nil means true (default on)
	SyncPush          *bool    `toml:"sync_push"`      // nil means true (default on)
	DefaultBranch     string   `toml:"default_branch,omitempty"`
	BranchSyncSource  *bool    `toml:"branch_sync_source"`  // nil means true (default on)
	BranchSetUpstream *bool    `toml:"branch_set_upstream"` // nil means true (default on)
	BranchPush        *bool    `toml:"branch_push"`         // nil means true (default on)
}

// WorkspaceBlock is one entry in the [[workspace]] array-of-tables.
type WorkspaceBlock struct {
	Name        string   `toml:"name"`
	Description string   `toml:"description,omitempty"`
	Repos       []string `toml:"repos,omitempty"`
}

// BranchAction is the typed action for a branch rule.
type BranchAction string

const (
	ActionAllow       BranchAction = "allow"
	ActionBlock       BranchAction = "block"
	ActionWarn        BranchAction = "warn"
	ActionRequireFlag BranchAction = "require-flag"
)

// BranchRuleConfig is one [[remote.branch_rule]] entry.
type BranchRuleConfig struct {
	Pattern   string       `toml:"pattern,omitempty"`
	Action    BranchAction `toml:"action"`
	Reason    string       `toml:"reason,omitempty"`
	Flag      string       `toml:"flag,omitempty"`
	Untracked *bool        `toml:"untracked,omitempty"`
	Explicit  *bool        `toml:"explicit,omitempty"`
}

// RemoteConfig is one [[remote]] entry.
type RemoteConfig struct {
	Name        string             `toml:"name"`
	Kind        string             `toml:"kind,omitempty"`
	URL         string             `toml:"url,omitempty"`
	User        string             `toml:"user,omitempty"`
	TokenEnv    string             `toml:"token_env,omitempty"`
	Org         string             `toml:"org,omitempty"`
	RepoPrefix  string             `toml:"repo_prefix,omitempty"`
	RepoSuffix  string             `toml:"repo_suffix,omitempty"`
	Direction   string             `toml:"direction,omitempty"`
	PushMode    string             `toml:"push_mode,omitempty"`
	FetchMode   string             `toml:"fetch_mode,omitempty"`
	UseSSH      bool               `toml:"use_ssh,omitempty"`
	SSHHost     string             `toml:"ssh_host,omitempty"`
	Critical    bool               `toml:"critical,omitempty"`
	Private     bool               `toml:"private,omitempty"`
	BranchRules []BranchRuleConfig `toml:"branch_rule,omitempty"`
}

// SyncPairConfig is one [[sync_pair]] entry.
type SyncPairConfig struct {
	From string   `toml:"from"`
	To   string   `toml:"to"`
	Refs []string `toml:"refs,omitempty"`
}

// MergeRemote merges base and override RemoteConfig. For each field, the
// override value wins if non-zero; otherwise the base value is used.
// BranchRules from override replace base BranchRules entirely if non-nil.
func MergeRemote(base, override RemoteConfig) RemoteConfig {
	merged := base

	if override.Name != "" {
		merged.Name = override.Name
	}

	if override.Kind != "" {
		merged.Kind = override.Kind
	}

	if override.URL != "" {
		merged.URL = override.URL
	}

	if override.User != "" {
		merged.User = override.User
	}

	if override.TokenEnv != "" {
		merged.TokenEnv = override.TokenEnv
	}

	if override.Org != "" {
		merged.Org = override.Org
	}

	if override.RepoPrefix != "" {
		merged.RepoPrefix = override.RepoPrefix
	}

	if override.RepoSuffix != "" {
		merged.RepoSuffix = override.RepoSuffix
	}

	if override.Direction != "" {
		merged.Direction = override.Direction
	}

	if override.PushMode != "" {
		merged.PushMode = override.PushMode
	}

	if override.FetchMode != "" {
		merged.FetchMode = override.FetchMode
	}

	if override.UseSSH {
		merged.UseSSH = override.UseSSH
	}

	if override.SSHHost != "" {
		merged.SSHHost = override.SSHHost
	}

	if override.Critical {
		merged.Critical = override.Critical
	}

	if override.Private {
		merged.Private = override.Private
	}

	if override.BranchRules != nil {
		merged.BranchRules = override.BranchRules
	}

	return merged
}

// MergeSyncPair merges base and override SyncPairConfig. For each field,
// the override value wins if non-zero; otherwise the base value is used.
// Refs from override replace base Refs entirely if non-empty.
func MergeSyncPair(base, override SyncPairConfig) SyncPairConfig {
	merged := base

	if override.From != "" {
		merged.From = override.From
	}

	if override.To != "" {
		merged.To = override.To
	}

	if len(override.Refs) > 0 {
		merged.Refs = override.Refs
	}

	return merged
}

// RepoConfig represents one tracked repository.
type RepoConfig struct {
	Name          string   `toml:"name"`
	Path          string   `toml:"path"`
	CloneURL      string   `toml:"clone_url,omitempty"`
	Flags         []string `toml:"flags,omitempty"`
	DefaultBranch string   `toml:"default_branch,omitempty"`
	TrackBranch   string   `toml:"track_branch,omitempty"`
	Upstream      string   `toml:"upstream,omitempty"`
	Remotes       []string `toml:"remotes,omitempty"`
}

// IsAlias reports whether this repo is an env alias (has track_branch set).
func (r RepoConfig) IsAlias() bool {
	return r.TrackBranch != ""
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

// RepoByName returns the RepoConfig for the given name and whether it was found.
func (c *WorkspaceConfig) RepoByName(name string) (RepoConfig, bool) {
	rc, ok := c.Repos[name]
	return rc, ok
}

// RemoteByName returns the RemoteConfig with the given name and whether it was found.
func (c *WorkspaceConfig) RemoteByName(name string) (RemoteConfig, bool) {
	for _, r := range c.Remotes {
		if r.Name == name {
			return r, true
		}
	}

	return RemoteConfig{}, false
}

// AutoGitignoreEnabled reports whether auto-gitignore is on (nil means default true).
func (c WorkspaceConfig) AutoGitignoreEnabled() bool {
	return c.Metarepo.AutoGitignore == nil || *c.Metarepo.AutoGitignore
}

// SyncPushEnabled reports whether sync runs push by default (nil means true).
func (c WorkspaceConfig) SyncPushEnabled() bool {
	return c.Metarepo.SyncPush == nil || *c.Metarepo.SyncPush
}

// BranchSyncSourceEnabled reports whether branch creation syncs the source branch (nil means true).
func (c WorkspaceConfig) BranchSyncSourceEnabled() bool {
	return c.Metarepo.BranchSyncSource == nil || *c.Metarepo.BranchSyncSource
}

// BranchSetUpstreamEnabled reports whether branch creation sets upstream (nil means true).
func (c WorkspaceConfig) BranchSetUpstreamEnabled() bool {
	return c.Metarepo.BranchSetUpstream == nil || *c.Metarepo.BranchSetUpstream
}

// BranchPushEnabled reports whether branch creation pushes by default (nil means true).
func (c WorkspaceConfig) BranchPushEnabled() bool {
	return c.Metarepo.BranchPush == nil || *c.Metarepo.BranchPush
}

// ResolveDefaultBranch returns the source branch for a repo.
func (c WorkspaceConfig) ResolveDefaultBranch(repoName string) string {
	// Worktree repos use their own branch as the source (e.g. infra-dev -> dev).
	if branch, ok := c.WorktreeBranchForRepo(repoName); ok {
		return branch
	}

	if repoCfg, ok := c.Repos[repoName]; ok && repoCfg.DefaultBranch != "" {
		return repoCfg.DefaultBranch
	}

	if c.Metarepo.DefaultBranch != "" {
		return c.Metarepo.DefaultBranch
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
