package config

// WorkspaceConfig is the merged result of `.gitworkspace` and `.gitworkspace.local`.
// Repos and Groups maps are always non-nil after loading.
type WorkspaceConfig struct {
	Workspace WorkspaceMeta          `toml:"workspace"`
	Context   ContextConfig          `toml:"context"` // sourced from .gitworkspace.local
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

// ContextConfig holds the active context (stored in .gitworkspace.local).
type ContextConfig struct {
	Active string `toml:"active"`
}
