package branch

import (
	"context"
	"fmt"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/spf13/cobra"
)

type checkoutFlags struct {
	branchFlags
	Pull bool
}

type checkoutStrategy int

const (
	strategySkip   checkoutStrategy = iota
	strategyLocal
	strategyRemote
	strategyCreate
)

func registerCheckout(branchCmd *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "checkout <branchname> [repos/groups/sets]...",
		Aliases: []string{"co", "switch"},
		Short:   "Check out a branch across repos, creating it if missing",
		RunE:    runCheckout,
	}

	cmd.Flags().Bool("pull", false, "pull after checking out an existing branch")
	cmd.Flags().Bool("sync-source", false, "sync source branch before creating missing branches (worktree sets fetch bare repo once)")
	cmd.Flags().Bool("no-sync-source", false, "skip syncing source branch")
	cmd.Flags().Bool("allow-upstream", false, "set tracking upstream on newly created branches")
	cmd.Flags().Bool("no-upstream", false, "skip setting upstream")
	cmd.Flags().Bool("push", false, "push newly created branches to origin")
	cmd.Flags().Bool("no-push", false, "skip pushing")

	branchCmd.AddCommand(cmd)
}

func runCheckout(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	branchName, targets, err := parseBranchArgs(args)
	if err != nil {
		return err
	}

	cfg, cfgPath, repos, err := loadBranchInputs(cmd, targets)
	if err != nil {
		return err
	}

	flags, err := resolveCheckoutFlags(cmd, cfg)
	if err != nil {
		return err
	}

	reports := collectCheckoutReports(ctx, cfgPath, repos, cfg, branchName, flags)
	writeBranchReports(cmd, reports)
	writeSummary(cmd, reports, "branch checkout")

	return branchReportsError(reports, "branch checkout")
}

func resolveCheckoutFlags(cmd *cobra.Command, cfg *config.WorkspaceConfig) (checkoutFlags, error) {
	flags, err := resolveBranchFlags(cmd, cfg)
	if err != nil {
		return checkoutFlags{}, err
	}

	pull, _ := cmd.Flags().GetBool("pull")

	return checkoutFlags{branchFlags: flags, Pull: pull}, nil
}

func collectCheckoutReports(ctx context.Context, cfgPath string, repos []repo.Repo, cfg *config.WorkspaceConfig, branchName string, flags checkoutFlags) []branchReport {
	units := buildBranchUnits(cfg, repos)

	if len(units) == 0 {
		return nil
	}

	if len(units) == 1 {
		return executeCheckoutUnit(ctx, cfgPath, units[0], branchName, flags)
	}

	workers := parallel.MaxWorkers(0, len(units))
	allReports := parallel.RunFanOut(units, workers, func(unit branchUnit) []branchReport {
		return executeCheckoutUnit(ctx, cfgPath, unit, branchName, flags)
	})

	return flattenBranchReports(allReports)
}

func executeCheckoutUnit(ctx context.Context, cfgPath string, unit branchUnit, branchName string, flags checkoutFlags) []branchReport {
	if !unit.isWorktree {
		return []branchReport{checkoutInPlainRepo(ctx, *unit.plain, branchName, unit.sourceBranch, flags)}
	}

	return checkoutInWorktreeSet(ctx, cfgPath, unit, branchName, flags)
}

func checkoutInPlainRepo(ctx context.Context, r repo.Repo, branchName, sourceBranch string, flags checkoutFlags) branchReport {
	report := branchReport{RepoName: r.Name}

	strategy, err := resolveCheckoutStrategy(ctx, r, branchName)
	if err != nil {
		recordStep(&report, "checkout", err, false)
		return report
	}

	applyCheckout(ctx, &report, r, branchName, sourceBranch, strategy, flags)

	return report
}

func resolveCheckoutStrategy(ctx context.Context, r repo.Repo, branchName string) (checkoutStrategy, error) {
	cur, err := gitutil.CurrentBranch(ctx, r.AbsPath)
	if err != nil {
		return 0, err
	}

	if cur == branchName {
		return strategySkip, nil
	}

	return resolveFromBranchState(ctx, r, branchName)
}

func resolveFromBranchState(ctx context.Context, r repo.Repo, branchName string) (checkoutStrategy, error) {
	loc, err := gitutil.ResolveBranchLocation(ctx, r.AbsPath, branchName)
	if err != nil {
		return 0, err
	}

	return locationToStrategy(loc), nil
}

func locationToStrategy(loc gitutil.BranchLocation) checkoutStrategy {
	switch loc {
	case gitutil.BranchLocal:
		return strategyLocal
	case gitutil.BranchRemote:
		return strategyRemote
	default:
		return strategyCreate
	}
}

func applyCheckoutBase(ctx context.Context, report *branchReport, r repo.Repo, branchName string, strategy checkoutStrategy, pull bool, createFn func()) {
	switch strategy {
	case strategySkip:
		recordSkippedCheckout(report, branchName)
		optionalPull(ctx, report, r, branchName, pull)
	case strategyLocal:
		checkoutExistingBranch(ctx, report, r, branchName, pull)
	case strategyRemote:
		fetchAndCheckoutRemote(ctx, report, r, branchName, pull)
	default:
		createFn()
	}
}

func applyCheckout(ctx context.Context, report *branchReport, r repo.Repo, branchName, sourceBranch string, strategy checkoutStrategy, flags checkoutFlags) {
	applyCheckoutBase(ctx, report, r, branchName, strategy, flags.Pull, func() {
		createAndCheckoutPlain(ctx, report, r, branchName, sourceBranch, flags)
	})
}

