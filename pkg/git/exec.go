package git

import (
	"fmt"

	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/spf13/cobra"
)

func registerExec(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:     "exec [repos...] -- <git-args>",
		Aliases: []string{"x", "run"},
		Short:   "Execute an arbitrary git command across repos",
		Long: `Runs any git command in each registered repo concurrently.
Repo names before '--' filter targets; everything after '--' is passed to git.

NOTE: Because flag parsing is disabled for this command (to preserve '--'),
the --config root flag is not available. Use the GIT_W_CONFIG
environment variable instead if you need to specify a custom config path.`,
		// DisableFlagParsing preserves "--" in args so splitExecArgs can split on it.
		DisableFlagParsing: true,
		RunE:               runExec,
	})
}

func runExec(cmd *cobra.Command, args []string) error {
	repoNames, gitArgs := splitExecArgs(args)

	if len(gitArgs) == 0 {
		return fmt.Errorf("no git command specified; use: exec [repos...] -- <git-args>")
	}

	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	repos, err := repo.Filter(cfg, cfgPath, repoNames)
	if err != nil {
		return err
	}

	// exec always runs async (with [name] prefix) regardless of repo count,
	// because the command is arbitrary and users need to know which repo produced which output.
	// This differs from named git subcommands (fetch/pull/push/status) which suppress the
	// prefix for single-repo targets.
	opts := ExecOptions{Async: true}
	results := RunParallel(repos, gitArgs, opts)
	WriteResults(cmd.OutOrStdout(), results)

	return ExecErrors(results)
}

func splitExecArgs(args []string) (repoNames, gitArgs []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}

	return nil, args
}
