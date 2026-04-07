package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/robertwritescode/git-w/pkg/agents"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/toml"
	"github.com/spf13/cobra"
)

// Load reads configPath `.gitw`, merges `.git/.gitw` if present, then
// merges `.gitw.local` if present.
// Returns a WorkspaceConfig with non-nil Repos and Groups maps.
func Load(configPath string) (*WorkspaceConfig, error) {
	cfg, err := loadMainConfig(configPath)
	if err != nil {
		return nil, err
	}

	if err := mergePrivateConfig(cfg, configPath); err != nil {
		return nil, err
	}

	if err := mergeLocalConfig(cfg, configPath+".local"); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadMainConfig(configPath string) (*WorkspaceConfig, error) {
	var dc diskConfig

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", configPath, err)
	}

	if err := toml.Unmarshal(data, &dc); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", configPath, err)
	}

	cfg := &WorkspaceConfig{
		Metarepo:    dc.Metarepo,
		Workspaces:  dc.Workspaces,
		Remotes:     dc.RemoteList,
		SyncPairs:   dc.SyncPairList,
		Workstreams: dc.WorkstreamList,
		Groups:      dc.Groups,
		Worktrees:   dc.Worktrees,
	}

	if err := buildReposIndex(cfg, dc.RepoList); err != nil {
		return nil, err
	}

	ensureWorkspaceMaps(cfg)

	cfg.V1WorkgroupCount = countV1WorkgroupBlocks(data)

	if err := buildAndValidate(configPath, cfg); err != nil {
		return nil, err
	}

	applyMetarepoDefaults(cfg)

	return cfg, nil
}

func applyMetarepoDefaults(cfg *WorkspaceConfig) {
	if len(cfg.Metarepo.AgenticFrameworks) == 0 {
		cfg.Metarepo.AgenticFrameworks = []string{"gsd"}
	}
}

func buildAndValidate(configPath string, cfg *WorkspaceConfig) error {
	if err := detectV1Workgroups(cfg); err != nil {
		return err
	}

	if err := validateRepoNames(cfg); err != nil {
		return err
	}

	if err := validateWorktreePaths(configPath, cfg); err != nil {
		return err
	}

	if err := synthesizeWorktreeTargets(cfg); err != nil {
		return err
	}

	if err := validateRepoPaths(configPath, cfg); err != nil {
		return err
	}

	warnNonConformingRepoPaths(cfg)

	if err := validateAgenticFrameworks(cfg); err != nil {
		return err
	}

	if err := validateRemotes(configPath, cfg); err != nil {
		return err
	}

	if err := validateWorkstreams(configPath, cfg); err != nil {
		return err
	}

	if err := validateSyncPairFields(cfg); err != nil {
		return err
	}

	if err := detectSyncCycles(cfg); err != nil {
		return err
	}

	return validateAliasFields(cfg)
}

func validateAliasFields(cfg *WorkspaceConfig) error {
	// upstream -> track_branch -> first repo name that claimed it
	seen := make(map[string]map[string]string)

	for name, rc := range cfg.Repos {
		hasTrack := rc.TrackBranch != ""
		hasUp := rc.Upstream != ""

		if hasTrack != hasUp {
			return fmt.Errorf("repo %q: track_branch and upstream must both be set or both be absent", name)
		}

		if !hasTrack {
			continue
		}

		if seen[rc.Upstream] == nil {
			seen[rc.Upstream] = make(map[string]string)
		}

		if prior, dup := seen[rc.Upstream][rc.TrackBranch]; dup {
			return fmt.Errorf("repo %q: track_branch %q already used by %q in upstream group %q",
				name, rc.TrackBranch, prior, rc.Upstream)
		}

		seen[rc.Upstream][rc.TrackBranch] = name
	}

	return nil
}

func validateAgenticFrameworks(cfg *WorkspaceConfig) error {
	if _, err := agents.FrameworksFor(cfg.Metarepo.AgenticFrameworks); err != nil {
		return fmt.Errorf("agentic_frameworks: %w", err)
	}

	return nil
}

func validateRemotes(cfgPath string, cfg *WorkspaceConfig) error {
	isPrivateFile := strings.HasSuffix(filepath.ToSlash(cfgPath), ".git/.gitw")
	seen := make(map[string]struct{}, len(cfg.Remotes))

	for i, r := range cfg.Remotes {
		if r.Name == "" {
			return fmt.Errorf("[[remote]] entry at index %d: missing required name field", i)
		}

		if _, dup := seen[r.Name]; dup {
			return fmt.Errorf("duplicate [[remote]] name %q", r.Name)
		}
		seen[r.Name] = struct{}{}

		if r.Kind != "" {
			switch r.Kind {
			case "gitea", "forgejo", "github", "generic":
			default:
				return fmt.Errorf("remote %q: kind %q is not valid; must be one of: gitea, forgejo, github, generic", r.Name, r.Kind)
			}
		}

		for j, rule := range r.BranchRules {
			if rule.Action != "" {
				switch rule.Action {
				case ActionAllow, ActionBlock, ActionWarn, ActionRequireFlag:
				default:
					return fmt.Errorf("remote %q branch_rule[%d]: action %q is not valid; must be one of: allow, block, warn, require-flag", r.Name, j, rule.Action)
				}
			}
		}

		if r.Private && !isPrivateFile {
			return fmt.Errorf("remote %q: private remotes must be defined in .git/.gitw, not .gitw", r.Name)
		}
	}

	return nil
}

