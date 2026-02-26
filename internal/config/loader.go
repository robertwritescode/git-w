package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Load reads configPath `.gitworkspace` and merges `.gitworkspace.local` if present.
// Returns a WorkspaceConfig with non-nil Repos and Groups maps.
func Load(configPath string) (*WorkspaceConfig, error) {
	cfg, err := loadMainConfig(configPath)
	if err != nil {
		return nil, err
	}

	if err := mergeLocalConfig(cfg, configPath+".local"); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadMainConfig(configPath string) (*WorkspaceConfig, error) {
	cfg := &WorkspaceConfig{}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", configPath, err)
	}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", configPath, err)
	}

	if cfg.Repos == nil {
		cfg.Repos = make(map[string]RepoConfig)
	}
	if cfg.Groups == nil {
		cfg.Groups = make(map[string]GroupConfig)
	}

	return cfg, nil
}

func mergeLocalConfig(cfg *WorkspaceConfig, localPath string) error {
	localData, err := os.ReadFile(localPath)
	if err != nil {
		return nil // .local is optional
	}

	var local WorkspaceConfig
	if err := toml.Unmarshal(localData, &local); err != nil {
		return fmt.Errorf("parsing .local config %s: %w", localPath, err)
	}

	if local.Context.Active != "" {
		cfg.Context = local.Context
	}
	return nil
}

// Save writes cfg to configPath atomically (write to .tmp, then rename).
// Only the workspace, repos, and groups sections are written; context lives in .gitworkspace.local.
func Save(configPath string, cfg *WorkspaceConfig) error {
	type diskConfig struct {
		Workspace WorkspaceMeta          `toml:"workspace"`
		Repos     map[string]RepoConfig  `toml:"repos,omitempty"`
		Groups    map[string]GroupConfig `toml:"groups,omitempty"`
	}
	data, err := toml.Marshal(diskConfig{
		Workspace: cfg.Workspace,
		Repos:     cfg.Repos,
		Groups:    cfg.Groups,
	})
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return atomicWriteFile(configPath, data)
}

// SaveLocal writes the context section to configPath+".local" atomically.
func SaveLocal(configPath string, ctx ContextConfig) error {
	type localFile struct {
		Context ContextConfig `toml:"context"`
	}
	data, err := toml.Marshal(localFile{Context: ctx})
	if err != nil {
		return fmt.Errorf("marshaling local config: %w", err)
	}

	return atomicWriteFile(configPath+".local", data)
}

func atomicWriteFile(path string, data []byte) error {
	tmp := path + ".tmp"

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

func ConfigDir(configPath string) string {
	return filepath.Dir(configPath)
}
