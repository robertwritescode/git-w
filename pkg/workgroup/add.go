package workgroup

import (
	"context"
	"fmt"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/spf13/cobra"
)

type addInputs struct {
	cfgPath    string
	wgName     string
	repos      []repo.Repo
	cfg        *config.WorkspaceConfig
	existingWg config.WorkgroupConfig
	flags      checkoutWorkFlags
}

func registerAdd(workCmd *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "add <name> [repos]...",
		Short: "Add repos to an existing workgroup",
		RunE:  runAdd,
	}

	cmd.Flags().Bool("pull", false, "pull after attaching to an existing branch")
	cmd.Flags().Bool("push", false, "push newly created branches to origin")
	cmd.Flags().Bool("no-push", false, "skip pushing")
	cmd.Flags().Bool("allow-upstream", false, "set tracking upstream on newly created branches")
	cmd.Flags().Bool("no-upstream", false, "skip setting upstream")

	workCmd.AddCommand(cmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	inputs, err := resolveAddInputs(cmd, args)
	if err != nil {
		return err
	}

	reports := collectAddReports(ctx, inputs)
	writeWorkReports(cmd, reports)
	writeSummary(cmd, reports, "work add")

	if err := persistAddWorkgroup(inputs, reports); err != nil {
		return err
	}

	return workReportsError(reports, "work add")
}

func resolveAddInputs(cmd *cobra.Command, args []string) (addInputs, error) {
	wgName, targets, err := parseWorkArgs(args)
	if err != nil {
		return addInputs{}, err
	}

	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return addInputs{}, err
	}

	repos, existingWg, err := resolveNewAddTargets(cfg, cfgPath, wgName, targets)
	if err != nil {
		return addInputs{}, err
	}

	flags, err := resolveCheckoutWorkFlags(cmd, cfg)
	if err != nil {
		return addInputs{}, err
	}

	return addInputs{cfgPath: cfgPath, wgName: wgName, repos: repos, cfg: cfg, existingWg: existingWg, flags: flags}, nil
}

func resolveNewAddTargets(cfg *config.WorkspaceConfig, cfgPath, wgName string, targets []string) ([]repo.Repo, config.WorkgroupConfig, error) {
	existingWg, ok := cfg.Workgroups[wgName]
	if !ok {
		return nil, config.WorkgroupConfig{}, fmt.Errorf("workgroup %q not found", wgName)
	}

	newTargets := filterNewRepos(targets, existingWg.Repos)
	if len(newTargets) == 0 {
		return nil, config.WorkgroupConfig{}, fmt.Errorf("all specified repos are already in workgroup %q", wgName)
	}

	repos, err := repo.Filter(cfg, cfgPath, newTargets)
	if err != nil {
		return nil, config.WorkgroupConfig{}, err
	}

	if len(repos) == 0 {
		return nil, config.WorkgroupConfig{}, fmt.Errorf("no repos matched")
	}

	return repos, existingWg, nil
}

func filterNewRepos(targets, existing []string) []string {
	existingSet := make(map[string]bool, len(existing))
	for _, r := range existing {
		existingSet[r] = true
	}

	result := make([]string, 0, len(targets))
	for _, t := range targets {
		if !existingSet[t] {
			result = append(result, t)
		}
	}

	return result
}

func collectAddReports(ctx context.Context, inputs addInputs) []workReport {
	if len(inputs.repos) == 0 {
		return nil
	}

	workers := parallel.MaxWorkers(0, len(inputs.repos))

	return parallel.RunFanOut(inputs.repos, workers, func(r repo.Repo) workReport {
		sourceBranch := inputs.cfg.ResolveDefaultBranch(r.Name)
		return checkoutInRepo(ctx, r, inputs.cfgPath, inputs.wgName, sourceBranch, inputs.flags)
	})
}

func persistAddWorkgroup(inputs addInputs, reports []workReport) error {
	succeeded := successRepoNames(reports, inputs.repos)
	if len(succeeded) == 0 {
		return nil
	}

	merged := unionRepos(inputs.existingWg.Repos, succeeded)

	wg := inputs.existingWg
	wg.Repos = merged

	return config.SaveLocalWorkgroup(inputs.cfgPath, inputs.wgName, wg)
}