func applyCheckoutWorktree(ctx context.Context, report *branchReport, r repo.Repo, branchName, sourceBranch string, strategy checkoutStrategy, flags checkoutFlags) {
	applyCheckoutBase(ctx, report, r, branchName, strategy, flags.Pull, func() {
		createAndCheckoutWorktree(ctx, report, r, branchName, sourceBranch, flags)
	})
}

func recordSkippedCheckout(report *branchReport, branchName string) {
	report.Steps = append(report.Steps, branchStep{
		name:    "checkout",
		skipped: true,
		detail:  fmt.Sprintf("already on %s", branchName),
	})
}

func optionalPull(ctx context.Context, report *branchReport, r repo.Repo, branchName string, pull bool) {
	if !pull {
		return
	}

	if !hasRemote(ctx, r) {
		skipNoRemote(report, "pull")
		return
	}

	pullSoft(ctx, report, r.AbsPath, branchName)
}

func checkoutExistingBranch(ctx context.Context, report *branchReport, r repo.Repo, branchName string, pull bool) {
	if !runStep(report, "checkout", func() error {
		return gitutil.CheckoutBranch(ctx, r.AbsPath, branchName)
	}) {
		return
	}

	optionalPull(ctx, report, r, branchName, pull)
}

func fetchAndCheckoutRemote(ctx context.Context, report *branchReport, r repo.Repo, branchName string, pull bool) {
	if !runStep(report, "fetch", func() error {
		return gitutil.FetchOrigin(ctx, r.AbsPath)
	}) {
		return
	}

	if !runStep(report, "checkout", func() error {
		return gitutil.CheckoutBranch(ctx, r.AbsPath, branchName)
	}) {
		return
	}

	optionalPull(ctx, report, r, branchName, pull)
}

func createAndCheckoutPlain(ctx context.Context, report *branchReport, r repo.Repo, branchName, sourceBranch string, flags checkoutFlags) {
	if flags.SyncSource {
		if !syncSourcePlainRepo(ctx, report, r, sourceBranch) {
			return
		}
	}

	if !runStep(report, "branch", func() error {
		return gitutil.CreateBranch(ctx, r.AbsPath, branchName, sourceBranch)
	}) {
		return
	}

	if !runStep(report, "checkout", func() error {
		return gitutil.CheckoutBranch(ctx, r.AbsPath, branchName)
	}) {
		return
	}

	applyCreateRemoteOps(ctx, report, r, branchName, flags.branchFlags)
}

func createAndCheckoutWorktree(ctx context.Context, report *branchReport, r repo.Repo, branchName, sourceBranch string, flags checkoutFlags) {
	if !syncWorktreeSource(ctx, report, r, sourceBranch, flags.branchFlags) {
		return
	}

	if !runStep(report, "branch", func() error {
		return gitutil.CreateBranch(ctx, r.AbsPath, branchName, sourceBranch)
	}) {
		return
	}

	if !runStep(report, "checkout", func() error {
		return gitutil.CheckoutBranch(ctx, r.AbsPath, branchName)
	}) {
		return
	}

	applyCreateRemoteOps(ctx, report, r, branchName, flags.branchFlags)
}

func checkoutInWorktreeSet(ctx context.Context, cfgPath string, unit branchUnit, branchName string, flags checkoutFlags) []branchReport {
	if flags.SyncSource || flags.Pull {
		return checkoutWorktreeSetWithFetch(ctx, cfgPath, unit, branchName, flags)
	}

	return runCheckoutWorktreeReports(ctx, unit, branchName, flags)
}

func checkoutWorktreeSetWithFetch(ctx context.Context, cfgPath string, unit branchUnit, branchName string, flags checkoutFlags) []branchReport {
	if err := fetchBareRepo(ctx, cfgPath, unit.setConfig); err != nil {
		setReport := branchReport{RepoName: unit.setName, Failed: true, isSet: true}
		recordStep(&setReport, "fetch", err, false)
		return append([]branchReport{setReport}, failedSetReports(unit.setRepos, "fetch", err)...)
	}

	setReport := branchReport{RepoName: unit.setName, isSet: true}
	recordStep(&setReport, "fetch", nil, false)

	reports := runCheckoutWorktreeReports(ctx, unit, branchName, flags)

	return append([]branchReport{setReport}, reports...)
}

func runCheckoutWorktreeReports(ctx context.Context, unit branchUnit, branchName string, flags checkoutFlags) []branchReport {
	if len(unit.setRepos) == 0 {
		return nil
	}

	workers := parallel.MaxWorkers(0, len(unit.setRepos))

	return parallel.RunFanOut(unit.setRepos, workers, func(r repo.Repo) branchReport {
		return checkoutWorktreeReport(ctx, unit, r, branchName, flags)
	})
}

func checkoutWorktreeReport(ctx context.Context, unit branchUnit, r repo.Repo, branchName string, flags checkoutFlags) branchReport {
	sourceBranch, ok := worktreeBranchForRepo(unit, r.Name)
	if !ok {
		return missingWorktreeBranchReport(r)
	}

	folderName, ok := extractWorktreeFolderName(unit.setName, r.Name)
	if !ok {
		return missingWorktreeBranchReport(r)
	}

	finalBranchName := fmt.Sprintf("%s-%s", folderName, branchName)

	return checkoutInWorktreeRepo(ctx, r, finalBranchName, sourceBranch, flags)
}

func checkoutInWorktreeRepo(ctx context.Context, r repo.Repo, branchName, sourceBranch string, flags checkoutFlags) branchReport {
	report := branchReport{RepoName: r.Name}

	strategy, err := resolveCheckoutStrategy(ctx, r, branchName)
	if err != nil {
		recordStep(&report, "checkout", err, false)
		return report
	}

	applyCheckoutWorktree(ctx, &report, r, branchName, sourceBranch, strategy, flags)

	return report
}
