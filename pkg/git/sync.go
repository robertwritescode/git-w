package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/repo"
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

type syncUnit struct {
	isWorktree bool
	plain      *repo.Repo
	setName    string
	setRepos   []repo.Repo
	setConfig  config.WorktreeConfig
}

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

func resolveSyncPush(cmd *cobra.Command, cfg *config.WorkspaceConfig) (bool, error) {
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

func collectSyncReports(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath string, repos []repo.Repo, doPush bool) []syncReport {
	units := buildSyncUnits(cfg, repos)

	if len(units) == 0 {
		return nil
	}

	if len(units) == 1 {
		return executeSyncUnit(cmd, cfgPath, units[0], doPush)
	}

	workers := parallel.MaxWorkers(0, len(units))
	allReports := parallel.RunFanOut(units, workers, func(unit syncUnit) []syncReport {
		return executeSyncUnit(cmd, cfgPath, unit, doPush)
	})

	// Flatten the reports
	reports := make([]syncReport, 0)
	for _, unitReports := range allReports {
		reports = append(reports, unitReports...)
	}

	return reports
}

func buildSyncUnits(cfg *config.WorkspaceConfig, repos []repo.Repo) []syncUnit {
	byRepo := worktreeRepoToSet(cfg)
	units := make([]syncUnit, 0)
	setRepos := make(map[string][]repo.Repo)

	// Separate plain repos and worktree repos
	for _, r := range repos {
		setName, isWorktree := byRepo[r.Name]
		if !isWorktree {
			// Each plain repo is its own unit
			rCopy := r
			units = append(units, syncUnit{
				isWorktree: false,
				plain:      &rCopy,
			})
			continue
		}

		// Group worktree repos by set
		setRepos[setName] = append(setRepos[setName], r)
	}

	// Each worktree set becomes a unit
	for _, setName := range config.SortedStringKeys(setRepos) {
		units = append(units, syncUnit{
			isWorktree: true,
			setName:    setName,
			setRepos:   setRepos[setName],
			setConfig:  cfg.Worktrees[setName],
		})
	}

	return units
}

func executeSyncUnit(cmd *cobra.Command, cfgPath string, unit syncUnit, doPush bool) []syncReport {
	if !unit.isWorktree {
		// Plain repo: just sync it
		return []syncReport{syncPlainRepo(*unit.plain, doPush)}
	}

	// Worktree set: fetch the bare repo once, then sync all worktrees
	if err := fetchSetBare(cmd, cfgPath, unit.setName, unit.setConfig); err != nil {
		output.Writef(cmd.ErrOrStderr(), "[%s] fetch error: %v\n", unit.setName, err)
		return failedSetReports(unit.setRepos)
	}

	return runSyncReports(unit.setRepos, doPush, syncWorktreeRepo)
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
				output.Writef(cmd.ErrOrStderr(), "[%s] %s error: %s\n", report.RepoName, step.Op, syncStepMessage(step))
				continue
			}

			output.Writef(cmd.OutOrStdout(), "[%s] %s\n", report.RepoName, step.Op)
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
	output.Writef(cmd.OutOrStdout(), "sync complete: %d ok, %d failed\n", ok, failed)
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
	combined := make([]byte, 0, len(step.Stderr)+len(step.Stdout))
	combined = append(combined, step.Stderr...)
	combined = append(combined, step.Stdout...)
	text := strings.TrimSpace(string(combined))
	if text != "" {
		return text
	}
	if step.Err != nil {
		return step.Err.Error()
	}
	return fmt.Sprintf("exit %d", step.ExitCode)
}
