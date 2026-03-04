package branch

import "github.com/spf13/cobra"

// Register wires branch subcommands into the root command.
func Register(root *cobra.Command) {
	branchCmd := newBranchCmd()
	registerCreate(branchCmd)
	registerDefault(branchCmd)
	root.AddCommand(branchCmd)
}

func newBranchCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "branch",
		Aliases: []string{"b"},
		Short:   "Manage branches across repos",
	}
}
