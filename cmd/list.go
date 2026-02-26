package cmd

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list [name]",
	Aliases: []string{"ls"},
	Short:   "List repo names or print the path of a single repo",
	Long: `Without arguments, lists all registered repo names (sorted).
With a repo name argument, prints the absolute path to that repo.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	if len(args) == 1 {
		return printRepoPath(cmd, cfg, cfgPath, args[0])
	}

	return printRepoNames(cmd, cfg)
}

func printRepoPath(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath string, name string) error {
	repo, exists := cfg.Repos[name]
	if !exists {
		return fmt.Errorf("repo %q not found", name)
	}
	configDir := config.ConfigDir(cfgPath)
	fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(configDir, repo.Path))
	return nil
}

func printRepoNames(cmd *cobra.Command, cfg *config.WorkspaceConfig) error {
	names := make([]string, 0, len(cfg.Repos))
	for name := range cfg.Repos {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintln(cmd.OutOrStdout(), name)
	}
	return nil
}
