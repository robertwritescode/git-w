package repo

import (
	"fmt"

	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/spf13/cobra"
)

func registerList(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:     "list [name]",
		Aliases: []string{"ls"},
		Short:   "List repo names or print the path of a single repo",
		Long: `Without arguments, lists all registered repo names (sorted).
With a repo name argument, prints the absolute path to that repo.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runList,
	})
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		return printRepoPath(cmd, cfg, cfgPath, args[0])
	}

	return printRepoNames(cmd, cfg)
}

func printRepoPath(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath string, name string) error {
	r, exists := cfg.Repos[name]
	if !exists {
		return fmt.Errorf("repo %q not found", name)
	}

	absPath, err := config.ResolveRepoPath(cfgPath, r.Path)
	if err != nil {
		return fmt.Errorf("resolving path for repo %q: %w", name, err)
	}

	output.Writef(cmd.OutOrStdout(), "%s\n", absPath)
	return nil
}

func printRepoNames(cmd *cobra.Command, cfg *config.WorkspaceConfig) error {
	for _, name := range config.SortedStringKeys(cfg.Repos) {
		output.Writef(cmd.OutOrStdout(), "%s\n", name)
	}

	return nil
}
