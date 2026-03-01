package repo

import (
	"fmt"

	"github.com/robertwritescode/git-w/pkg/workspace"
)

// Filter resolves names as repo and/or group names.
// With no names: falls back to active context, then all repos.
// With names: expands group names, deduplicates by repo name, errors on unknown names.
func Filter(cfg *workspace.WorkspaceConfig, cfgPath string, names []string) ([]Repo, error) {
	if len(names) == 0 {
		return ForContext(cfg, cfgPath)
	}

	return resolveTargets(cfg, cfgPath, names)
}

// ForContext returns the active context's repos, or all repos if no context is set.
func ForContext(cfg *workspace.WorkspaceConfig, cfgPath string) ([]Repo, error) {
	if cfg.Context.Active == "" {
		return FromConfig(cfg, cfgPath), nil
	}

	g, ok := cfg.Groups[cfg.Context.Active]
	if !ok {
		return nil, fmt.Errorf("active context group %q not found", cfg.Context.Active)
	}

	return forGroup(cfg, cfgPath, cfg.Context.Active, g)
}

// ForGroup returns repos belonging to the named group.
func ForGroup(cfg *workspace.WorkspaceConfig, cfgPath string, groupName string) ([]Repo, error) {
	g, ok := cfg.Groups[groupName]
	if !ok {
		return nil, fmt.Errorf("group %q not found", groupName)
	}

	return forGroup(cfg, cfgPath, groupName, g)
}

func resolveTargets(cfg *workspace.WorkspaceConfig, cfgPath string, names []string) ([]Repo, error) {
	all := FromConfig(cfg, cfgPath)
	byRepo := repoIndex(all)
	seen := make(map[string]bool)

	var result []Repo
	for _, name := range names {
		resolved, err := resolveTarget(cfg, cfgPath, name, byRepo)
		if err != nil {
			return nil, err
		}

		for _, r := range resolved {
			if !seen[r.Name] {
				seen[r.Name] = true
				result = append(result, r)
			}
		}
	}
	return result, nil
}

func resolveTarget(cfg *workspace.WorkspaceConfig, cfgPath, name string, byRepo map[string]Repo) ([]Repo, error) {
	if r, ok := byRepo[name]; ok {
		return []Repo{r}, nil
	}

	if g, ok := cfg.Groups[name]; ok {
		return forGroup(cfg, cfgPath, name, g)
	}

	return nil, fmt.Errorf("%q is not a registered repo or group", name)
}

func forGroup(cfg *workspace.WorkspaceConfig, cfgPath string, groupName string, g workspace.GroupConfig) ([]Repo, error) {
	for _, member := range g.Repos {
		if _, ok := cfg.Repos[member]; !ok {
			return nil, fmt.Errorf("group %q references unknown repo %q", groupName, member)
		}
	}

	return FromNames(cfg, cfgPath, g.Repos), nil
}

func repoIndex(repos []Repo) map[string]Repo {
	m := make(map[string]Repo, len(repos))

	for _, r := range repos {
		m[r.Name] = r
	}

	return m
}
