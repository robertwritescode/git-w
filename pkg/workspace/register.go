package workspace

import "github.com/spf13/cobra"

// Register adds all workspace commands to root.
func Register(root *cobra.Command) {
	registerInit(root)
	registerContext(root)
	registerGroup(root)
}
