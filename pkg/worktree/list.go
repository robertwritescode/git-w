package worktree

import (
	"fmt"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/spf13/cobra"
)

func registerList(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:     "list [set-name]",
		Aliases: []string{"ls"},
		Short:   "List worktree sets or branches in a set",
		Args:    cobra.MaximumNArgs(1),
		RunE:    runList,
	})
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, _, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return listWorktreeSets(cmd, cfg)
	}

	return listWorktreeBranches(cmd, cfg, args[0])
}

func listWorktreeSets(cmd *cobra.Command, cfg *config.WorkspaceConfig) error {
	for _, name := range config.SortedStringKeys(cfg.Worktrees) {
		output.Writef(cmd.OutOrStdout(), "%s\n", name)
	}

	return nil
}

func listWorktreeBranches(cmd *cobra.Command, cfg *config.WorkspaceConfig, setName string) error {
	wt, exists := cfg.Worktrees[setName]
	if !exists {
		return fmt.Errorf("worktree set %q not found", setName)
	}

	for _, branch := range config.SortedWorktreeBranchNames(wt.Branches) {
		output.Writef(cmd.OutOrStdout(), "%s\t%s\n", branch, wt.Branches[branch])
	}

	return nil
}