func validateSyncPairFields(cfg *WorkspaceConfig) error {
	type pairKey struct{ from, to string }
	seen := make(map[pairKey]struct{}, len(cfg.SyncPairs))

	for i, p := range cfg.SyncPairs {
		if p.From == "" {
			return fmt.Errorf("[[sync_pair]] entry at index %d: missing required %q field", i, "from")
		}

		if p.To == "" {
			return fmt.Errorf("[[sync_pair]] entry at index %d: missing required %q field", i, "to")
		}

		k := pairKey{p.From, p.To}
		if _, dup := seen[k]; dup {
			return fmt.Errorf("duplicate [[sync_pair]] (from=%q, to=%q)", p.From, p.To)
		}

		seen[k] = struct{}{}
	}

	return nil
}

func validateWorkstreams(configPath string, cfg *WorkspaceConfig) error {
	entries, err := loadWorkstreamRawEntries(configPath)
	if err != nil {
		return err
	}

	if len(entries) != len(cfg.Workstreams) {
		return fmt.Errorf("[[workstream]] parse mismatch: expected %d entries, got %d", len(cfg.Workstreams), len(entries))
	}

	knownRemotes := make(map[string]struct{}, len(cfg.Remotes))
	for _, remote := range cfg.Remotes {
		knownRemotes[remote.Name] = struct{}{}
	}

	seenNames := make(map[string]struct{}, len(cfg.Workstreams))

	for i := range cfg.Workstreams {
		entry := entries[i]
		if err := validateWorkstreamEntryKeys(i, entry); err != nil {
			return err
		}

		if _, ok := entry["remotes"]; !ok {
			return fmt.Errorf("[[workstream]] entry at index %d: missing required remotes key", i)
		}

		workstream := &cfg.Workstreams[i]
		if strings.TrimSpace(workstream.Name) == "" {
			return fmt.Errorf("[[workstream]] entry at index %d: missing required name field", i)
		}

		if _, dup := seenNames[workstream.Name]; dup {
			return fmt.Errorf("duplicate [[workstream]] name %q", workstream.Name)
		}
		seenNames[workstream.Name] = struct{}{}

		seenWorkstreamRemotes := make(map[string]struct{}, len(workstream.Remotes))
		for _, remoteName := range workstream.Remotes {
			if _, ok := knownRemotes[remoteName]; !ok {
				return fmt.Errorf("workstream %q: unknown remote %q", workstream.Name, remoteName)
			}

			if _, dup := seenWorkstreamRemotes[remoteName]; dup {
				return fmt.Errorf("workstream %q: duplicate remote %q", workstream.Name, remoteName)
			}
			seenWorkstreamRemotes[remoteName] = struct{}{}
		}

		sort.Strings(workstream.Remotes)
	}

	sort.Slice(cfg.Workstreams, func(i, j int) bool {
		return cfg.Workstreams[i].Name < cfg.Workstreams[j].Name
	})

	return nil
}

func loadWorkstreamRawEntries(configPath string) ([]map[string]any, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", configPath, err)
	}

	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", configPath, err)
	}

	return parseWorkstreamEntries(raw)
}

