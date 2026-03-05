package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/robertwritescode/git-w/pkg/toml"
	"github.com/spf13/cobra"
)

// Load reads configPath `.gitw` and merges `.gitw.local` if present.
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

	ensureWorkspaceMaps(cfg)

	if err := buildAndValidate(configPath, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func buildAndValidate(configPath string, cfg *WorkspaceConfig) error {
	if err := validateWorktreePaths(configPath, cfg); err != nil {
		return err
	}

	if err := synthesizeWorktreeTargets(cfg); err != nil {
		return err
	}

	return validateRepoPaths(configPath, cfg)
}

func ensureWorkspaceMaps(cfg *WorkspaceConfig) {
	if cfg.Repos == nil {
		cfg.Repos = make(map[string]RepoConfig)
	}

	if cfg.Groups == nil {
		cfg.Groups = make(map[string]GroupConfig)
	}

	if cfg.Worktrees == nil {
		cfg.Worktrees = make(map[string]WorktreeConfig)
	}

	if cfg.Workgroups == nil {
		cfg.Workgroups = make(map[string]WorkgroupConfig)
	}
}

// localDiskConfig is the schema for the .gitw.local file.
type localDiskConfig struct {
	Context    ContextConfig              `toml:"context"`
	Workgroups map[string]WorkgroupConfig `toml:"workgroup,omitempty"`
}

func mergeLocalConfig(cfg *WorkspaceConfig, localPath string) error {
	local, err := readLocalDiskConfig(localPath)
	if err != nil {
		return err
	}

	if local.Context.Active != "" {
		cfg.Context = local.Context
	}

	for name, wg := range local.Workgroups {
		cfg.Workgroups[name] = wg
	}

	return nil
}

// Save writes cfg to configPath atomically (write to .tmp, then rename).
// Only the workspace, repos, and groups sections are written; context lives in .gitw.local.
// Comments and formatting from the original file are preserved where possible.
func Save(configPath string, cfg *WorkspaceConfig) error {
	if err := validateRepoPaths(configPath, cfg); err != nil {
		return err
	}

	newConfig := prepareDiskConfig(cfg)
	data, err := saveWithCommentPreservation(configPath, newConfig)
	if err != nil {
		return err
	}

	return atomicWriteFile(configPath, data)
}

type diskConfig struct {
	Workspace WorkspaceMeta             `toml:"workspace"`
	Repos     map[string]RepoConfig     `toml:"repos,omitempty"`
	Groups    map[string]GroupConfig    `toml:"groups,omitempty"`
	Worktrees map[string]WorktreeConfig `toml:"worktrees,omitempty"`
}

func prepareDiskConfig(cfg *WorkspaceConfig) diskConfig {
	return diskConfig{
		Workspace: cfg.Workspace,
		Repos:     withoutSynthesizedRepos(cfg.Repos, cfg.Worktrees),
		Groups:    withoutSynthesizedGroups(cfg.Groups, cfg.Worktrees),
		Worktrees: cfg.Worktrees,
	}
}

func SaveLocal(configPath string, ctx ContextConfig) error {
	localPath := configPath + ".local"

	existing, err := readLocalDiskConfig(localPath)
	if err != nil {
		return err
	}

	existing.Context = ctx

	return writeLocalDiskConfig(localPath, existing)
}

func SaveLocalWorkgroup(configPath, name string, wg WorkgroupConfig) error {
	localPath := configPath + ".local"

	existing, err := readLocalDiskConfig(localPath)
	if err != nil {
		return err
	}

	if existing.Workgroups == nil {
		existing.Workgroups = make(map[string]WorkgroupConfig)
	}

	existing.Workgroups[name] = wg

	return writeLocalDiskConfig(localPath, existing)
}

func RemoveLocalWorkgroup(configPath, name string) error {
	localPath := configPath + ".local"

	existing, err := readLocalDiskConfig(localPath)
	if err != nil {
		return err
	}

	delete(existing.Workgroups, name)

	return writeLocalDiskConfig(localPath, existing)
}

func readLocalDiskConfig(localPath string) (localDiskConfig, error) {
	data, err := os.ReadFile(localPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return localDiskConfig{}, nil
		}

		return localDiskConfig{}, fmt.Errorf("reading local config %s: %w", localPath, err)
	}

	var cfg localDiskConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return localDiskConfig{}, fmt.Errorf("parsing local config %s: %w", localPath, err)
	}

	return cfg, nil
}

