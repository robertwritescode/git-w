package git

import "github.com/spf13/cobra"

// Register adds all git commands to root.
func Register(root *cobra.Command) {
	registerGit(root)
	registerExec(root)
	registerInfo(root)
}
