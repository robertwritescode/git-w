package repo

import (
	"fmt"
	"maps"
	"slices"

	"github.com/robertwritescode/git-w/pkg/workspace"
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
	cfg, cfgPath, err := workspace.LoadConfig(cmd)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		return printRepoPath(cmd, cfg, cfgPath, args[0])
	}

	return printRepoNames(cmd, cfg)
}

func printRepoPath(cmd *cobra.Command, cfg *workspace.WorkspaceConfig, cfgPath string, name string) error {
	r, exists := cfg.Repos[name]
	if !exists {
		return fmt.Errorf("repo %q not found", name)
	}

	absPath, err := workspace.ResolveRepoPath(cfgPath, r.Path)
	if err != nil {
		return fmt.Errorf("resolving path for repo %q: %w", name, err)
	}

	writef(cmd.OutOrStdout(), "%s\n", absPath)
	return nil
}

func printRepoNames(cmd *cobra.Command, cfg *workspace.WorkspaceConfig) error {
	for _, name := range slices.Sorted(maps.Keys(cfg.Repos)) {
		writef(cmd.OutOrStdout(), "%s\n", name)
	}

	return nil
}