func writeLocalDiskConfig(localPath string, cfg localDiskConfig) error {
	data, err := saveWithCommentPreservation(localPath, cfg)
	if err != nil {
		return err
	}

	return atomicWriteFile(localPath, data)
}

func saveWithCommentPreservation(path string, newConfig interface{}) ([]byte, error) {
	original, err := os.ReadFile(path)
	if err != nil {
		return marshalToml(newConfig)
	}

	var oldConfig interface{}
	if err := toml.Unmarshal(original, &oldConfig); err != nil {
		return marshalToml(newConfig)
	}

	data, err := toml.UpdatePreservingComments(original, oldConfig, newConfig)
	if err != nil {
		return nil, fmt.Errorf("updating config: %w", err)
	}

	return data, nil
}

func marshalToml(cfg interface{}) ([]byte, error) {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}
	return data, nil
}

func atomicWriteFile(path string, data []byte) error {
	tmp, err := createTempFile(filepath.Dir(path), data)
	if err != nil {
		return err
	}

	return commitTempFile(tmp, path)
}

func createTempFile(dir string, data []byte) (string, error) {
	tmp, err := os.CreateTemp(dir, ".gitw-*.tmp")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	tmpName := tmp.Name()
	if err := writeSyncClose(tmp, data); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}

	return tmpName, nil
}

