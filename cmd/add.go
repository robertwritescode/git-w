package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/spf13/cobra"
)

var addGroup string

var addCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Register an existing local git repo",
	Long: `Registers a local git repository in the .gitworkspace config.
The repo name defaults to the base directory name.
Use -g/--group to also add the repo to a group.`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addGroup, "group", "g", "", "add repo to this group")
}

func runAdd(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(args[0])
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
