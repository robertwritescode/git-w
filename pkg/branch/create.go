package branch

import (
	"context"
	"fmt"
	"strings"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/spf13/cobra"
)

type branchFlags struct {
	SyncSource  bool
	SetUpstream bool
	Push        bool
}

type branchStep struct {
	name    string
	err     error
	skipped bool
}

type branchReport struct {
	RepoName string
	Steps    []branchStep
	Failed   bool
	isSet    bool
}

type branchUnit struct {
	isWorktree   bool
	plain        *repo.Repo
	sourceBranch string
	branches     map[string]string
	setName      string
	setRepos     []repo.Repo
	setConfig    config.WorktreeConfig
}

func registerCreate(root *cobra.Command) {
	branchCmd := &cobra.Command{
		Use:     "branch",
		Aliases: []string{"b"},
		Short:   "Manage branches across repos",
	}

	createCmd := &cobra.Command{
		Use:     "create <branchname> [repos/groups/sets]...",
		Aliases: []string{"c", "cut", "new"},
		Short:   "Create a branch across repos",
		RunE:    runCreate,
	}

	createCmd.Flags().Bool("sync-source", false, "sync source branch before creating")
	createCmd.Flags().Bool("no-sync-source", false, "skip syncing source branch")
	createCmd.Flags().Bool("allow-upstream", false, "set tracking upstream")
	createCmd.Flags().Bool("no-upstream", false, "skip setting upstream")
	createCmd.Flags().Bool("push", false, "push branch to origin")
	createCmd.Flags().Bool("no-push", false, "skip pushing branch to origin")

	branchCmd.AddCommand(createCmd)
	root.AddCommand(branchCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	branchName, targets, err := parseBranchArgs(args)
	if err != nil {
		return err
	}

	cfg, cfgPath, repos, err := loadBranchInputs(cmd, targets)
	if err != nil {
		return err
	}

	flags, err := resolveBranchFlags(cmd, cfg)
	if err != nil {
		return err
	}

	reports := collectBranchReports(ctx, cfgPath, repos, cfg, branchName, flags)
	writeBranchReports(cmd, reports)
	writeBranchSummary(cmd, reports)

	return branchReportsError(reports)
}

func parseBranchArgs(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("branch name is required")
	}

	return args[0], args[1:], nil
}

