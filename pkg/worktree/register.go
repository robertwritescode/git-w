package worktree

import "github.com/spf13/cobra"

func Register(root *cobra.Command) {
	worktreeCmd := &cobra.Command{
		Use:     "worktree",
		Aliases: []string{"tree", "t"},
		Short:   "Manage git worktree sets",
	}

	registerClone(worktreeCmd)
	registerAdd(worktreeCmd)
	registerRm(worktreeCmd)
	registerDrop(worktreeCmd)
	registerList(worktreeCmd)

	root.AddCommand(worktreeCmd)
}
