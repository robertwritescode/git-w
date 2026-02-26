package workspace

import (
	"fmt"
	"io"
	"path/filepath"
)

// WorkspaceConfig is the merged result of `.gitw` and `.gitw.local`.
// Repos and Groups maps are always non-nil after loading.
type WorkspaceConfig struct {
	Workspace WorkspaceMeta          `toml:"workspace"`
	Context   ContextConfig          `toml:"context"` // sourced from .gitw.local
	Repos     map[string]RepoConfig  `toml:"repos"`
	Groups    map[string]GroupConfig `toml:"groups"`
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

// writef writes formatted output, discarding write errors
// (appropriate for terminal I/O where write failures are unrecoverable).
func writef(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...)
}
