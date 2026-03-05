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

type createInputs struct {
	cfgPath string
	wgName  string
	repos   []repo.Repo
	cfg     *config.WorkspaceConfig
	flags   workFlags
}

func registerCreate(workCmd *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "create <name> [repos/groups]...",
		Short: "Create a workgroup with worktrees across repos",
		RunE:  runCreate,
	}

	cmd.Flags().BoolP("checkout", "c", false, "attach to existing branches instead of failing")
	cmd.Flags().Bool("push", false, "push newly created branches to origin")
	cmd.Flags().Bool("no-push", false, "skip pushing branches to origin")
	cmd.Flags().Bool("allow-upstream", false, "set tracking upstream on newly created branches")
	cmd.Flags().Bool("no-upstream", false, "skip setting upstream")

	workCmd.AddCommand(cmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	inputs, err := resolveCreateInputs(cmd, args)
	if err != nil {
		return err
	}

	reports := collectCreateReports(ctx, inputs)
	writeWorkReports(cmd, reports)
	writeSummary(cmd, reports, "work create")

	if err := persistCreateWorkgroup(inputs, reports); err != nil {
		return err
	}

	return workReportsError(reports, "work create")
}

func resolveCreateInputs(cmd *cobra.Command, args []string) (createInputs, error) {
	wgName, targets, err := parseWorkArgs(args)
	if err != nil {
		return createInputs{}, err
	}

	cfg, cfgPath, repos, err := loadWorkInputs(cmd, targets)
	if err != nil {
		return createInputs{}, err
	}

	if err := validateCreatePreconditions(cmd, cfgPath, cfg, wgName); err != nil {
		return createInputs{}, err
	}

	flags, err := resolveWorkFlags(cmd, cfg)
	if err != nil {
		return createInputs{}, err
	}

	return createInputs{cfgPath: cfgPath, wgName: wgName, repos: repos, cfg: cfg, flags: flags}, nil
}

func validateCreatePreconditions(cmd *cobra.Command, cfgPath string, cfg *config.WorkspaceConfig, wgName string) error {
	checkout, _ := cmd.Flags().GetBool("checkout")

	if _, exists := cfg.Workgroups[wgName]; exists && !checkout {
		return fmt.Errorf("workgroup %q already exists; use --checkout/-c to attach to existing branches", wgName)
	}

	return gitutil.EnsureGitignore(config.ConfigDir(cfgPath), ".workgroup/")
}

func collectCreateReports(ctx context.Context, inputs createInputs) []workReport {
	if len(inputs.repos) == 0 {
		return nil
	}

	workers := parallel.MaxWorkers(0, len(inputs.repos))

	return parallel.RunFanOut(inputs.repos, workers, func(r repo.Repo) workReport {
		sourceBranch := inputs.cfg.ResolveDefaultBranch(r.Name)
		return createInRepo(ctx, r, inputs.cfgPath, inputs.wgName, sourceBranch, inputs.flags)
	})
}

func createInRepo(ctx context.Context, r repo.Repo, cfgPath, wgName, sourceBranch string, flags workFlags) workReport {
	report := workReport{RepoName: r.Name}
	treePath := worktreePath(cfgPath, wgName, r.Name)

	if pathExists(treePath) {
		resolveExistingTreePath(ctx, &report, r, treePath, wgName, false)
		return report
	}

	if flags.Checkout {
		createIdempotent(ctx, &report, r, treePath, wgName, sourceBranch, flags)
	} else {
		createForce(ctx, &report, r, treePath, wgName, sourceBranch, flags)
	}

	return report
}

func createForce(ctx context.Context, report *workReport, r repo.Repo, treePath, wgName, sourceBranch string, flags workFlags) {
	if !runStep(report, "worktree", func() error {
		return gitutil.AddWorktreeNewBranch(ctx, r.AbsPath, treePath, wgName, sourceBranch)
	}) {
		return
	}

	applyWorkRemoteOps(ctx, report, r, wgName, flags)
}

func createIdempotent(ctx context.Context, report *workReport, r repo.Repo, treePath, wgName, sourceBranch string, flags workFlags) {
	loc, err := gitutil.ResolveBranchLocation(ctx, r.AbsPath, wgName)
	if err != nil {
		recordStep(report, "worktree", err, false)
		return
	}

	switch loc {
	case gitutil.BranchLocal:
		runStep(report, "worktree", func() error {
			return gitutil.AddWorktree(ctx, r.AbsPath, treePath, wgName)
		})
	case gitutil.BranchRemote:
		attachRemoteBranch(ctx, report, r, treePath, wgName)
	case gitutil.BranchMissing:
		createForce(ctx, report, r, treePath, wgName, sourceBranch, flags)
	}
}

func persistCreateWorkgroup(inputs createInputs, reports []workReport) error {
	succeeded := successRepoNames(reports, inputs.repos)
	if len(succeeded) == 0 {
		return nil
	}

	wg := config.WorkgroupConfig{
		Repos:   succeeded,
		Branch:  inputs.wgName,
		Created: time.Now().UTC().Format(time.RFC3339),
	}

	return config.SaveLocalWorkgroup(inputs.cfgPath, inputs.wgName, wg)
}
