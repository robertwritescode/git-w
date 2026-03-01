package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func registerCompletion(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate a shell completion script",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE:      runCompletion,
	})
}

func runCompletion(cmd *cobra.Command, args []string) error {
	root := cmd.Root()
	switch args[0] {
	case "bash":
		return root.GenBashCompletion(cmd.OutOrStdout())
	case "zsh":
		return root.GenZshCompletion(cmd.OutOrStdout())
	case "fish":
		return root.GenFishCompletion(cmd.OutOrStdout(), true)
	case "powershell":
		return root.GenPowerShellCompletion(cmd.OutOrStdout())
	default:
		return fmt.Errorf("unsupported shell: %s", args[0])
	}
}
