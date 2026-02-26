package git

import (
	"github.com/spf13/cobra"
)

// registerGit adds all git execution commands to root.
func registerGit(root *cobra.Command) {
	root.AddCommand(
		&cobra.Command{
			Use:     "fetch [repos...]",
			Aliases: []string{"f"},
			Short:   "Run git fetch in repos",
			RunE:    runFetch,
		},
		&cobra.Command{
			Use:     "pull [repos...]",
			Aliases: []string{"pl"},
			Short:   "Run git pull in repos",
			RunE:    runPull,
		},
		&cobra.Command{
			Use:     "push [repos...]",
			Aliases: []string{"ps"},
			Short:   "Run git push in repos",
			RunE:    runPush,
		},
		&cobra.Command{
			Use:     "status [repos...]",
			Aliases: []string{"st"},
			Short:   "Run git status -sb in repos",
			RunE:    runStatus,
		},
	)
}

func runFetch(cmd *cobra.Command, args []string) error {
	return runGitCmd(cmd, args, "fetch")
}

func runPull(cmd *cobra.Command, args []string) error {
	return runGitCmd(cmd, args, "pull")
}

func runPush(cmd *cobra.Command, args []string) error {
	return runGitCmd(cmd, args, "push")
}

func runStatus(cmd *cobra.Command, args []string) error {
	return runGitCmd(cmd, args, "status", "-sb")
}
