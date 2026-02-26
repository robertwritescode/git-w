package cmd

import (
	"github.com/robertwritescode/git-workspace/internal/executor"
	"github.com/spf13/cobra"
)

var (
	fetchCmd  = &cobra.Command{Use: "fetch [repos...]", Short: "Run git fetch in repos", RunE: runFetch}
	pullCmd   = &cobra.Command{Use: "pull [repos...]", Short: "Run git pull in repos", RunE: runPull}
	pushCmd   = &cobra.Command{Use: "push [repos...]", Short: "Run git push in repos", RunE: runPush}
	statusCmd = &cobra.Command{Use: "status [repos...]", Aliases: []string{"st"}, Short: "Run git status -sb in repos", RunE: runStatus}
)

func init() {
	rootCmd.AddCommand(fetchCmd, pullCmd, pushCmd, statusCmd)
}

func runFetch(cmd *cobra.Command, args []string) error  { return runGitCmd(cmd, args, "fetch") }
func runPull(cmd *cobra.Command, args []string) error   { return runGitCmd(cmd, args, "pull") }
func runPush(cmd *cobra.Command, args []string) error   { return runGitCmd(cmd, args, "push") }
func runStatus(cmd *cobra.Command, args []string) error { return runGitCmd(cmd, args, "status", "-sb") }

func runGitCmd(cmd *cobra.Command, args []string, gitArgs ...string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	repos, err := filterRepos(cfg, cfgPath, args)
	if err != nil {
		return err
	}

	opts := executor.ExecOptions{Async: len(repos) > 1}
	results := executor.RunParallel(repos, gitArgs, opts)
	writeResults(cmd.OutOrStdout(), results)

	return execErrors(results)
}
