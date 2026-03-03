package branch

import "github.com/spf13/cobra"

// Register wires branch subcommands into the root command.
func Register(root *cobra.Command) {
	registerCreate(root)
}
