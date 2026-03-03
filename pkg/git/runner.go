package git

import (
	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/spf13/cobra"
)

func runGitCmd(cmd *cobra.Command, args []string, gitArgs ...string) error {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	repos, err := repo.Filter(cfg, cfgPath, args)
	if err != nil {
		return err
	}

	opts := ExecOptions{Async: len(repos) > 1}
	results := RunParallel(repos, gitArgs, opts)
	WriteResults(cmd.OutOrStdout(), results)

	return ExecErrors(results)
}
