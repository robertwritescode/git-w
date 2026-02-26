package cmd

import (
	"fmt"
	"os"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "git-workspace",
	Short: "Manage multiple git repos defined in .gitworkspace",
	Long: `git-workspace is a Git plugin to manage and run commands across multiple repositories from a single workspace config.
Invoke as 'git workspace <cmd>' via git's plugin system (git-workspace must be in $PATH).`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to .gitworkspace config (default: nearest .gitworkspace found by walking up from CWD)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadConfig() (*config.WorkspaceConfig, string, error) {
	path := cfgFile
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", fmt.Errorf("getting working directory: %w", err)
		}
		path, err = config.Discover(cwd)
		if err != nil {
			return nil, "", err
		}
	}

	cfg, err := config.Load(path)
	if err != nil {
		return nil, "", err
	}
	return cfg, path, nil
}

func autoGitignoreEnabled(cfg *config.WorkspaceConfig) bool {
	return cfg.Workspace.AutoGitignore == nil || *cfg.Workspace.AutoGitignore
}
