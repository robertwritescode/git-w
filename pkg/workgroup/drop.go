package workgroup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/spf13/cobra"
)

type dropOp struct {
	cfgPath      string
	wgName       string
	wg           config.WorkgroupConfig
	cfg          *config.WorkspaceConfig
	force        bool
	deleteBranch bool
}

func registerDrop(workCmd *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "drop <name>",
		Short: "Remove worktrees and local entry for a workgroup",
		Args:  cobra.ExactArgs(1),
		RunE:  runDrop,
	}

	cmd.Flags().Bool("force", false, "drop even if worktrees are dirty or have unpushed commits")
	cmd.Flags().Bool("delete-branch", false, "delete the workgroup branch after removing worktrees")

	workCmd.AddCommand(cmd)
}

func runDrop(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	op, err := prepareDropOp(cmd, args)
	if err != nil {
		return err
	}

	if err := executeDropOp(ctx, op); err != nil {
		return err
	}

	if err := config.RemoveLocalWorkgroup(op.cfgPath, op.wgName); err != nil {
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Dropped workgroup %q\n", op.wgName)
	return nil
}

func prepareDropOp(cmd *cobra.Command, args []string) (dropOp, error) {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return dropOp{}, err
	}

	wgName := args[0]
	wg, ok := cfg.Workgroups[wgName]
	if !ok {
		return dropOp{}, fmt.Errorf("workgroup %q not found", wgName)
	}

	force, _ := cmd.Flags().GetBool("force")
	deleteBranch, _ := cmd.Flags().GetBool("delete-branch")

	return dropOp{cfgPath: cfgPath, wgName: wgName, wg: wg, cfg: cfg, force: force, deleteBranch: deleteBranch}, nil
}

func executeDropOp(ctx context.Context, op dropOp) error {
	if err := validateDropSafety(ctx, op); err != nil {
		return err
	}

	return removeDropWorktrees(ctx, op)
}

func validateDropSafety(ctx context.Context, op dropOp) error {
	if op.force {
		return nil
	}

	violations, err := collectDropViolations(ctx, op)
	if err != nil {
		return err
	}

	if len(violations) == 0 {
		return nil
	}

	return fmt.Errorf("refusing to drop %q:\n%s", op.wgName, strings.Join(violations, "\n"))
}

func collectDropViolations(ctx context.Context, op dropOp) ([]string, error) {
	var violations []string

	for _, repoName := range op.wg.Repos {
		treePath := config.WorkgroupWorktreePath(op.cfgPath, op.wgName, repoName)
		if !pathExists(treePath) {
			continue
		}

		r := repo.Repo{Name: repoName, AbsPath: treePath}
		msgs, err := dropSafetyViolations(ctx, r)
		if err != nil {
			return nil, err
		}

		for _, msg := range msgs {
			violations = append(violations, fmt.Sprintf("- %s: %s", repoName, msg))
		}
	}

	return violations, nil
}

func dropSafetyViolations(ctx context.Context, r repo.Repo) ([]string, error) {
	return repo.SafetyViolations(ctx, r)
}

func removeDropWorktrees(ctx context.Context, op dropOp) error {
	for _, repoName := range op.wg.Repos {
		repoAbsPath, err := repoAbsPath(op.cfg, op.cfgPath, repoName)
		if err != nil {
			return err
		}

		treePath := config.WorkgroupWorktreePath(op.cfgPath, op.wgName, repoName)
		if err := removeOneWorktree(ctx, repoAbsPath, treePath, op.wg.Branch, op.force, op.deleteBranch); err != nil {
			return err
		}
	}

	// Remove the now-empty workgroup directory. os.Remove is a no-op if the
	// directory is non-empty (e.g. some repos were skipped).
	_ = os.Remove(workgroupRootPath(op.cfgPath, op.wgName))

	return nil
}

func removeOneWorktree(ctx context.Context, repoAbsPath, treePath, branchName string, force, deleteBranch bool) error {
	if _, err := os.Stat(treePath); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	var removeErr error
	if force {
		removeErr = gitutil.RemoveWorktreeForce(ctx, repoAbsPath, treePath)
	} else {
		removeErr = gitutil.RemoveWorktree(ctx, repoAbsPath, treePath)
	}

	if removeErr != nil {
		return removeErr
	}

	if deleteBranch {
		return gitutil.DeleteBranch(ctx, repoAbsPath, branchName)
	}

	return nil
}

func repoAbsPath(cfg *config.WorkspaceConfig, cfgPath, repoName string) (string, error) {
	rc, ok := cfg.Repos[repoName]
	if !ok {
		return "", fmt.Errorf("repo %q not found in config", repoName)
	}

	return config.ResolveRepoPath(cfgPath, rc.Path)
}
