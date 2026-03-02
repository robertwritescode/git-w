package worktree

import (
	"fmt"
	"strings"

	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/spf13/cobra"
)

type rmOperation struct {
	target    branchTarget
	wt        config.WorktreeConfig
	branchAbs string
	bareAbs   string
	force     bool
}

func registerRm(root *cobra.Command) {
	rmCmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove one worktree from a set",
		Args:  cobra.ExactArgs(1),
		RunE:  runRm,
	}
	rmCmd.Flags().Bool("force", false, "remove even if dirty or local-ahead")
	root.AddCommand(rmCmd)
}

func runRm(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	op, err := prepareRmOperation(cmd, cfg, cfgPath, args[0])
	if err != nil {
		return err
	}

	if err := executeRmOperation(op); err != nil {
		return err
	}

	if err := persistRm(cfg, cfgPath, op.target, op.wt); err != nil {
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Removed %q from set %q\n", op.target.RepoName, op.target.SetName)
	return nil
}

func prepareRmOperation(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath, name string) (rmOperation, error) {
	target, wt, err := findRmTarget(cfg, name)
	if err != nil {
		return rmOperation{}, err
	}

	force, _ := cmd.Flags().GetBool("force")
	branchAbs, err := config.ResolveRepoPath(cfgPath, target.RelPath)
	if err != nil {
		return rmOperation{}, err
	}

	bareAbs, err := bareAbsPath(cfgPath, wt)
	if err != nil {
		return rmOperation{}, err
	}

	return rmOperation{target: target, wt: wt, branchAbs: branchAbs, bareAbs: bareAbs, force: force}, nil
}

func executeRmOperation(op rmOperation) error {
	if err := validateRmSafety(op.force, op.target, op.branchAbs); err != nil {
		return err
	}

	return removeOneWorktree(op.bareAbs, op.branchAbs, op.force)
}

func findRmTarget(cfg *config.WorkspaceConfig, name string) (branchTarget, config.WorktreeConfig, error) {
	target, ok := findByRepoName(cfg, name)
	if !ok {
		return branchTarget{}, config.WorktreeConfig{}, fmt.Errorf("%q is not a worktree repo name", name)
	}

	wt := cfg.Worktrees[target.SetName]
	if len(wt.Branches) <= 1 {
		return branchTarget{}, config.WorktreeConfig{}, fmt.Errorf("cannot remove last worktree — use `git w worktree drop %s`", target.SetName)
	}

	return target, wt, nil
}

func validateRmSafety(force bool, target branchTarget, branchAbs string) error {
	if force {
		return nil
	}

	violations, err := safetyViolations(repo.Repo{Name: target.RepoName, AbsPath: branchAbs})
	if err != nil {
		return err
	}

	if len(violations) == 0 {
		return nil
	}

	return fmt.Errorf("refusing to remove %q: %s", target.RepoName, strings.Join(violations, "; "))
}

func persistRm(cfg *config.WorkspaceConfig, cfgPath string, target branchTarget, wt config.WorktreeConfig) error {
	cfg.RemoveRepoFromManualGroups(target.RepoName)
	delete(wt.Branches, target.Branch)
	cfg.Worktrees[target.SetName] = wt
	return config.Save(cfgPath, cfg)
}

func removeOneWorktree(bareAbs, branchAbs string, force bool) error {
	if force {
		return gitutil.RemoveWorktreeForce(bareAbs, branchAbs)
	}

	return gitutil.RemoveWorktree(bareAbs, branchAbs)
}