func loadBranchInputs(cmd *cobra.Command, targets []string) (*config.WorkspaceConfig, string, []repo.Repo, error) {
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

func resolveBranchFlags(cmd *cobra.Command, cfg *config.WorkspaceConfig) (branchFlags, error) {
	syncSource, err := resolveBoolFlag(cmd, "sync-source", "no-sync-source", cfg.BranchSyncSourceEnabled())
	if err != nil {
		return branchFlags{}, err
	}

	setUpstream, err := resolveBoolFlag(cmd, "allow-upstream", "no-upstream", cfg.BranchSetUpstreamEnabled())
	if err != nil {
		return branchFlags{}, err
	}

	push, err := resolveBoolFlag(cmd, "push", "no-push", cfg.BranchPushEnabled())
	if err != nil {
		return branchFlags{}, err
	}

	return branchFlags{SyncSource: syncSource, SetUpstream: setUpstream, Push: push}, nil
}

func resolveBoolFlag(cmd *cobra.Command, onFlag, offFlag string, dflt bool) (bool, error) {
	on, _ := cmd.Flags().GetBool(onFlag)
	off, _ := cmd.Flags().GetBool(offFlag)

	if on && off {
		return false, fmt.Errorf("--%s and --%s cannot be used together", onFlag, offFlag)
	}

	if on {
		return true, nil
	}

	if off {
		return false, nil
	}

	return dflt, nil
}

func collectBranchReports(ctx context.Context, cfgPath string, repos []repo.Repo, cfg *config.WorkspaceConfig, branchName string, flags branchFlags) []branchReport {
	units := buildBranchUnits(cfg, repos)

	if len(units) == 0 {
		return nil
	}

	if len(units) == 1 {
		return executeBranchUnit(ctx, cfgPath, units[0], branchName, flags)
	}

	workers := parallel.MaxWorkers(0, len(units))
	allReports := parallel.RunFanOut(units, workers, func(unit branchUnit) []branchReport {
		return executeBranchUnit(ctx, cfgPath, unit, branchName, flags)
	})

	return flattenBranchReports(allReports)
}

func flattenBranchReports(allReports [][]branchReport) []branchReport {
	reports := make([]branchReport, 0)

	for _, unitReports := range allReports {
		reports = append(reports, unitReports...)
	}

	return reports
}

func buildBranchUnits(cfg *config.WorkspaceConfig, repos []repo.Repo) []branchUnit {
	plain, sets := classifyRepos(cfg, repos)

	units := plainUnits(cfg, plain)
	units = append(units, worktreeUnits(cfg, sets)...)

	return units
}

func classifyRepos(cfg *config.WorkspaceConfig, repos []repo.Repo) ([]repo.Repo, map[string][]repo.Repo) {
	byRepo := config.WorktreeRepoToSetIndex(cfg)
	plain := make([]repo.Repo, 0, len(repos))
	sets := make(map[string][]repo.Repo)

	for _, r := range repos {
		setName, isWorktree := byRepo[r.Name]
		if !isWorktree {
			plain = append(plain, r)
			continue
		}

		sets[setName] = append(sets[setName], r)
	}

	return plain, sets
}

func plainUnits(cfg *config.WorkspaceConfig, repos []repo.Repo) []branchUnit {
	units := make([]branchUnit, 0, len(repos))

	for _, r := range repos {
		rCopy := r
		units = append(units, branchUnit{plain: &rCopy, sourceBranch: cfg.ResolveDefaultBranch(r.Name)})
	}

	return units
}

func worktreeUnits(cfg *config.WorkspaceConfig, sets map[string][]repo.Repo) []branchUnit {
	units := make([]branchUnit, 0, len(sets))

	for _, setName := range config.SortedStringKeys(sets) {
		repos := sets[setName]
		branches := make(map[string]string, len(repos))
		for _, r := range repos {
			if branch, ok := cfg.WorktreeBranchForRepo(r.Name); ok {
				branches[r.Name] = branch
			}
		}

		units = append(units, branchUnit{
			isWorktree: true,
			setName:    setName,
			setRepos:   repos,
			branches:   branches,
			setConfig:  cfg.Worktrees[setName],
		})
	}

	return units
}

func executeBranchUnit(ctx context.Context, cfgPath string, unit branchUnit, branchName string, flags branchFlags) []branchReport {
	if !unit.isWorktree {
		return []branchReport{createInPlainRepo(ctx, *unit.plain, branchName, unit.sourceBranch, flags)}
	}

	return createInWorktreeSet(ctx, cfgPath, unit, branchName, flags)
}

func createInPlainRepo(ctx context.Context, r repo.Repo, branchName, sourceBranch string, flags branchFlags) branchReport {
	report := branchReport{RepoName: r.Name}

	if flags.SyncSource {
		if !syncSourcePlainRepo(ctx, &report, r, sourceBranch) {
			return report
		}
	}

	if !createBranchStep(ctx, &report, r, branchName, sourceBranch) {
		return report
	}

	if !hasRemote(ctx, r) {
		skipRemoteOps(&report, flags)
		return report
	}

	applyRemoteOps(ctx, &report, r, branchName, flags)
	return report
}

func syncSourcePlainRepo(ctx context.Context, report *branchReport, r repo.Repo, sourceBranch string) bool {
	if !runStep(report, "checkout", func() error { return gitutil.CheckoutBranch(ctx, r.AbsPath, sourceBranch) }) {
		return false
	}

	if !hasRemote(ctx, r) {
		return true
	}

	if !runStep(report, "fetch", func() error { return gitutil.FetchOrigin(ctx, r.AbsPath) }) {
		return false
	}

	return runStep(report, "pull", func() error { return gitutil.PullBranch(ctx, r.AbsPath, sourceBranch) })
}

func applyRemoteOps(ctx context.Context, report *branchReport, r repo.Repo, branchName string, flags branchFlags) {
	if flags.Push {
		runStep(report, "push", func() error { return gitutil.PushBranchUpstream(ctx, r.AbsPath, "origin", branchName) })
		return
	}

	if flags.SetUpstream {
		runStep(report, "upstream", func() error { return gitutil.SetBranchUpstream(ctx, r.AbsPath, branchName, "origin") })
	}
}

func createInWorktreeSet(ctx context.Context, cfgPath string, unit branchUnit, branchName string, flags branchFlags) []branchReport {
	if flags.SyncSource {
		if err := fetchBareRepo(ctx, cfgPath, unit.setConfig); err != nil {
			setReport := branchReport{RepoName: unit.setName, Failed: true, isSet: true}
			recordStep(&setReport, "fetch", err, false)
			return append([]branchReport{setReport}, failedSetReports(unit.setRepos, "fetch", err)...)
		}

		setReport := branchReport{RepoName: unit.setName, isSet: true}
		recordStep(&setReport, "fetch", nil, false)

		reports := runWorktreeReports(ctx, unit, branchName, flags)
		return append([]branchReport{setReport}, reports...)
	}

	return runWorktreeReports(ctx, unit, branchName, flags)
}

func runWorktreeReports(ctx context.Context, unit branchUnit, branchName string, flags branchFlags) []branchReport {
	repos := unit.setRepos
	if len(repos) == 0 {
		return nil
	}

	if len(repos) == 1 {
		r := repos[0]
		branch, ok := worktreeBranchForRepo(unit, r.Name)
		if !ok {
			return []branchReport{missingWorktreeBranchReport(r)}
		}

		folderName, ok := extractWorktreeFolderName(unit.setName, r.Name)
		if !ok {
			return []branchReport{missingWorktreeBranchReport(r)}
		}

		finalBranchName := fmt.Sprintf("%s-%s", folderName, branchName)
		return []branchReport{createInWorktree(ctx, r, finalBranchName, branch, flags)}
	}

	workers := parallel.MaxWorkers(0, len(repos))
	return parallel.RunFanOut(repos, workers, func(r repo.Repo) branchReport {
		branch, ok := worktreeBranchForRepo(unit, r.Name)
		if !ok {
			return missingWorktreeBranchReport(r)
		}

		folderName, ok := extractWorktreeFolderName(unit.setName, r.Name)
		if !ok {
			return missingWorktreeBranchReport(r)
		}

		finalBranchName := fmt.Sprintf("%s-%s", folderName, branchName)
		return createInWorktree(ctx, r, finalBranchName, branch, flags)
	})
}

func createInWorktree(ctx context.Context, r repo.Repo, branchName, sourceBranch string, flags branchFlags) branchReport {
	report := branchReport{RepoName: r.Name}

	if !syncWorktreeSource(ctx, &report, r, sourceBranch, flags) {
		return report
	}

	if !createBranchStep(ctx, &report, r, branchName, sourceBranch) {
		return report
	}

	if !hasRemote(ctx, r) {
		skipRemoteOps(&report, flags)
		return report
	}

	applyRemoteOps(ctx, &report, r, branchName, flags)
	return report
}

func syncWorktreeSource(ctx context.Context, report *branchReport, r repo.Repo, sourceBranch string, flags branchFlags) bool {
	if !flags.SyncSource || !hasRemote(ctx, r) {
		return true
	}

	if !runStep(report, "checkout", func() error { return gitutil.CheckoutBranch(ctx, r.AbsPath, sourceBranch) }) {
		return false
	}

	return runStep(report, "pull", func() error { return gitutil.PullBranch(ctx, r.AbsPath, sourceBranch) })
}

func fetchBareRepo(ctx context.Context, cfgPath string, wt config.WorktreeConfig) error {
	bareAbsPath, err := config.ResolveRepoPath(cfgPath, wt.BarePath)
	if err != nil {
		return err
	}

	return gitutil.FetchBare(ctx, bareAbsPath)
}

func worktreeBranchForRepo(unit branchUnit, repoName string) (string, bool) {
	branch, ok := unit.branches[repoName]
	return branch, ok
}

func extractWorktreeFolderName(setName, repoName string) (string, bool) {
	prefix := setName + "-"
	if !strings.HasPrefix(repoName, prefix) {
		return "", false
	}
	return strings.TrimPrefix(repoName, prefix), true
}

func missingWorktreeBranchReport(r repo.Repo) branchReport {
	report := branchReport{RepoName: r.Name, Failed: true}
	recordStep(&report, "branch", fmt.Errorf("worktree branch not found for %s", r.Name), false)
	return report
}

func failedSetReports(repos []repo.Repo, stepName string, err error) []branchReport {
	reports := make([]branchReport, 0, len(repos))

	for _, r := range repos {
		report := branchReport{RepoName: r.Name, Failed: true}
		recordStep(&report, stepName, err, false)
		reports = append(reports, report)
	}

	return reports
}

func createBranchStep(ctx context.Context, report *branchReport, r repo.Repo, branchName, sourceBranch string) bool {
	exists, err := gitutil.BranchExists(ctx, r.AbsPath, branchName)
	if err != nil {
		recordStep(report, "branch", err, false)
		return false
	}

	if exists {
		recordStep(report, "branch", nil, true)
		return false
	}

	if err := gitutil.CreateBranch(ctx, r.AbsPath, branchName, sourceBranch); err != nil {
		return handleCreateBranchError(ctx, report, r, branchName, err)
	}

	recordStep(report, "branch", nil, false)
	return true
}

func handleCreateBranchError(ctx context.Context, report *branchReport, r repo.Repo, branchName string, err error) bool {
	exists, existsErr := gitutil.BranchExists(ctx, r.AbsPath, branchName)
	if existsErr == nil && exists {
		recordStep(report, "branch", nil, true)
		return false
	}

	recordStep(report, "branch", err, false)
	return false
}

func runStep(report *branchReport, stepName string, fn func() error) bool {
	err := fn()

	recordStep(report, stepName, err, false)
	return err == nil
}

func recordStep(report *branchReport, stepName string, err error, skipped bool) {
	report.Steps = append(report.Steps, branchStep{name: stepName, err: err, skipped: skipped})

	if err != nil {
		report.Failed = true
	}
}

func hasRemote(ctx context.Context, r repo.Repo) bool {
	return gitutil.RemoteURL(ctx, r.AbsPath) != ""
}

func skipRemoteOps(report *branchReport, flags branchFlags) {
	if flags.Push {
		skipNoRemote(report, "push")
		return
	}

	if flags.SetUpstream {
		skipNoRemote(report, "upstream")
	}
}

func skipNoRemote(report *branchReport, stepName string) {
	recordStep(report, stepName, nil, true)
}

func writeBranchReports(cmd *cobra.Command, reports []branchReport) {
	for _, report := range reports {
		for _, step := range report.Steps {
			if step.skipped {
				output.Writef(cmd.OutOrStdout(), "[%s] %s: %s, skipped\n", report.RepoName, step.name, stepSkipMessage(step.name))
				continue
			}

			if step.err != nil {
				output.Writef(cmd.ErrOrStderr(), "[%s] %s error: %v\n", report.RepoName, step.name, step.err)
				continue
			}

			output.Writef(cmd.OutOrStdout(), "[%s] %s\n", report.RepoName, step.name)
		}
	}
}

func stepSkipMessage(stepName string) string {
	if stepName == "branch" {
		return "already exists"
	}

	if stepName == "push" || stepName == "upstream" {
		return "no remote"
	}

	return "skipped"
}

func writeBranchSummary(cmd *cobra.Command, reports []branchReport) {
	ok, failed := countBranchReports(reports)

	output.Writef(cmd.OutOrStdout(), "branch create complete: %d ok, %d failed\n", ok, failed)
}

func repoReports(reports []branchReport) []branchReport {
	out := make([]branchReport, 0, len(reports))

	for _, r := range reports {
		if !r.isSet {
			out = append(out, r)
		}
	}

	return out
}

func countBranchReports(reports []branchReport) (int, int) {
	filtered := repoReports(reports)
	failed := 0

	for _, r := range filtered {
		if r.Failed {
			failed++
		}
	}

	return len(filtered) - failed, failed
}

func branchReportsError(reports []branchReport) error {
	filtered := repoReports(reports)
	failures := make([]string, 0)

	for _, r := range filtered {
		if r.Failed {
			failures = append(failures, fmt.Sprintf("  [%s]: branch create failed", r.RepoName))
		}
	}

	return parallel.FormatFailureError(failures, len(filtered))
}
