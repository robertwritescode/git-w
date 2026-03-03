package worktree

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/spf13/cobra"
)

type addOperation struct {
	setName   string
	branch    string
	wt        config.WorktreeConfig
	branchAbs string
	relPath   string
}

func registerAdd(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:   "add <set-name> <branch> [path]",
		Short: "Add a branch worktree to an existing set",
		Args:  cobra.RangeArgs(2, 3),
		RunE:  runAdd,
	})
}

func runAdd(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	op, err := prepareAddOperation(cfg, cfgPath, args)
	if err != nil {
		return err
	}

	if err := executeAddOperation(cfgPath, &op); err != nil {
		return err
	}

	if err := finalizeAddOperation(cmd, cfg, cfgPath, op); err != nil {
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Added %q to set %q\n", op.branch, op.setName)
	return nil
}

func prepareAddOperation(cfg *config.WorkspaceConfig, cfgPath string, args []string) (addOperation, error) {
	setName, branch := args[0], args[1]
	wt, err := lookupAddWorktreeSet(cfg, setName, branch)
	if err != nil {
		return addOperation{}, err
	}

	branchAbs, err := resolveAddBranchPath(cfgPath, wt, args)
	if err != nil {
		return addOperation{}, err
	}

	return addOperation{setName: setName, branch: branch, wt: wt, branchAbs: branchAbs}, nil
}

func executeAddOperation(cfgPath string, op *addOperation) error {
	if err := materializeAddedWorktree(cfgPath, op.wt, op.branchAbs, op.branch); err != nil {
		return err
	}

	relPath, err := config.RelPath(cfgPath, op.branchAbs)
	if err != nil {
		return err
	}

	op.relPath = relPath
	return nil
}

func finalizeAddOperation(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath string, op addOperation) error {
	if err := persistAddedWorktree(cfg, cfgPath, op.setName, op.branch, op.wt, op.relPath); err != nil {
		return err
	}

	writeGitignoreWarning(cmd, cfgPath, op.relPath, cfg.AutoGitignoreEnabled())
	return nil
}

func lookupAddWorktreeSet(cfg *config.WorkspaceConfig, setName, branch string) (config.WorktreeConfig, error) {
	wt, exists := cfg.Worktrees[setName]
	if !exists {
		return config.WorktreeConfig{}, fmt.Errorf("worktree set %q not found", setName)
	}

	if _, exists := wt.Branches[branch]; exists {
		return config.WorktreeConfig{}, fmt.Errorf("branch %q is already registered in set %q", branch, setName)
	}

	return wt, nil
}

func resolveAddBranchPath(cfgPath string, wt config.WorktreeConfig, args []string) (string, error) {
	if len(args) == 3 {
		branchAbs, err := filepath.Abs(args[2])
		if err != nil {
			return "", fmt.Errorf("resolving path: %w", err)
		}
		return branchAbs, nil
	}

	return defaultBranchAbsPath(cfgPath, wt, args[1])
}

func materializeAddedWorktree(cfgPath string, wt config.WorktreeConfig, branchAbs, branch string) error {
	bareAbs, err := bareAbsPath(cfgPath, wt)
	if err != nil {
		return err
	}

	return gitutil.AddWorktree(context.Background(), bareAbs, branchAbs, branch)
}

func persistAddedWorktree(cfg *config.WorkspaceConfig, cfgPath, setName, branch string, wt config.WorktreeConfig, relPath string) error {
	wt.Branches[branch] = relPath
	cfg.Worktrees[setName] = wt
	return config.Save(cfgPath, cfg)
}