func parseWorkstreamEntries(raw map[string]any) ([]map[string]any, error) {
	rawWorkstreams, ok := raw["workstream"]
	if !ok {
		return nil, nil
	}

	var list []any
	switch v := rawWorkstreams.(type) {
	case []any:
		list = v
	case []map[string]any:
		list = make([]any, 0, len(v))
		for _, item := range v {
			list = append(list, item)
		}
	default:
		return nil, fmt.Errorf("[[workstream]] should be an array of tables")
	}

	entries := make([]map[string]any, 0, len(list))
	for i, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("[[workstream]] entry at index %d is not a table", i)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func validateWorkstreamEntryKeys(index int, entry map[string]any) error {
	for key := range entry {
		if key == "name" || key == "remotes" {
			continue
		}

		return fmt.Errorf("[[workstream]] entry at index %d: unknown key %q", index, key)
	}

	return nil
}

func detectSyncCycles(cfg *WorkspaceConfig) error {
	adj := make(map[string][]string)
	for _, p := range cfg.SyncPairs {
		adj[p.From] = append(adj[p.From], p.To)
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	for node := range adj {
		if visited[node] {
			continue
		}

		if cycle := dfsSyncCycle(node, adj, visited, inStack, []string{node}); cycle != nil {
			return fmt.Errorf("sync_pair cycle detected: %s", strings.Join(cycle, " → "))
		}
	}

	return nil
}

func dfsSyncCycle(node string, adj map[string][]string, visited, inStack map[string]bool, path []string) []string {
	visited[node] = true
	inStack[node] = true

	for _, neighbor := range adj[node] {
		if inStack[neighbor] {
			// cycle found: find the start of the cycle in path, append closing node
			for i, n := range path {
				if n == neighbor {
					cycle := make([]string, len(path[i:])+1)
					copy(cycle, path[i:])
					cycle[len(cycle)-1] = neighbor
					return cycle
				}
			}
			return append(path, neighbor)
		}

		if !visited[neighbor] {
			if cycle := dfsSyncCycle(neighbor, adj, visited, inStack, append(path, neighbor)); cycle != nil {
				return cycle
			}
		}
	}

	inStack[node] = false
	return nil
}

func warnNonConformingRepoPaths(cfg *WorkspaceConfig) {
	synthIndex := WorktreeRepoToSetIndex(cfg)

	for name, rc := range cfg.Repos {
		if _, isSynth := synthIndex[name]; isSynth {
			continue
		}

		clean := filepath.Clean(rc.Path)
		parts := strings.Split(clean, string(filepath.Separator))
		if len(parts) == 2 && parts[0] == "repos" && parts[1] != "" {
			continue
		}

		suggested := "repos/" + filepath.Base(rc.Path)
		cfg.Warnings = append(cfg.Warnings, fmt.Sprintf(
			"warning: repo %q path %q does not follow repos/<n> convention; suggested: %q; run 'git w migrate' to update",
			name, rc.Path, suggested,
		))
	}

	sort.Strings(cfg.Warnings)
}

func detectV1Workgroups(cfg *WorkspaceConfig) error {
	if cfg.V1WorkgroupCount == 0 {
		return nil
	}

	return fmt.Errorf("v1 config detected: found %d [[workgroup]] block(s) \u2014 run 'git w migrate' to upgrade", cfg.V1WorkgroupCount)
}

// countV1WorkgroupBlocks counts [[workgroup]] array-of-tables headers in raw TOML bytes.
// It looks for lines that are exactly "[[workgroup]]" or "[[workgroup]]" with trailing whitespace.
func countV1WorkgroupBlocks(data []byte) int {
	count := 0

	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "[[workgroup]]" {
			count++
		}
	}

	return count
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

// buildReposIndex converts a [[repo]] slice into the in-memory cfg.Repos map.
func buildReposIndex(cfg *WorkspaceConfig, list []RepoConfig) error {
	cfg.Repos = make(map[string]RepoConfig, len(list))

	for _, rc := range list {
		if rc.Name == "" {
			return fmt.Errorf("missing required name field in [[repo]] entry")
		}

		if _, exists := cfg.Repos[rc.Name]; exists {
			return fmt.Errorf("duplicate [[repo]] name %q", rc.Name)
		}

		cfg.Repos[rc.Name] = rc
	}

	return nil
}

func validateRepoNames(cfg *WorkspaceConfig) error {
	for name := range cfg.Repos {
		if name == "" {
			return fmt.Errorf("missing required name field in [[repo]] entry")
		}
	}

	return nil
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

func privateConfigPath(cfgPath string) string {
	return filepath.Join(filepath.Dir(cfgPath), ".git", ".gitw")
}

// mergePrivateConfig reads .git/.gitw (if present) and merges its blocks
// into cfg using field-level semantics: private file wins on all conflicts.
// Absent .git/.gitw is silently skipped.
func mergePrivateConfig(cfg *WorkspaceConfig, cfgPath string) error {
	privatePath := privateConfigPath(cfgPath)

	data, err := os.ReadFile(privatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reading private config %s: %w", privatePath, err)
	}

	var dc diskConfig
	if err := toml.Unmarshal(data, &dc); err != nil {
		return fmt.Errorf("parsing private config %s: %w", privatePath, err)
	}

	cfg.Metarepo = mergeMetarepo(cfg.Metarepo, dc.Metarepo)

	if err := mergePrivateRemotes(cfg, dc.RemoteList); err != nil {
		return err
	}

	if err := mergePrivateRepos(cfg, dc.RepoList); err != nil {
		return err
	}

	mergePrivateSyncPairs(cfg, dc.SyncPairList)
	mergePrivateWorkstreams(cfg, dc.WorkstreamList)
	mergePrivateWorkspaces(cfg, dc.Workspaces)

	return nil
}

func mergePrivateRemotes(cfg *WorkspaceConfig, overrides []RemoteConfig) error {
	idx := make(map[string]int, len(cfg.Remotes))
	for i, r := range cfg.Remotes {
		idx[r.Name] = i
	}

	for _, override := range overrides {
		if i, ok := idx[override.Name]; ok {
			cfg.Remotes[i] = MergeRemote(cfg.Remotes[i], override)
		} else {
			cfg.Remotes = append(cfg.Remotes, override)
		}
	}

	return nil
}

func mergePrivateRepos(cfg *WorkspaceConfig, overrides []RepoConfig) error {
	for _, override := range overrides {
		base, ok := cfg.Repos[override.Name]
		if !ok {
			return fmt.Errorf("private config: repo %q is not declared in .gitw", override.Name)
		}
		cfg.Repos[override.Name] = MergeRepo(base, override)
	}

	return nil
}

func mergePrivateSyncPairs(cfg *WorkspaceConfig, overrides []SyncPairConfig) {
	type key struct{ from, to string }
	idx := make(map[key]int, len(cfg.SyncPairs))
	for i, p := range cfg.SyncPairs {
		idx[key{p.From, p.To}] = i
	}

	for _, override := range overrides {
		k := key{override.From, override.To}
		if i, ok := idx[k]; ok {
			cfg.SyncPairs[i] = MergeSyncPair(cfg.SyncPairs[i], override)
		} else {
			cfg.SyncPairs = append(cfg.SyncPairs, override)
		}
	}
}

func mergePrivateWorkstreams(cfg *WorkspaceConfig, overrides []WorkstreamConfig) {
	idx := make(map[string]int, len(cfg.Workstreams))
	for i, w := range cfg.Workstreams {
		idx[w.Name] = i
	}

	for _, override := range overrides {
		if i, ok := idx[override.Name]; ok {
			cfg.Workstreams[i] = MergeWorkstream(cfg.Workstreams[i], override)
		} else {
			cfg.Workstreams = append(cfg.Workstreams, override)
		}
	}
}

func mergePrivateWorkspaces(cfg *WorkspaceConfig, overrides []WorkspaceBlock) {
	idx := make(map[string]int, len(cfg.Workspaces))
	for i, w := range cfg.Workspaces {
		idx[w.Name] = i
	}

	for _, override := range overrides {
		if i, ok := idx[override.Name]; ok {
			cfg.Workspaces[i] = MergeWorkspace(cfg.Workspaces[i], override)
		} else {
			cfg.Workspaces = append(cfg.Workspaces, override)
		}
	}
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
	Metarepo       MetarepoConfig            `toml:"metarepo"`
	Workspaces     []WorkspaceBlock          `toml:"workspace,omitempty"`
	RepoList       []RepoConfig              `toml:"repo,omitempty"`
	RemoteList     []RemoteConfig            `toml:"remote,omitempty"`
	SyncPairList   []SyncPairConfig          `toml:"sync_pair,omitempty"`
	WorkstreamList []WorkstreamConfig        `toml:"workstream,omitempty"`
	Groups         map[string]GroupConfig    `toml:"groups,omitempty"`
	Worktrees      map[string]WorktreeConfig `toml:"worktrees,omitempty"`
}

func prepareDiskConfig(cfg *WorkspaceConfig) diskConfig {
	return diskConfig{
		Metarepo:       cfg.Metarepo,
		Workspaces:     cfg.Workspaces,
		RepoList:       buildRepoList(cfg),
		RemoteList:     cfg.Remotes,
		SyncPairList:   cfg.SyncPairs,
		WorkstreamList: cfg.Workstreams,
		Groups:         withoutSynthesizedGroups(cfg.Groups, cfg.Worktrees),
		Worktrees:      cfg.Worktrees,
	}
}

func buildRepoList(cfg *WorkspaceConfig) []RepoConfig {
	plain := withoutSynthesizedRepos(cfg.Repos, cfg.Worktrees)
	names := SortedStringKeys(plain)
	list := make([]RepoConfig, 0, len(names))

	for _, name := range names {
		rc := plain[name]
		rc.Name = name
		list = append(list, rc)
	}

	return list
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

// WorkgroupWorktreePath returns the absolute path to a repo's worktree within a workgroup.
func WorkgroupWorktreePath(cfgPath, wgName, repoName string) string {
	return filepath.Join(ConfigDir(cfgPath), ".workgroup", wgName, repoName)
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

	cfg, cfgPath, err := LoadCWD(override)
	if err != nil {
		return nil, "", err
	}

	for _, w := range cfg.Warnings {
		output.Writef(cmd.ErrOrStderr(), "%s\n", w)
	}

	return cfg, cfgPath, nil
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
			Path:     setCfg.Branches[branch],
			CloneURL: setCfg.URL,
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
