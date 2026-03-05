package workgroup

import (
	"context"
	"fmt"
	"time"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/spf13/cobra"
)

type checkoutInputs struct {
	cfgPath    string
	wgName     string
	repos      []repo.Repo
	cfg        *config.WorkspaceConfig
	flags      checkoutWorkFlags
	existingWg *config.WorkgroupConfig
}

func registerCheckout(workCmd *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "checkout <name> [repos/groups]...",
		Aliases: []string{"co", "switch"},
		Short:   "Resume or create a workgroup branch across repos",
		RunE:    runCheckout,
	}

	cmd.Flags().Bool("pull", false, "pull after attaching to an existing branch")
	cmd.Flags().Bool("push", false, "push newly created branches to origin")
	cmd.Flags().Bool("no-push", false, "skip pushing")
	cmd.Flags().Bool("allow-upstream", false, "set tracking upstream on newly created branches")
	cmd.Flags().Bool("no-upstream", false, "skip setting upstream")

	workCmd.AddCommand(cmd)
}

func runCheckout(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	inputs, err := resolveCheckoutInputs(cmd, args)
	if err != nil {
		return err
	}

	reports := collectCheckoutReports(ctx, inputs)
	writeWorkReports(cmd, reports)
	writeSummary(cmd, reports, "work checkout")

	if err := mergeWorkgroupRepos(inputs, reports); err != nil {
		return err
	}

	return workReportsError(reports, "work checkout")
}

func resolveCheckoutInputs(cmd *cobra.Command, args []string) (checkoutInputs, error) {
	wgName, targets, err := parseWorkArgs(args)
	if err != nil {
		return checkoutInputs{}, err
	}

	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return checkoutInputs{}, err
	}

	repos, existingWg, err := resolveCheckoutRepos(cfg, cfgPath, wgName, targets)
	if err != nil {
		return checkoutInputs{}, err
	}

	flags, err := prepareCheckoutEnv(cmd, cfg, cfgPath)
	if err != nil {
		return checkoutInputs{}, err
	}

	return checkoutInputs{cfgPath: cfgPath, wgName: wgName, repos: repos, cfg: cfg, flags: flags, existingWg: existingWg}, nil
}

func prepareCheckoutEnv(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath string) (checkoutWorkFlags, error) {
	if err := gitutil.EnsureGitignore(config.ConfigDir(cfgPath), ".workgroup/"); err != nil {
		return checkoutWorkFlags{}, err
	}

	return resolveCheckoutWorkFlags(cmd, cfg)
}

func resolveCheckoutRepos(cfg *config.WorkspaceConfig, cfgPath, wgName string, targets []string) ([]repo.Repo, *config.WorkgroupConfig, error) {
	targets, existingWg, err := resolveCheckoutTargets(cfg, wgName, targets)
	if err != nil {
		return nil, nil, err
	}

	repos, err := repo.Filter(cfg, cfgPath, targets)
	if err != nil {
		return nil, nil, err
	}

	if len(repos) == 0 {
		return nil, nil, fmt.Errorf("no repos matched")
	}

	return repos, existingWg, nil
}

func resolveCheckoutTargets(cfg *config.WorkspaceConfig, wgName string, targets []string) ([]string, *config.WorkgroupConfig, error) {
	var existingWg *config.WorkgroupConfig

	if wg, ok := cfg.Workgroups[wgName]; ok {
		existingWg = &wg
		if len(targets) == 0 {
			targets = wg.Repos
		}
	}

	if len(targets) == 0 {
		return nil, nil, fmt.Errorf("workgroup %q not found; specify repos/groups to create it", wgName)
	}

	return targets, existingWg, nil
}

func collectCheckoutReports(ctx context.Context, inputs checkoutInputs) []workReport {
	if len(inputs.repos) == 0 {
		return nil
	}

	workers := parallel.MaxWorkers(0, len(inputs.repos))

	return parallel.RunFanOut(inputs.repos, workers, func(r repo.Repo) workReport {
		sourceBranch := inputs.cfg.ResolveDefaultBranch(r.Name)
		return checkoutInRepo(ctx, r, inputs.cfgPath, inputs.wgName, sourceBranch, inputs.flags)
	})
}