func writeSyncClose(f *os.File, data []byte) (err error) {
	defer closeWithError(f, &err)

	if err := writeTempFileData(f, data); err != nil {
		return err
	}

	if err := f.Chmod(0o600); err != nil {
		return fmt.Errorf("setting temp file permissions: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("syncing temp file: %w", err)
	}

	return nil
}

func closeWithError(f *os.File, err *error) {
	if closeErr := f.Close(); closeErr != nil && *err == nil {
		*err = closeErr
	}
}

func writeTempFileData(f *os.File, data []byte) error {
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	return nil
}

func commitTempFile(tmpName, destPath string) error {
	if err := os.Rename(tmpName, destPath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

// ConfigDir returns the directory containing the config file.
func ConfigDir(configPath string) string {
	return filepath.Dir(configPath)
}

func validateRepoPaths(cfgPath string, cfg *WorkspaceConfig) error {
	for name, rc := range cfg.Repos {
		if _, err := ResolveRepoPath(cfgPath, rc.Path); err != nil {
			return fmt.Errorf("invalid path for repo %q: %w", name, err)
		}
	}

	return nil
}

func validateWorktreePaths(cfgPath string, cfg *WorkspaceConfig) error {
	for name, wt := range cfg.Worktrees {
		if wt.BarePath == "" {
			continue
		}

		if _, err := ResolveRepoPath(cfgPath, wt.BarePath); err != nil {
			return fmt.Errorf("invalid bare_path for worktree set %q: %w", name, err)
		}
	}

	return nil
}

func validateRepoPath(repoPath string) error {
	if strings.TrimSpace(repoPath) == "" {
		return fmt.Errorf("path is empty")
	}

	if filepath.IsAbs(repoPath) {
		return fmt.Errorf("path must be relative")
	}

	return nil
}

// ResolveRepoPath resolves a repo path from config against cfgPath's directory.
func ResolveRepoPath(cfgPath, repoPath string) (string, error) {
	if err := validateRepoPath(repoPath); err != nil {
		return "", err
	}

	cfgRoot := filepath.Clean(ConfigDir(cfgPath))
	canonicalRoot := canonicalPath(cfgRoot)
	canonicalResolved := filepath.Clean(filepath.Join(canonicalRoot, repoPath))

	rel, err := filepath.Rel(canonicalRoot, canonicalResolved)
	if err != nil {
		return "", fmt.Errorf("resolving repo path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path resolves outside workspace root")
	}

	return canonicalResolved, nil
}

// RelPath returns absPath relative to the config file's directory.
func RelPath(cfgPath, absPath string) (string, error) {
	rel, err := filepath.Rel(canonicalPath(ConfigDir(cfgPath)), canonicalPath(absPath))
	if err != nil {
		return "", fmt.Errorf("computing relative path: %w", err)
	}

	return rel, nil
}

func canonicalPath(path string) string {
	clean := filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(clean); err == nil {
		return resolved
	}

	parent := filepath.Dir(clean)
	base := filepath.Base(clean)
	if resolvedParent, err := filepath.EvalSymlinks(parent); err == nil {
		return filepath.Join(resolvedParent, base)
	}

	return clean
}

// LoadCWD discovers and loads the workspace config, starting from the current
// working directory. If override is non-empty it is used as the config path directly
// (e.g. from the --config flag), bypassing discovery.
func LoadCWD(override string) (*WorkspaceConfig, string, error) {
	if override != "" {
		cfg, err := Load(override)
		return cfg, override, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("getting working directory: %w", err)
	}

	cfgPath, err := Discover(cwd)
	if err != nil {
		return nil, "", err
	}

	cfg, err := Load(cfgPath)
	return cfg, cfgPath, err
}

// LoadConfig reads the --config flag from the root command and loads the workspace config.
func LoadConfig(cmd *cobra.Command) (*WorkspaceConfig, string, error) {
	override, err := cmd.Root().PersistentFlags().GetString("config")
	if err != nil {
		return nil, "", err
	}

	return LoadCWD(override)
}

func synthesizeWorktreeTargets(cfg *WorkspaceConfig) error {
	for _, setName := range SortedStringKeys(cfg.Worktrees) {
		if err := synthesizeWorktreeSet(cfg, setName); err != nil {
			return err
		}
	}

	return nil
}

func synthesizeWorktreeSet(cfg *WorkspaceConfig, setName string) error {
	setCfg := cfg.Worktrees[setName]
	if setCfg.Branches == nil {
		setCfg.Branches = make(map[string]string)
		cfg.Worktrees[setName] = setCfg
	}

	if _, exists := cfg.Groups[setName]; exists {
		return fmt.Errorf("worktree set %q conflicts with existing group %q", setName, setName)
	}

	repoNames, err := synthesizeWorktreeRepos(cfg, setName, setCfg)
	if err != nil {
		return err
	}

	cfg.Groups[setName] = GroupConfig{Repos: repoNames}
	return nil
}

func synthesizeWorktreeRepos(cfg *WorkspaceConfig, setName string, setCfg WorktreeConfig) ([]string, error) {
	repoNames := make([]string, 0, len(setCfg.Branches))

	for _, branch := range SortedWorktreeBranchNames(setCfg.Branches) {
		repoName := WorktreeRepoName(setName, branch)
		if _, exists := cfg.Repos[repoName]; exists {
			return nil, fmt.Errorf("worktree branch %q in set %q conflicts with existing repo %q", branch, setName, repoName)
		}

		cfg.Repos[repoName] = RepoConfig{
			Path: setCfg.Branches[branch],
			URL:  setCfg.URL,
		}
		repoNames = append(repoNames, repoName)
	}

	return repoNames, nil
}

func withoutSynthesizedRepos(repos map[string]RepoConfig, worktrees map[string]WorktreeConfig) map[string]RepoConfig {
	synth := make(map[string]struct{})
	for setName, wt := range worktrees {
		for branch := range wt.Branches {
			synth[WorktreeRepoName(setName, branch)] = struct{}{}
		}
	}

	result := make(map[string]RepoConfig)
	for name, rc := range repos {
		if _, isSynth := synth[name]; isSynth {
			continue
		}
		result[name] = rc
	}

	return result
}

func withoutSynthesizedGroups(groups map[string]GroupConfig, worktrees map[string]WorktreeConfig) map[string]GroupConfig {
	result := make(map[string]GroupConfig)
	for name, group := range groups {
		if _, isWorktreeSet := worktrees[name]; isWorktreeSet {
			continue
		}
		result[name] = group
	}

	return result
}
