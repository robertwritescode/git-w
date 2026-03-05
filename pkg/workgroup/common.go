package workgroup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/robertwritescode/git-w/pkg/cmdutil"
	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/spf13/cobra"
)

type workStep struct {
	name    string
	err     error
	skipped bool
	detail  string
}

type workReport struct {
	RepoName string
	Steps    []workStep
	Failed   bool
}

type workFlags struct {
	SyncSource  bool
	SetUpstream bool
	Push        bool
	Checkout    bool
}

type checkoutWorkFlags struct {
	workFlags
	Pull bool
}

func runStep(report *workReport, stepName string, fn func() error) bool {
	err := fn()
	recordStep(report, stepName, err, false)
	return err == nil
}

func recordStep(report *workReport, stepName string, err error, skipped bool) {
	report.Steps = append(report.Steps, workStep{name: stepName, err: err, skipped: skipped})
	if err != nil {
		report.Failed = true
	}
}

func skipStep(report *workReport, stepName, detail string) {
	report.Steps = append(report.Steps, workStep{name: stepName, skipped: true, detail: detail})
}

func worktreePath(cfgPath, wgName, repoName string) string {
	return filepath.Join(config.ConfigDir(cfgPath), ".workgroup", wgName, repoName)
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func parseWorkArgs(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("workgroup name is required")
	}

	return args[0], args[1:], nil
}

func loadWorkInputs(cmd *cobra.Command, targets []string) (*config.WorkspaceConfig, string, []repo.Repo, error) {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return nil, "", nil, err
	}

	repos, err := repo.Filter(cfg, cfgPath, targets)
	if err != nil {
		return nil, "", nil, err
	}

	if len(repos) == 0 {
		return nil, "", nil, fmt.Errorf("no repos matched")
	}

	return cfg, cfgPath, repos, nil
}

func resolveWorkFlags(cmd *cobra.Command, cfg *config.WorkspaceConfig) (workFlags, error) {
	push, err := cmdutil.ResolveBoolFlag(cmd, "push", "no-push", cfg.BranchPushEnabled())
	if err != nil {
		return workFlags{}, err
	}

	setUpstream, err := cmdutil.ResolveBoolFlag(cmd, "allow-upstream", "no-upstream", cfg.BranchSetUpstreamEnabled())
	if err != nil {
		return workFlags{}, err
	}

	checkout, _ := cmd.Flags().GetBool("checkout")

	return workFlags{Push: push, SetUpstream: setUpstream, Checkout: checkout}, nil
}

func resolveCheckoutWorkFlags(cmd *cobra.Command, cfg *config.WorkspaceConfig) (checkoutWorkFlags, error) {
	flags, err := resolveWorkFlags(cmd, cfg)
	if err != nil {
		return checkoutWorkFlags{}, err
	}

	pull, _ := cmd.Flags().GetBool("pull")

	return checkoutWorkFlags{workFlags: flags, Pull: pull}, nil
}

func applyWorkRemoteOps(ctx context.Context, report *workReport, r repo.Repo, branchName string, flags workFlags) {
	if !gitutil.HasRemote(ctx, r.AbsPath) {
		skipRemoteOps(report, flags)
		return
	}

	if flags.Push {
		runStep(report, "push", func() error {
			return gitutil.PushBranchUpstream(ctx, r.AbsPath, "origin", branchName)
		})

		return
	}

	if flags.SetUpstream {
		runStep(report, "upstream", func() error {
			return gitutil.SetBranchUpstream(ctx, r.AbsPath, branchName, "origin")
		})
	}
}

func skipRemoteOps(report *workReport, flags workFlags) {
	if flags.Push {
		skipStep(report, "push", "no remote")
	} else if flags.SetUpstream {
		skipStep(report, "upstream", "no remote")
	}
}

func attachRemoteBranch(ctx context.Context, report *workReport, r repo.Repo, treePath, branchName string) {
	if !runStep(report, "fetch", func() error {
		return gitutil.FetchOrigin(ctx, r.AbsPath)
	}) {
		return
	}

	runStep(report, "worktree", func() error {
		return gitutil.AddWorktree(ctx, r.AbsPath, treePath, branchName)
	})
}

func optionalPullWorktree(ctx context.Context, report *workReport, r repo.Repo, treePath, branchName string, pull bool) {
	if !pull {
		return
	}

	if !gitutil.HasRemote(ctx, r.AbsPath) {
		skipStep(report, "pull", "no remote")
		return
	}

	runStep(report, "pull", func() error {
		return gitutil.PullBranch(ctx, treePath, branchName)
	})
}

func resolveExistingTreePath(ctx context.Context, report *workReport, r repo.Repo, treePath, wgName string, pull bool) {
	cur, err := gitutil.CurrentBranch(ctx, treePath)
	if err != nil {
		recordStep(report, "worktree", fmt.Errorf("path exists but not a worktree: %w", err), false)
		return
	}

	if cur != wgName {
		recordStep(report, "worktree", fmt.Errorf("path %s on branch %s, expected %s", treePath, cur, wgName), false)
		return
	}

	skipStep(report, "worktree", "already exists")
	optionalPullWorktree(ctx, report, r, treePath, wgName, pull)
}

func successRepoNames(reports []workReport, repos []repo.Repo) []string {
	failed := make(map[string]bool, len(reports))
	for _, r := range reports {
		if r.Failed {
			failed[r.RepoName] = true
		}
	}

	names := make([]string, 0, len(repos))
	for _, r := range repos {
		if !failed[r.Name] {
			names = append(names, r.Name)
		}
	}

	return names
}

func writeWorkReports(cmd *cobra.Command, reports []workReport) {
	for _, report := range reports {
		for _, step := range report.Steps {
			writeWorkStep(cmd, report.RepoName, step)
		}
	}
}

func writeWorkStep(cmd *cobra.Command, repoName string, step workStep) {
	if step.skipped {
		msg := step.detail
		if msg == "" {
			msg = "skipped"
		}
		output.Writef(cmd.OutOrStdout(), "[%s] %s: %s, skipped\n", repoName, step.name, msg)
		return
	}

	if step.err != nil {
		output.Writef(cmd.ErrOrStderr(), "[%s] %s error: %v\n", repoName, step.name, step.err)
		return
	}

	output.Writef(cmd.OutOrStdout(), "[%s] %s\n", repoName, step.name)
}

func writeSummary(cmd *cobra.Command, reports []workReport, opName string) {
	ok, failed := countWorkReports(reports)
	output.Writef(cmd.OutOrStdout(), "%s complete: %d ok, %d failed\n", opName, ok, failed)
}

func countWorkReports(reports []workReport) (int, int) {
	failed := 0
	for _, r := range reports {
		if r.Failed {
			failed++
		}
	}

	return len(reports) - failed, failed
}

func workReportsError(reports []workReport, opName string) error {
	failures := make([]string, 0)
	for _, r := range reports {
		if r.Failed {
			failures = append(failures, fmt.Sprintf("  [%s]: %s failed", r.RepoName, opName))
		}
	}

	return parallel.FormatFailureError(failures, len(reports))
}
