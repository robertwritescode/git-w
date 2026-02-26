package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/spf13/cobra"
)

var addGroup string
var addRecursive bool

var addCmd = &cobra.Command{
	Use:   "add [<path>]",
	Short: "Register an existing local git repo",
	Long: `Registers a local git repository in the .gitworkspace config.
The repo name defaults to the base directory name.
Use -g/--group to also add the repo to a group.
Use -r to recursively find and register all git repos under a directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addGroup, "group", "g", "", "add repo to this group")
	addCmd.Flags().BoolVarP(&addRecursive, "recursive", "r", false, "recursively add all git repos in <dir> (or CWD if no dir given)")
}

func runAdd(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	if addRecursive {
		var dirArg string
		if len(args) == 1 {
			dirArg = args[0]
		}
		return runAddRecursive(cmd, cfg, cfgPath, dirArg)
	}

	if len(args) == 0 {
		return fmt.Errorf("accepts 1 arg(s), received 0")
	}

	return runAddSingle(cmd, cfg, cfgPath, args[0])
}

func runAddSingle(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath, pathArg string) error {
	absPath, err := filepath.Abs(pathArg)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if !isGitRepo(absPath) {
		return fmt.Errorf("%s is not a git repository", absPath)
	}

	name, err := resolveRepoName(cfg, absPath)
	if err != nil {
		return err
	}

	relPath, err := computeRelPath(cfgPath, absPath)
	if err != nil {
		return err
	}

	cfg.Repos[name] = config.RepoConfig{
		Path: relPath,
		URL:  detectRemoteURL(absPath),
	}

	if autoGitignoreEnabled(cfg) {
		if err := ensureGitignore(config.ConfigDir(cfgPath), relPath); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not update .gitignore: %v\n", err)
		}
	}

	if addGroup != "" {
		addRepoToGroup(cfg, addGroup, name)
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added repo %q (%s)\n", name, relPath)
	return nil
}

func runAddRecursive(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath, dirArg string) error {
	walkDir, err := resolveWalkDir(dirArg)
	if err != nil {
		return err
	}

	paths, err := walkGitRepos(walkDir)
	if err != nil {
		return err
	}

	count := 0
	for _, p := range paths {
		if registerSingleRepo(cmd, cfg, cfgPath, p, autoGroupName(p, walkDir)) {
			count++
		}
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added %d repo(s)\n", count)
	return nil
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

		if !d.IsDir() {
			return nil
		}

		if path != root && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}

		if isGitRepo(path) {
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}

			paths = append(paths, abs)
			return fs.SkipDir
		}

		return nil
	})
	return paths, err
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

func registerSingleRepo(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath, absPath, groupName string) bool {
	if !isGitRepo(absPath) {
		return false
	}

	name := filepath.Base(absPath)
	if _, exists := cfg.Repos[name]; exists {
		return false
	}

	relPath, err := computeRelPath(cfgPath, absPath)
	if err != nil {
		return false
	}

	cfg.Repos[name] = config.RepoConfig{Path: relPath, URL: detectRemoteURL(absPath)}

	if autoGitignoreEnabled(cfg) {
		if err := ensureGitignore(config.ConfigDir(cfgPath), relPath); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not update .gitignore: %v\n", err)
		}
	}

	if groupName != "" {
		addRepoToGroup(cfg, groupName, name)
	}

	return true
}

func resolveRepoName(cfg *config.WorkspaceConfig, absPath string) (string, error) {
	name := filepath.Base(absPath)
	if _, exists := cfg.Repos[name]; exists {
		return "", fmt.Errorf("repo %q is already registered", name)
	}
	return name, nil
}

func computeRelPath(cfgPath, absPath string) (string, error) {
	relPath, err := filepath.Rel(config.ConfigDir(cfgPath), absPath)
	if err != nil {
		return "", fmt.Errorf("computing relative path: %w", err)
	}
	return relPath, nil
}

func addRepoToGroup(cfg *config.WorkspaceConfig, group, name string) {
	g := cfg.Groups[group]
	g.Repos = append(g.Repos, name)
	cfg.Groups[group] = g
}

func isGitRepo(path string) bool {
	// Use Open+Close instead of Stat to avoid a redundant syscall on success.
	f, err := os.Open(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func detectRemoteURL(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
