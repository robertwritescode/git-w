package repo

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/spf13/cobra"
)

func registerAdd(root *cobra.Command) {
	addCmd := &cobra.Command{
		Use:     "add [<path>]",
		Aliases: []string{"register"},
		Short:   "Register an existing local git repo",
		Long: `Registers a local git repository in the .gitw config.
The repo name defaults to the base directory name.
Use -g/--group to also add the repo to a group.
Use -r to recursively find and register all git repos under a directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runAdd,
	}
	addCmd.Flags().StringP("group", "g", "", "add repo to this group")
	addCmd.Flags().BoolP("recursive", "r", false, "recursively add all git repos in <dir> (or CWD if no dir given)")
	root.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := workspace.LoadConfig(cmd)
	if err != nil {
		return err
	}

	recursive, _ := cmd.Flags().GetBool("recursive")
	group, _ := cmd.Flags().GetString("group")

	if recursive {
		var dirArg string
		if len(args) == 1 {
			dirArg = args[0]
		}
		return runAddRecursive(cmd, cfg, cfgPath, dirArg, group)
	}

	if len(args) == 0 {
		return fmt.Errorf("path argument required when not using --recursive")
	}

	return runAddSingle(cmd, cfg, cfgPath, args[0], group)
}

func runAddSingle(cmd *cobra.Command, cfg *workspace.WorkspaceConfig, cfgPath, pathArg, group string) error {
	absPath, err := filepath.Abs(pathArg)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if !IsGitRepo(absPath) {
		return fmt.Errorf("%s is not a git repository", absPath)
	}

	name, err := cfg.RepoName(absPath)
	if err != nil {
		return err
	}

	relPath, err := workspace.RelPath(cfgPath, absPath)
	if err != nil {
		return err
	}

	cfg.Repos[name] = workspace.RepoConfig{
		Path: relPath,
		URL:  gitutil.RemoteURL(absPath),
	}

	applyMeta(cmd, cfg, cfgPath, relPath, name, group)

	if err := workspace.Save(cfgPath, cfg); err != nil {
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Added repo %q (%s)\n", name, relPath)
	return nil
}

func runAddRecursive(cmd *cobra.Command, cfg *workspace.WorkspaceConfig, cfgPath, dirArg, group string) error {
	walkDir, err := resolveWalkDir(dirArg)
	if err != nil {
		return err
	}

	paths, err := walkGitRepos(walkDir)
	if err != nil {
		return err
	}

	count := registerDiscoveredRepos(cmd, cfg, cfgPath, paths, walkDir, group)

	if err := workspace.Save(cfgPath, cfg); err != nil {
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Added %d repo(s)\n", count)
	return nil
}

func registerDiscoveredRepos(cmd *cobra.Command, cfg *workspace.WorkspaceConfig, cfgPath string, paths []string, walkDir, group string) int {
	count := 0
	for _, p := range paths {
		groupName := effectiveGroupName(group, p, walkDir)
		ok, err := registerSingleRepo(cmd, cfg, cfgPath, p, groupName)
		if err != nil {
			output.Writef(cmd.ErrOrStderr(), "warning: skipping %s: %v\n", p, err)
			continue
		}

		if ok {
			count++
		}
	}
	return count
}

func effectiveGroupName(group, repoPath, walkDir string) string {
	if group != "" {
		return group
	}

	return autoGroupName(repoPath, walkDir)
}

func resolveWalkDir(dir string) (string, error) {
	if dir == "" {
		return os.Getwd()
	}

	return filepath.Abs(dir)
}

func walkGitRepos(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return visitDir(root, path, d, &paths)
	})

	return paths, err
}

func visitDir(root, path string, d fs.DirEntry, paths *[]string) error {
	if !d.IsDir() {
		return nil
	}

	if isHiddenDir(root, path, d) {
		return fs.SkipDir
	}

	if IsGitRepo(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		*paths = append(*paths, abs)
		return fs.SkipDir
	}
	return nil
}

func isHiddenDir(root, path string, d fs.DirEntry) bool {
	return path != root && strings.HasPrefix(d.Name(), ".")
}

func autoGroupName(repoAbsPath, walkRoot string) string {
	rel, err := filepath.Rel(walkRoot, repoAbsPath)
	if err != nil || rel == "." {
		return ""
	}

	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) <= 1 {
		return ""
	}

	return parts[0]
}

func registerSingleRepo(cmd *cobra.Command, cfg *workspace.WorkspaceConfig, cfgPath, absPath, groupName string) (bool, error) {
	if !IsGitRepo(absPath) {
		return false, nil
	}

	name, err := cfg.RepoName(absPath)
	if err != nil {
		return false, nil // already registered, skip silently
	}

	relPath, err := workspace.RelPath(cfgPath, absPath)
	if err != nil {
		return false, err // real error, propagate
	}

	cfg.Repos[name] = workspace.RepoConfig{Path: relPath, URL: gitutil.RemoteURL(absPath)}

	applyMeta(cmd, cfg, cfgPath, relPath, name, groupName)

	return true, nil
}

func applyMeta(cmd *cobra.Command, cfg *workspace.WorkspaceConfig, cfgPath, relPath, repoName, groupName string) {
	if cfg.AutoGitignoreEnabled() {
		if err := gitutil.EnsureGitignore(workspace.ConfigDir(cfgPath), relPath); err != nil {
			output.Writef(cmd.ErrOrStderr(), "warning: could not update .gitignore: %v\n", err)
		}
	}

	if groupName != "" {
		cfg.AddRepoToGroup(groupName, repoName)
	}
}
