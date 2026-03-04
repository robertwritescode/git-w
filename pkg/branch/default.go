package branch

import (
	"context"
	"fmt"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/spf13/cobra"
)

type defaultUnit struct {
	isWorktree   bool
	plain        *repo.Repo
	targetBranch string
	setName      string
	setRepos     []repo.Repo
	setConfig    config.WorktreeConfig
}

func registerDefault(branchCmd *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "default [repos...]",
		Aliases: []string{"d"},
		Short:   "Switch each repo to its default branch",
		RunE:    runDefault,
	}
	cmd.Flags().Bool("pull", false, "pull after switching to default branch")
	branchCmd.AddCommand(cmd)
}

func runDefault(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	repos, err := repo.Filter(cfg, cfgPath, args)
	if err != nil {
		return err
	}

	pull, _ := cmd.Flags().GetBool("pull")
	reports := collectDefaultReports(ctx, cmd, cfgPath, repos, cfg, pull)
	writeBranchReports(cmd, reports)
	writeSummary(cmd, reports, "branch default")

	return branchReportsError(reports, "branch default")
}

func collectDefaultReports(ctx context.Context, cmd *cobra.Command, cfgPath string, repos []repo.Repo, cfg *config.WorkspaceConfig, pull bool) []branchReport {
	units := buildDefaultUnits(cfg, repos)

	if len(units) == 0 {
		return nil
	}

	if len(units) == 1 {
		return executeDefaultUnit(ctx, cmd, cfgPath, cfg, units[0], pull)
	}

	workers := parallel.MaxWorkers(0, len(units))
	allReports := parallel.RunFanOut(units, workers, func(unit defaultUnit) []branchReport {
		return executeDefaultUnit(ctx, cmd, cfgPath, cfg, unit, pull)
	})

	return flattenBranchReports(allReports)
}

func buildDefaultUnits(cfg *config.WorkspaceConfig, repos []repo.Repo) []defaultUnit {
	plain, sets := classifyRepos(cfg, repos)

	units := plainDefaultUnits(cfg, plain)
	units = append(units, worktreeDefaultUnits(cfg, sets)...)

	return units
}

func plainDefaultUnits(cfg *config.WorkspaceConfig, repos []repo.Repo) []defaultUnit {
	units := make([]defaultUnit, 0, len(repos))

	for _, r := range repos {
		rCopy := r
		units = append(units, defaultUnit{plain: &rCopy, targetBranch: cfg.ResolveDefaultBranch(r.Name)})
	}

	return units
}

func worktreeDefaultUnits(cfg *config.WorkspaceConfig, sets map[string][]repo.Repo) []defaultUnit {
	units := make([]defaultUnit, 0, len(sets))

	for _, setName := range config.SortedStringKeys(sets) {
		units = append(units, defaultUnit{
			isWorktree: true,
			setName:    setName,
			setRepos:   sets[setName],
			setConfig:  cfg.Worktrees[setName],
		})
	}

	return units
}

func executeDefaultUnit(ctx context.Context, cmd *cobra.Command, cfgPath string, cfg *config.WorkspaceConfig, unit defaultUnit, pull bool) []branchReport {
	if !unit.isWorktree {
		return []branchReport{switchInPlainRepo(ctx, *unit.plain, unit.targetBranch, pull)}
	}

	return switchInWorktreeSet(ctx, cmd, cfgPath, cfg, unit, pull)
}

func switchInPlainRepo(ctx context.Context, r repo.Repo, targetBranch string, pull bool) branchReport {
	report := branchReport{RepoName: r.Name}

	if !checkoutDefault(ctx, &report, r, targetBranch) {
		return report
	}

	if !pull {
		return report
	}

	if !hasRemote(ctx, r) {
		skipNoRemote(&report, "pull")
		return report
	}

	pullSoft(ctx, &report, r.AbsPath, targetBranch)
	return report
}

func checkoutDefault(ctx context.Context, report *branchReport, r repo.Repo, branch string) bool {
	cur, err := gitutil.CurrentBranch(ctx, r.AbsPath)
	if err != nil {
		return runStep(report, "checkout", func() error { return err })
	}

	if cur == branch {
		report.Steps = append(report.Steps, branchStep{name: "checkout", skipped: true, detail: fmt.Sprintf("already on %s", branch)})
		return true
	}

	return runStep(report, "checkout", func() error {
		return gitutil.CheckoutBranch(ctx, r.AbsPath, branch)
	})
}

func pullSoft(ctx context.Context, report *branchReport, repoPath, branch string) {
	err := gitutil.PullBranch(ctx, repoPath, branch)
	report.Steps = append(report.Steps, branchStep{name: "pull", err: err})
}

func switchInWorktreeSet(ctx context.Context, cmd *cobra.Command, cfgPath string, cfg *config.WorkspaceConfig, unit defaultUnit, pull bool) []branchReport {
	if pull {
		if err := fetchBareRepo(ctx, cfgPath, unit.setConfig); err != nil {
			output.Writef(cmd.ErrOrStderr(), "[%s] fetch error: %v\n", unit.setName, err)
			return failedSetReports(unit.setRepos, "fetch", err)
		}
		output.Writef(cmd.OutOrStdout(), "[%s] fetch\n", unit.setName)
	}

	workers := parallel.MaxWorkers(0, len(unit.setRepos))
	return parallel.RunFanOut(unit.setRepos, workers, func(r repo.Repo) branchReport {
		return switchInWorktree(ctx, r, worktreeTargetBranch(cfg, r.Name), pull)
	})
}

func switchInWorktree(ctx context.Context, r repo.Repo, targetBranch string, pull bool) branchReport {
	report := branchReport{RepoName: r.Name}

	if !checkoutDefault(ctx, &report, r, targetBranch) {
		return report
	}

	if pull {
		pullSoft(ctx, &report, r.AbsPath, targetBranch)
	}

	return report
}

func worktreeTargetBranch(cfg *config.WorkspaceConfig, repoName string) string {
	branch, ok := cfg.WorktreeBranchForRepo(repoName)
	if ok {
		return branch
	}

	return "main"
}
