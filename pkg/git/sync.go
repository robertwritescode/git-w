package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/spf13/cobra"
)

type syncStep struct {
	Op       string
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Err      error
}

type syncReport struct {
	RepoName string
	Steps    []syncStep
	Failed   bool
}

type syncRepoFn func(repo.Repo, bool) syncReport

func registerSync(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "sync [repos...]",
		Aliases: []string{"s"},
		Short:   "Fetch, pull, and push all repos",
		RunE:    runSync,
	}
	cmd.Flags().Bool("push", false, "enable push (overrides config)")
	cmd.Flags().Bool("no-push", false, "skip push (overrides config)")
	root.AddCommand(cmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, repos, err := loadInputs(cmd, args)
	if err != nil {
		return err
	}

	doPush, err := resolveSyncPush(cmd, cfg)
	if err != nil {
		return err
	}

	reports := collectSyncReports(cmd, cfg, cfgPath, repos, doPush)
	writeSyncReports(cmd, reports)
	writeSyncSummary(cmd, reports)

	return syncReportsError(reports)
}

func resolveSyncPush(cmd *cobra.Command, cfg *workspace.WorkspaceConfig) (bool, error) {
	push, _ := cmd.Flags().GetBool("push")
	noPush, _ := cmd.Flags().GetBool("no-push")

	if push && noPush {
		return false, fmt.Errorf("--push and --no-push cannot be used together")
	}
	if push {
		return push, nil
	}
	if noPush {
		return !noPush, nil
	}

	return cfg.SyncPushEnabled(), nil
}

func collectSyncReports(cmd *cobra.Command, cfg *workspace.WorkspaceConfig, cfgPath string, repos []repo.Repo, doPush bool) []syncReport {
	plain, setRepos := splitSyncTargets(cfg, repos)
	reports := runSyncReports(plain, doPush, syncPlainRepo)
	reports = append(reports, runWorktreeSetSync(cmd, cfg, cfgPath, setRepos, doPush)...)
	return reports
}

func splitSyncTargets(cfg *workspace.WorkspaceConfig, repos []repo.Repo) ([]repo.Repo, map[string][]repo.Repo) {
	byRepo := worktreeRepoToSet(cfg)
	plain := make([]repo.Repo, 0, len(repos))
	setRepos := make(map[string][]repo.Repo)

	for _, r := range repos {
		setName, isWorktree := byRepo[r.Name]
		if !isWorktree {
			plain = append(plain, r)
			continue
		}

		setRepos[setName] = append(setRepos[setName], r)
	}

	return plain, setRepos
}

func runWorktreeSetSync(cmd *cobra.Command, cfg *workspace.WorkspaceConfig, cfgPath string, setRepos map[string][]repo.Repo, doPush bool) []syncReport {
	reports := make([]syncReport, 0)

	for _, setName := range workspace.SortedStringKeys(setRepos) {
		if err := fetchSetBare(cmd, cfgPath, setName, cfg.Worktrees[setName]); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "[%s] fetch error: %v\n", setName, err)
			reports = append(reports, failedSetReports(setRepos[setName])...)
			continue
		}

		reports = append(reports, runSyncReports(setRepos[setName], doPush, syncWorktreeRepo)...)
	}

	return reports
}

func failedSetReports(repos []repo.Repo) []syncReport {
	reports := make([]syncReport, 0, len(repos))
	for _, r := range repos {
		reports = append(reports, syncReport{RepoName: r.Name, Failed: true})
	}
	return reports
}

func runSyncReports(repos []repo.Repo, doPush bool, fn syncRepoFn) []syncReport {
	if len(repos) <= 1 {
		if len(repos) == 0 {
			return nil
		}
		return []syncReport{fn(repos[0], doPush)}
	}

	workers := parallel.MaxWorkers(0, len(repos))
	return parallel.RunFanOut(repos, workers, func(r repo.Repo) syncReport {
		return fn(r, doPush)
	})
}

func syncPlainRepo(r repo.Repo, doPush bool) syncReport {
	report := syncReport{RepoName: r.Name}
	if !runSyncStep(&report, r, "fetch") {
		return report
	}
	if !runSyncStep(&report, r, "pull") {
		return report
	}
	if doPush {
		runSyncStep(&report, r, "push")
	}
	return report
}

func syncWorktreeRepo(r repo.Repo, doPush bool) syncReport {
	report := syncReport{RepoName: r.Name}
	if !runSyncStep(&report, r, "pull") {
		return report
	}
	if doPush {
		runSyncStep(&report, r, "push")
	}
	return report
}

func runSyncStep(report *syncReport, r repo.Repo, op string) bool {
	result := runOne(context.Background(), r, []string{op})
	step := syncStep{Op: op, Stdout: result.Stdout, Stderr: result.Stderr, ExitCode: result.ExitCode, Err: result.Err}
	report.Steps = append(report.Steps, step)

	if !syncStepFailed(step) {
		return true
	}

	report.Failed = true
	return false
}

func writeSyncReports(cmd *cobra.Command, reports []syncReport) {
	for _, report := range reports {
		for _, step := range report.Steps {
			if syncStepFailed(step) {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "[%s] %s error: %s\n", report.RepoName, step.Op, syncStepMessage(step))
				continue
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", report.RepoName, step.Op)
		}
	}
}

func writeSyncSummary(cmd *cobra.Command, reports []syncReport) {
	failed := 0
	for _, report := range reports {
		if report.Failed {
			failed++
		}
	}

	ok := len(reports) - failed
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "sync complete: %d ok, %d failed\n", ok, failed)
}

func syncReportsError(reports []syncReport) error {
	failures := make([]string, 0)
	for _, report := range reports {
		if report.Failed {
			failures = append(failures, fmt.Sprintf("  [%s]: sync failed", report.RepoName))
		}
	}

	return parallel.FormatFailureError(failures, len(reports))
}

func syncStepFailed(step syncStep) bool {
	return step.Err != nil || step.ExitCode != 0
}

func syncStepMessage(step syncStep) string {
	text := strings.TrimSpace(string(append(step.Stderr, step.Stdout...)))
	if text != "" {
		return text
	}
	if step.Err != nil {
		return step.Err.Error()
	}
	return fmt.Sprintf("exit %d", step.ExitCode)
}
