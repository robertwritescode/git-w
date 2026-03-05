package workgroup

import "github.com/spf13/cobra"

// Register wires work subcommands into the root command.
func Register(root *cobra.Command) {
	workCmd := newWorkCmd()
	registerCreate(workCmd)
	registerCheckout(workCmd)
	registerList(workCmd)
	registerDrop(workCmd)
	registerPush(workCmd)
	registerPath(workCmd)
	registerAdd(workCmd)
	root.AddCommand(workCmd)
}

func newWorkCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "workgroup",
		Aliases: []string{"work", "wg"},
		Short:   "Manage local workgroups of git worktrees across repos",
	}
}
