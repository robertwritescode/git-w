package repo

import "github.com/spf13/cobra"

// Register adds all repo commands to root.
func Register(root *cobra.Command) {
	repoCmd := &cobra.Command{
		Use:     "repo",
		Aliases: []string{"r"},
		Short:   "Manage tracked repositories",
	}

	registerAdd(repoCmd)
	registerClone(repoCmd)
	registerUnlink(repoCmd)
	registerRename(repoCmd)
	registerList(repoCmd)

	root.AddCommand(repoCmd)

	registerRestore(root)
}