func checkoutInRepo(ctx context.Context, r repo.Repo, cfgPath, wgName, sourceBranch string, flags checkoutWorkFlags) workReport {
	report := workReport{RepoName: r.Name}
	treePath := config.WorkgroupWorktreePath(cfgPath, wgName, r.Name)

	if pathExists(treePath) {
		resolveExistingTreePath(ctx, &report, r, treePath, wgName, flags.Pull)
		return report
	}

	if err := gitutil.PruneWorktrees(ctx, r.AbsPath); err != nil {
		recordStep(&report, "worktree", err, false)
		return report
	}

	loc, err := gitutil.ResolveBranchLocation(ctx, r.AbsPath, wgName)
	if err != nil {
		recordStep(&report, "worktree", err, false)
		return report
	}

	applyCheckoutWorktree(ctx, &report, r, treePath, wgName, sourceBranch, loc, flags)

	return report
}

func applyCheckoutWorktree(ctx context.Context, report *workReport, r repo.Repo, treePath, wgName, sourceBranch string, loc gitutil.BranchLocation, flags checkoutWorkFlags) {
	switch loc {
	case gitutil.BranchLocal:
		attachLocalBranch(ctx, report, r, treePath, wgName, flags.Pull)
	case gitutil.BranchRemote:
		attachRemoteBranchWithPull(ctx, report, r, treePath, wgName, flags.Pull)
	case gitutil.BranchMissing:
		createForce(ctx, report, r, treePath, wgName, sourceBranch, flags.workFlags)
	}
}

func attachLocalBranch(ctx context.Context, report *workReport, r repo.Repo, treePath, branchName string, pull bool) {
	if !runStep(report, "worktree", func() error {
		return gitutil.AddWorktree(ctx, r.AbsPath, treePath, branchName)
	}) {
		return
	}

	optionalPullWorktree(ctx, report, r, treePath, branchName, pull)
}

func attachRemoteBranchWithPull(ctx context.Context, report *workReport, r repo.Repo, treePath, branchName string, pull bool) {
	if !runStep(report, "fetch", func() error {
		return gitutil.FetchOrigin(ctx, r.AbsPath)
	}) {
		return
	}

	if !runStep(report, "worktree", func() error {
		return gitutil.AddWorktree(ctx, r.AbsPath, treePath, branchName)
	}) {
		return
	}

	optionalPullWorktree(ctx, report, r, treePath, branchName, pull)
}

func mergeWorkgroupRepos(inputs checkoutInputs, reports []workReport) error {
	succeeded := successRepoNames(reports, inputs.repos)
	if len(succeeded) == 0 {
		return nil
	}

	existingRepos := workgroupRepos(inputs.existingWg)
	merged := unionRepos(existingRepos, succeeded)

	wg := buildWorkgroupConfig(inputs, merged)

	return config.SaveLocalWorkgroup(inputs.cfgPath, inputs.wgName, wg)
}

func buildWorkgroupConfig(inputs checkoutInputs, repos []string) config.WorkgroupConfig {
	if inputs.existingWg != nil {
		wg := *inputs.existingWg
		wg.Repos = repos
		return wg
	}

	return config.WorkgroupConfig{
		Repos:   repos,
		Branch:  inputs.wgName,
		Created: time.Now().UTC().Format(time.RFC3339),
	}
}

func workgroupRepos(wg *config.WorkgroupConfig) []string {
	if wg == nil {
		return nil
	}

	return wg.Repos
}

func unionRepos(existing, added []string) []string {
	seen := make(map[string]bool, len(existing)+len(added))
	result := make([]string, 0, len(existing)+len(added))

	for _, r := range existing {
		if !seen[r] {
			seen[r] = true
			result = append(result, r)
		}
	}

	for _, r := range added {
		if !seen[r] {
			seen[r] = true
			result = append(result, r)
		}
	}

	return result
}
