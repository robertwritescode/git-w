package git

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/spf13/cobra"
)

type commitFlags struct {
	Message   string
	DryRun    bool
	NoVerify  bool
	Workgroup string
}

func registerCommit(root *cobra.Command) {
	root.AddCommand(newCommitCmd())
}

func newCommitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "commit [repos...]",
		Aliases: []string{"ci"},
		Short:   "Atomically commit staged changes across repos",
		RunE:    runCommit,
	}

	cmd.Flags().StringP("message", "m", "", "commit message (required)")
	cmd.Flags().Bool("dry-run", false, "show which repos would be committed without committing")
	cmd.Flags().Bool("no-verify", false, "skip pre-commit and commit-msg hooks")
	cmd.Flags().StringP("workgroup", "W", "", "scope commit to a workgroup's worktrees")
	_ = cmd.MarkFlagRequired("message")

	return cmd
}

func parseCommitFlags(cmd *cobra.Command) (commitFlags, error) {
	msg, err := cmd.Flags().GetString("message")
	if err != nil {
		return commitFlags{}, err
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return commitFlags{}, err
	}

	noVerify, err := cmd.Flags().GetBool("no-verify")
	if err != nil {
		return commitFlags{}, err
	}

	wg, err := cmd.Flags().GetString("workgroup")
	if err != nil {
		return commitFlags{}, err
	}

	return commitFlags{Message: msg, DryRun: dryRun, NoVerify: noVerify, Workgroup: wg}, nil
}

func runCommit(cmd *cobra.Command, args []string) error {
	flags, err := parseCommitFlags(cmd)
	if err != nil {
		return err
	}

	repos, err := resolveCommitRepos(cmd, args, flags)
	if err != nil {
		return err
	}

	staged, skipped := filterStaged(repos)

	return dispatchCommit(cmd.OutOrStdout(), staged, skipped, flags)
}

func dispatchCommit(w io.Writer, staged, skipped []repo.Repo, flags commitFlags) error {
	reportSkipped(w, skipped)

	if len(staged) == 0 {
		return errors.New("nothing to commit: no repos have staged changes")
	}

	if flags.DryRun {
		return reportDryRun(w, staged)
	}

	return executeCommit(w, staged, flags)
}

func resolveCommitRepos(cmd *cobra.Command, args []string, flags commitFlags) ([]repo.Repo, error) {
	if flags.Workgroup != "" && len(args) > 0 {
		return nil, errors.New("--workgroup and explicit repo targets are mutually exclusive")
	}

	if flags.Workgroup != "" {
		return resolveWorkgroupRepos(cmd, flags.Workgroup)
	}

	_, _, repos, err := loadInputs(cmd, args)

	return repos, err
}

func resolveWorkgroupRepos(cmd *cobra.Command, wgName string) ([]repo.Repo, error) {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	wg, ok := cfg.Workgroups[wgName]
	if !ok {
		return nil, fmt.Errorf("workgroup %q not found", wgName)
	}

	return workgroupWorktreeRepos(cfgPath, wgName, wg), nil
}

func workgroupWorktreeRepos(cfgPath, wgName string, wg config.WorkgroupConfig) []repo.Repo {
	repos := make([]repo.Repo, 0)

	for _, name := range wg.Repos {
		path := config.WorkgroupWorktreePath(cfgPath, wgName, name)
		if _, err := os.Stat(path); err != nil && errors.Is(err, os.ErrNotExist) {
			continue
		}

		repos = append(repos, repo.Repo{Name: name, AbsPath: path})
	}

	return repos
}

func filterStaged(repos []repo.Repo) (staged, skipped []repo.Repo) {
	results := RunParallel(repos, []string{"diff", "--cached", "--quiet"}, ExecOptions{Async: true})

	for i, r := range results {
		if r.ExitCode == 1 {
			staged = append(staged, repos[i])
		} else {
			skipped = append(skipped, repos[i])
		}
	}

	return staged, skipped
}

func executeCommit(w io.Writer, staged []repo.Repo, flags commitFlags) error {
	results := RunParallel(staged, buildCommitArgs(flags), ExecOptions{Async: true})

	if !anyFailed(results) {
		WriteResults(w, results)
		return nil
	}

	rollbackResults := rollback(succeededRepos(staged, results))
	reportFailure(w, results, rollbackResults)

	return ExecErrors(results)
}

func buildCommitArgs(flags commitFlags) []string {
	args := []string{"commit", "-m", flags.Message}

	if flags.NoVerify {
		args = append(args, "--no-verify")
	}

	return args
}

func anyFailed(results []ExecResult) bool {
	for _, r := range results {
		if r.ExitCode != 0 || r.Err != nil {
			return true
		}
	}

	return false
}

func succeededRepos(repos []repo.Repo, results []ExecResult) []repo.Repo {
	var out []repo.Repo

	for i, r := range results {
		if r.ExitCode == 0 && r.Err == nil {
			out = append(out, repos[i])
		}
	}

	return out
}

func rollback(repos []repo.Repo) []ExecResult {
	return RunParallel(repos, []string{"reset", "--soft", "HEAD~1"}, ExecOptions{Async: true})
}

func reportSkipped(w io.Writer, skipped []repo.Repo) {
	for _, r := range skipped {
		output.Writef(w, "[%s] skipped: no staged changes\n", r.Name)
	}
}

func reportDryRun(w io.Writer, staged []repo.Repo) error {
	output.Writef(w, "dry run — would commit %d repo(s):\n", len(staged))

	for _, r := range staged {
		output.Writef(w, "  %s\n", r.Name)
	}

	return nil
}

func reportFailure(w io.Writer, commitResults, rollbackResults []ExecResult) {
	output.Writef(w, "commit failed — rolling back\n\n")

	reportCommitFailures(w, commitResults)
	output.Writef(w, "\n")
	reportRollbackResults(w, rollbackResults)
}

// formatExecError normalizes stderr for display, removing a leading
// "[repoName] " prefix if present to avoid double-prefixing when
// combined with our own "[%s] failed: ..." formatting.
func formatExecError(repoName string, stderr []byte) string {
	msg := strings.TrimSpace(string(stderr))
	if msg == "" {
		return msg
	}

	prefix := fmt.Sprintf("[%s] ", repoName)

	return strings.TrimPrefix(msg, prefix)
}

func reportCommitFailures(w io.Writer, results []ExecResult) {
	for _, r := range results {
		if r.ExitCode != 0 || r.Err != nil {
			msg := formatExecError(r.RepoName, r.Stderr)
			output.Writef(w, "[%s] failed: %s\n", r.RepoName, msg)
		}
	}
}

func reportRollbackResults(w io.Writer, results []ExecResult) {
	for _, r := range results {
		if r.ExitCode != 0 || r.Err != nil {
			output.Writef(w, "[%s] rollback FAILED — run manually: git reset --soft HEAD~1\n", r.RepoName)
		} else {
			output.Writef(w, "[%s] rolled back — staged changes restored\n", r.RepoName)
		}
	}
}
