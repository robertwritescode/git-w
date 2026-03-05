package workgroup

import (
	"context"
	"fmt"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/spf13/cobra"
)

func registerPush(workCmd *cobra.Command) {
	workCmd.AddCommand(&cobra.Command{
		Use:   "push <name>",
		Short: "Push all branches in a workgroup to origin",
		Args:  cobra.ExactArgs(1),
		RunE:  runPush,
	})
}

func runPush(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	wgName := args[0]
	wg, ok := cfg.Workgroups[wgName]
	if !ok {
		return fmt.Errorf("workgroup %q not found", wgName)
	}

	repos := resolveWorkgroupRepos(cfg, cfgPath, wg)
	reports := collectPushReports(ctx, repos, wg.Branch)
	writeWorkReports(cmd, reports)
	writeSummary(cmd, reports, "work push")

	return workReportsError(reports, "work push")
}

func resolveWorkgroupRepos(cfg *config.WorkspaceConfig, cfgPath string, wg config.WorkgroupConfig) []repo.Repo {
	return repo.FromNames(cfg, cfgPath, wg.Repos)
}

func collectPushReports(ctx context.Context, repos []repo.Repo, branchName string) []workReport {
	if len(repos) == 0 {
		return nil
	}

	workers := parallel.MaxWorkers(0, len(repos))

	return parallel.RunFanOut(repos, workers, func(r repo.Repo) workReport {
		return pushInRepo(ctx, r, branchName)
	})
}

func pushInRepo(ctx context.Context, r repo.Repo, branchName string) workReport {
	report := workReport{RepoName: r.Name}

	if !gitutil.HasRemote(ctx, r.AbsPath) {
		skipStep(&report, "push", "no remote")
		return report
	}

	runStep(&report, "push", func() error {
		return gitutil.PushBranchUpstream(ctx, r.AbsPath, "origin", branchName)
	})

	return report
}
